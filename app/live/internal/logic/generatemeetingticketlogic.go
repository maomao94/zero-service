package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
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
	// 校验：meeting_no 和 meeting_code 至少提供一个
	if strings.TrimSpace(in.MeetingNo) == "" && strings.TrimSpace(in.MeetingCode) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号或会议码不能同时为空")
	}
	if strings.TrimSpace(in.Identity) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "参会人身份不能为空")
	}

	// 查找会议
	var meeting *gormmodel.LiveMeeting
	var err error
	if strings.TrimSpace(in.MeetingCode) != "" {
		meeting, err = l.svcCtx.MeetingRepo.GetMeetingByCode(l.ctx, in.MeetingCode)
	} else {
		meeting, err = l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	}
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
	}

	// 生成票据
	uuidStr, err := l.svcCtx.IdUtil.SimpleUUID()
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "生成票据失败")
	}
	ticket := fmt.Sprintf("%s-%s", carbon.Now().Format("Ymd"), uuidStr)

	// 计算过期时间
	expireSeconds := in.ExpireSeconds
	if expireSeconds == 0 {
		expireSeconds = 3600 // 默认 1 小时
	}
	expireTime := carbon.Now().AddSeconds(int(expireSeconds))

	// 票据类型：1-一次性（默认），2-有效期
	ticketType := in.TicketType
	if ticketType == 0 {
		ticketType = 1
	}

	// 存储到 Redis（包含会议号、identity、name、过期时间、权限、票据类型）
	ticketDataBytes, _ := json.Marshal(map[string]interface{}{
		"meetingNo":         meeting.MeetingNo,
		"identity":          in.Identity,
		"name":              in.Name,
		"expireTime":        expireTime.ToDateTimeString(),
		"canPublish":        in.CanPublish,
		"canSubscribe":      in.CanSubscribe,
		"canPublishData":    in.CanPublishData,
		"canPublishSources": in.CanPublishSources,
		"ticketType":        ticketType,
	})

	ticketKey := fmt.Sprintf("live:ticket:%s", ticket)
	err = l.svcCtx.Redis.SetexCtx(l.ctx, ticketKey, string(ticketDataBytes), int(expireSeconds))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "存储票据失败")
	}

	l.Logger.Infof("meeting ticket generated: meeting=%s identity=%s ticket=%s type=%d", meeting.MeetingNo, in.Identity, ticket, ticketType)
	return &live.GenerateMeetingTicketRes{
		Ticket:            ticket,
		ExpireTime:        expireTime.ToDateTimeString(),
		CanPublish:        in.CanPublish,
		CanSubscribe:      in.CanSubscribe,
		CanPublishData:    in.CanPublishData,
		CanPublishSources: in.CanPublishSources,
	}, nil
}
