package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReportMeetingMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReportMeetingMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportMeetingMessageLogic {
	return &ReportMeetingMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReportMeetingMessageLogic) ReportMeetingMessage(req *types.ReportMeetingMessageRequest) (*types.ReportMeetingMessageReply, error) {
	resp, err := l.svcCtx.LiveRpcCli.ReportMeetingMessage(l.ctx, &live.ReportMeetingMessageReq{
		MeetingNo:   req.MeetingNo,
		Content:     req.Content,
		MessageType: req.MessageType,
	})
	if err != nil {
		return nil, err
	}
	return &types.ReportMeetingMessageReply{
		MessageId: resp.MessageId,
	}, nil
}
