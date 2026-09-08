package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/livekitx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type JoinMeetingByTicketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewJoinMeetingByTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMeetingByTicketLogic {
	return &JoinMeetingByTicketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

type ticketData struct {
	MeetingNo         string   `json:"meetingNo"`
	Identity          string   `json:"identity"`
	Name              string   `json:"name"`
	ExpireTime        string   `json:"expireTime"`
	CanPublish        bool     `json:"canPublish"`
	CanSubscribe      bool     `json:"canSubscribe"`
	CanPublishData    bool     `json:"canPublishData"`
	CanPublishSources []string `json:"canPublishSources"`
	TicketType        int32    `json:"ticketType"`
}

// 根据票据加入会议
func (l *JoinMeetingByTicketLogic) JoinMeetingByTicket(in *live.JoinMeetingByTicketReq) (*live.JoinMeetingByTicketRes, error) {
	if in.Ticket == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "票据不能为空")
	}

	// 从 Redis 获取票据
	ticketKey := fmt.Sprintf("live:ticket:%s", in.Ticket)
	ticketJson, err := l.svcCtx.Redis.GetCtx(l.ctx, ticketKey)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_CACHE, "获取票据失败")
	}
	if ticketJson == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "票据不存在或已过期")
	}

	// 解析票据数据
	var data ticketData
	if err := json.Unmarshal([]byte(ticketJson), &data); err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "票据数据格式错误")
	}

	// 校验票据是否过期（expireTime 格式：yyyy-MM-dd HH:mm:ss）
	if data.ExpireTime != "" {
		if expireAt := carbon.Parse(data.ExpireTime); expireAt.IsValid() && expireAt.Lt(carbon.Now()) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "票据已过期")
		}
	}

	// 分布式锁防并发使用票据加入同一会议
	lockKey := redisMeetingLockPrefix + data.MeetingNo
	lock := redis.NewRedisLock(l.svcCtx.Redis, lockKey)
	lock.SetExpire(meetingLockTTL)
	ok, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "获取会议锁失败")
	}
	if !ok {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, "会议正在被操作，请稍后重试")
	}
	defer lock.Release()

	// 二次检查票据是否仍然有效（锁等待期间可能已被使用）
	ticketJson, err = l.svcCtx.Redis.GetCtx(l.ctx, ticketKey)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_CACHE, "获取票据失败")
	}
	if ticketJson == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "票据不存在或已过期")
	}

	// 根据票据类型处理：一次性票据删除，有效期票据保留
	if data.TicketType == 1 {
		// 一次性票据：删除 individual key
		l.svcCtx.Redis.DelCtx(l.ctx, ticketKey)
	}

	// 验证会议是否存在
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, data.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
	}

	// 校验参会人是否已在会议中
	if l.svcCtx.MeetingRepo.IsParticipantInMeeting(l.ctx, data.MeetingNo, data.Identity) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "您已在此会议中")
	}

	// 生成 LiveKit token（使用票据中绑定的 identity 和权限）
	token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
		APIKey:             l.svcCtx.Config.LiveKit.ApiKey,
		APISecret:          l.svcCtx.Config.LiveKit.ApiSecret,
		Room:               data.MeetingNo,
		Identity:           data.Identity,
		Name:               data.Name,
		ValidFor:           l.svcCtx.Config.LiveKit.TokenValidFor,
		CanPublish:         data.CanPublish,
		CanSubscribe:       data.CanSubscribe,
		CanPublishData:     data.CanPublishData,
		CanPublishSources:  data.CanPublishSources,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "生成入会 token 失败")
	}

	// Upsert 参会人记录（使用票据中绑定的 identity 和 name）
	now := carbonx.NowStartOfSecond().StdTime()
	participant := &gormmodel.LiveMeetingParticipant{
		MeetingNo: data.MeetingNo,
		Identity:  data.Identity,
		Name:      data.Name,
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  now,
	}
	if err := l.svcCtx.MeetingRepo.UpsertParticipant(l.ctx, participant); err != nil {
		l.Logger.Errorf("upsert participant failed: %v", err)
	}

	l.Logger.Infof("join by ticket: meeting=%s identity=%s", data.MeetingNo, data.Identity)
	return &live.JoinMeetingByTicketRes{
		Token:             token,
		Meeting:           toMeetingInfo(meeting),
		CanPublish:        data.CanPublish,
		CanSubscribe:      data.CanSubscribe,
		CanPublishData:    data.CanPublishData,
		CanPublishSources: data.CanPublishSources,
	}, nil
}
