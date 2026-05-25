package logic

import (
	"context"
	"strings"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetInterruptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetInterruptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInterruptLogic {
	return &GetInterruptLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetInterrupt 获取中断详情 (前端刷新后回填 UI 使用)。
//
// 校验:
//  1. interrupt_id 必填
//  2. 记录必须存在
//  3. 如果附带 user_id, 必须与记录的 user_id 一致, 防止跨用户越权查看。
func (l *GetInterruptLogic) GetInterrupt(in *aisolo.GetInterruptReq) (*aisolo.GetInterruptResp, error) {
	if in == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "get interrupt request is required")
	}
	interruptID := strings.TrimSpace(in.GetInterruptId())
	if interruptID == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "interrupt_id is required")
	}

	rec, err := l.svcCtx.Sessions.GetInterrupt(l.ctx, interruptID)
	if err != nil {
		return nil, err
	}
	if userID := strings.TrimSpace(in.GetUserId()); userID != "" && rec.UserID != userID {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_FORBIDDEN, "interrupt does not belong to current user")
	}

	return &aisolo.GetInterruptResp{Info: interruptToProto(rec)}, nil
}
