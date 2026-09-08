// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ticket

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type JoinMeetingByTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据票据加入会议（无需JWT）
func NewJoinMeetingByTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMeetingByTicketLogic {
	return &JoinMeetingByTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JoinMeetingByTicketLogic) JoinMeetingByTicket(req *types.JoinMeetingByTicketRequest) (resp *types.JoinMeetingByTicketReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.JoinMeetingByTicket(l.ctx, &live.JoinMeetingByTicketReq{
		Ticket: req.Ticket,
	})
	if err != nil {
		return nil, err
	}

	resp = &types.JoinMeetingByTicketReply{
		Token:             r.GetToken(),
		CanPublish:        r.GetCanPublish(),
		CanSubscribe:      r.GetCanSubscribe(),
		CanPublishData:    r.GetCanPublishData(),
		CanPublishSources: r.GetCanPublishSources(),
	}
	if m := r.GetMeeting(); m != nil {
		resp.Meeting = types.MeetingInfo{
			MeetingNo:  m.GetMeetingNo(),
			Title:      m.GetTitle(),
			Status:     m.GetStatus(),
			CreateUser: m.GetCreateUser(),
			UpdateUser: m.GetUpdateUser(),
			DeptCode:   m.GetDeptCode(),
			StartTime:  m.GetStartTime(),
			EndTime:    m.GetEndTime(),
			CreateTime: m.GetCreateTime(),
		}
	}
	return resp, nil
}
