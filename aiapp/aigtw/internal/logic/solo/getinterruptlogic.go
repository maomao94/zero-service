package solo

import (
	"context"
	"errors"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetInterruptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetInterruptLogic 获取中断详情 (页面刷新后回填 UI)。
// 直接透传到 aisolo gRPC, 由 aisolo 侧校验用户归属。
func NewGetInterruptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInterruptLogic {
	return &GetInterruptLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInterruptLogic) GetInterrupt(req *types.SoloGetInterruptRequest) (*types.SoloGetInterruptResponse, error) {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return nil, errors.New("missing user id in context")
	}
	if req == nil {
		return nil, errors.New("get interrupt request is required")
	}
	interruptID := strings.TrimSpace(req.InterruptId)
	if interruptID == "" {
		return nil, errors.New("interruptId is required")
	}
	resp, err := l.svcCtx.AiSoloCli.GetInterrupt(l.ctx, &aisolo.GetInterruptReq{
		InterruptId: interruptID,
		UserId:      userID,
	})
	if err != nil {
		return nil, err
	}
	return &types.SoloGetInterruptResponse{Info: interruptToType(resp.GetInfo())}, nil
}
