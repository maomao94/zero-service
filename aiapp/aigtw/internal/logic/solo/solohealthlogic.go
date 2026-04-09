package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SoloHealthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSoloHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SoloHealthLogic {
	return &SoloHealthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SoloHealthLogic) SoloHealth() (resp *types.SoloHealthResp, err error) {
	// Aisolo 服务健康检查
	// 直接返回健康状态，gRPC 连接由框架管理
	return &types.SoloHealthResp{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}
