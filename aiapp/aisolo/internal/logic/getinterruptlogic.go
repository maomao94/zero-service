package logic

import (
	"context"
	"errors"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"

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
	if in.GetInterruptId() == "" {
		return nil, errors.New("interrupt_id is required")
	}

	rec, err := l.svcCtx.Sessions.GetInterrupt(l.ctx, in.GetInterruptId())
	if err != nil {
		return nil, err
	}
	if in.GetUserId() != "" && rec.UserID != in.GetUserId() {
		return nil, errors.New("interrupt does not belong to current user")
	}

	return &aisolo.GetInterruptResp{Info: interruptToProto(rec)}, nil
}
