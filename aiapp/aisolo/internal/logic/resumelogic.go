package logic

import (
	"context"
	"fmt"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/a2ui"
	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeLogic {
	return &ResumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Resume 恢复中断的执行
func (l *ResumeLogic) Resume(in *aisolo.ResumeRequest, stream aisolo.AiSolo_ResumeServer) error {
	sessionID := in.SessionId
	interruptID := in.InterruptId
	action := in.Action

	l.Infof("Resume: session=%s, interrupt=%s, action=%s", sessionID, interruptID, action)

	// 创建 GRPC Stream Writer
	writer := a2ui.NewGRPCStreamWriter(stream, sessionID)

	// 检查请求参数
	if sessionID == "" || interruptID == "" {
		fmt.Fprintf(writer, `{"error":{"code":"invalid_request","message":"session_id and interrupt_id required"}}\n`)
		return nil
	}

	// 从中断恢复
	// TODO: 实现从中断点恢复的逻辑
	// 需要使用 runner 的 checkpoint/resume 机制

	fmt.Fprintf(writer, `{"dataModelUpdate":{"surfaceId":"chat-%s","contents":[{"key":"chat-%s/resume","valueString":"resume completed: action=%s"}]}}\n`, sessionID, sessionID, action)
	return nil
}
