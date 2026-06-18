package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFenceLogic {
	return &DeleteFenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteFence 删除电子围栏及其关联的空间索引数据。
func (l *DeleteFenceLogic) DeleteFence(in *gis.DeleteFenceReq) (*gis.DeleteFenceRes, error) {
	if in.FenceId == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "fenceId")
	}

	if err := l.svcCtx.FenceStore.RemoveFence(l.ctx, in.FenceId); err != nil {
		l.Logger.Errorf("删除围栏失败, fenceId=%s, err=%v", in.FenceId, err)
		return nil, err
	}

	return &gis.DeleteFenceRes{}, nil
}
