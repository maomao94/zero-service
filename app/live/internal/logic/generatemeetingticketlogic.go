package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateMeetingTicketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateMeetingTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateMeetingTicketLogic {
	return &GenerateMeetingTicketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 生成会议邀请票据
func (l *GenerateMeetingTicketLogic) GenerateMeetingTicket(in *live.GenerateMeetingTicketReq) (*live.GenerateMeetingTicketRes, error) {
	if in.MeetingNo == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号不能为空")
	}
	if in.Identity == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "绑定的身份不能为空")
	}

	// 验证会议是否存在
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == 3 { // 已结束
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
	}

	// 生成票据
	ticket := fmt.Sprintf("%s-%s", time.Now().Format("20060102"), uuid.New().String())
	
	// 计算过期时间
	expireSeconds := in.ExpireSeconds
	if expireSeconds == 0 {
		expireSeconds = 3600 // 默认 1 小时
	}
	expireTime := time.Now().Add(time.Duration(expireSeconds) * time.Second)

	// 存储到 Redis（包含会议号、identity、name、过期时间、权限）
	ticketDataBytes, _ := json.Marshal(map[string]interface{}{
		"meetingNo":         in.MeetingNo,
		"identity":          in.Identity,
		"name":              in.Name,
		"expireTime":        expireTime.Format("2006-01-02 15:04:05"),
		"canPublish":        in.CanPublish,
		"canSubscribe":      in.CanSubscribe,
		"canPublishData":    in.CanPublishData,
		"canPublishSources": in.CanPublishSources,
	})
	
	ticketKey := fmt.Sprintf("live:ticket:%s", ticket)
	err = l.svcCtx.Redis.SetexCtx(l.ctx, ticketKey, string(ticketDataBytes), int(expireSeconds))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "存储票据失败")
	}

	l.Logger.Infof("meeting ticket generated: meeting=%s identity=%s ticket=%s", in.MeetingNo, in.Identity, ticket)
	return &live.GenerateMeetingTicketRes{
		Ticket:            ticket,
		ExpireTime:        expireTime.Format("2006-01-02 15:04:05"),
		CanPublish:        in.CanPublish,
		CanSubscribe:      in.CanSubscribe,
		CanPublishData:    in.CanPublishData,
		CanPublishSources: in.CanPublishSources,
	}, nil
}
