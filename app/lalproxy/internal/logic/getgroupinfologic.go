package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询特定流分组的信息
func (l *GetGroupInfoLogic) GetGroupInfo(in *lalproxy.GetGroupInfoReq) (*lalproxy.GetGroupInfoRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.GetGroupInfoRes{}, nil
}
