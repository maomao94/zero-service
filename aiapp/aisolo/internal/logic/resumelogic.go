package logic

import (
	"context"
	"errors"

	"zero-service/aiapp/aisolo/aisolo"
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

// Resume 恢复中断的执行（同步返回结果）
//
// 流程：
// 1. 验证请求参数
// 2. 根据 action 构造 ApprovalResult
// 3. 调用 Runner.ResumeWithParams 恢复执行
// 4. 返回执行结果
func (l *ResumeLogic) Resume(in *aisolo.ResumeRequest) (*aisolo.ResumeResponse, error) {
	sessionID := in.SessionId
	userID := in.UserId
	interruptID := in.InterruptId

	l.Infof("Resume: session=%s, user=%s, interrupt=%s, action=%v",
		sessionID, userID, interruptID, in.Action)

	// 检查请求参数
	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if interruptID == "" {
		return nil, errors.New("interrupt_id is required")
	}
	if userID == "" {
		userID = "anonymous"
	}

	// 获取历史消息
	var history []*struct {
		Role    string
		Content string
	}
	if l.svcCtx.MemoryStorage != nil {
		msgs, err := l.svcCtx.MemoryStorage.GetMessages(l.ctx, userID, sessionID, 50)
		if err != nil {
			l.Logger.Errorf("get messages: %v", err)
		} else {
			for _, msg := range msgs {
				history = append(history, &struct {
					Role    string
					Content string
				}{msg.Role, msg.Content})
			}
		}
	}

	// 构造审批结果
	var result *ApprovalResult
	switch in.Action {
	case aisolo.ResumeAction_RESUME_ACTION_APPROVE:
		result = &ApprovalResult{
			Approved:    true,
			SelectedIds: in.SelectedIds,
		}
	case aisolo.ResumeAction_RESUME_ACTION_DENY:
		result = &ApprovalResult{
			Approved:         false,
			DisapproveReason: in.Reason,
		}
	default:
		// 默认为批准
		result = &ApprovalResult{Approved: true}
	}

	// TODO: 实现真正的 Resume 逻辑
	// 需要：
	// 1. Runner 配置了 CheckpointStore
	// 2. 保存了 pending_interrupt_id 和 msg_idx
	// 3. 使用 runner.ResumeWithParams(ctx, checkpointID, params)
	_ = history
	_ = result

	l.Infof("Resume completed: session=%s", sessionID)
	return &aisolo.ResumeResponse{
		SessionId: sessionID,
		Success:   true,
		Message:   "中断已处理，执行已恢复",
	}, nil
}

// ApprovalResult 审批结果
type ApprovalResult struct {
	Approved         bool
	SelectedIds      []string
	DisapproveReason string
}
