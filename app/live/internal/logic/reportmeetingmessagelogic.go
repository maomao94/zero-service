package logic

import (
	"context"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReportMeetingMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReportMeetingMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportMeetingMessageLogic {
	return &ReportMeetingMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 上报聊天消息
func (l *ReportMeetingMessageLogic) ReportMeetingMessage(in *live.ReportMeetingMessageReq) (*live.ReportMeetingMessageRes, error) {
	if in.MeetingNo == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号不能为空")
	}
	if in.Content == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "消息内容不能为空")
	}

	// 生成 messageId（如果前端未提供）
	messageID := in.MessageId
	if messageID == "" {
		messageID = uuid.New().String()
	}

	// 获取发送者信息
	senderID := authctx.GetUserId(l.ctx)
	senderName := authctx.GetUserName(l.ctx)

	// 消息类型默认为 text
	messageType := in.MessageType
	if messageType == "" {
		messageType = "text"
	}

	// 存储消息
	message := &gormmodel.LiveMeetingMessage{
		MeetingNo:   in.MeetingNo,
		MessageID:   messageID,
		SenderID:    senderID,
		SenderName:  senderName,
		Content:     in.Content,
		MessageType: messageType,
		CreateTime:  time.Now(),
	}
	if err := l.svcCtx.MeetingRepo.CreateMessage(l.ctx, message); err != nil {
		l.Logger.Errorf("create message failed: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "存储消息失败")
	}

	l.Logger.Infof("message reported: meeting=%s messageId=%s sender=%s", in.MeetingNo, messageID, senderID)
	return &live.ReportMeetingMessageRes{}, nil
}
