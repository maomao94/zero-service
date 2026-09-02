package live

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMeetingMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMeetingMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingMessagesLogic {
	return &ListMeetingMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMeetingMessagesLogic) ListMeetingMessages(req *types.ListMeetingMessagesRequest) (resp *types.ListMeetingMessagesReply, err error) {
	r, err := l.svcCtx.LiveRpcCli.ListMeetingMessages(l.ctx, &live.ListMeetingMessagesReq{
		MeetingNo: req.MeetingNo,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	messages := make([]types.MeetingMessageInfo, 0, len(r.GetMessages()))
	for _, m := range r.GetMessages() {
		messages = append(messages, types.MeetingMessageInfo{
			MessageId:   m.GetMessageId(),
			SenderId:    m.GetSenderId(),
			SenderName:  m.GetSenderName(),
			Content:     m.GetContent(),
			MessageType: m.GetMessageType(),
			CreateTime:  m.GetCreateTime(),
		})
	}
	return &types.ListMeetingMessagesReply{
		Messages: messages,
		Total:    r.GetTotal(),
	}, nil
}
