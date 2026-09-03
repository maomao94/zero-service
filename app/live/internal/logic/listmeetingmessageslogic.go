package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMeetingMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMeetingMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMeetingMessagesLogic {
	return &ListMeetingMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询聊天记录
func (l *ListMeetingMessagesLogic) ListMeetingMessages(in *live.ListMeetingMessagesReq) (*live.ListMeetingMessagesRes, error) {
	if in.MeetingNo == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号不能为空")
	}

	// 查询消息
	messages, total, err := l.svcCtx.MeetingRepo.ListMessages(l.ctx, in.MeetingNo, in.Page, in.PageSize)
	if err != nil {
		l.Logger.Errorf("list messages failed: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询消息失败")
	}

	// 转换为响应格式
	messageInfos := make([]*live.MeetingMessageInfo, 0, len(messages))
	for _, msg := range messages {
		messageInfos = append(messageInfos, &live.MeetingMessageInfo{
			MessageId:   msg.MessageID,
			SenderId:    msg.SenderID,
			SenderName:  msg.SenderName,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			CreateTime:  carbonx.FormatDateTimeOrEmpty(msg.CreateTime),
		})
	}

	return &live.ListMeetingMessagesRes{
		Messages: messageInfos,
		Total:    total,
	}, nil
}
