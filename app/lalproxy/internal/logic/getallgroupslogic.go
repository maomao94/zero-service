package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllGroupsLogic {
	return &GetAllGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询所有流分组的信息
func (l *GetAllGroupsLogic) GetAllGroups(in *lalproxy.GetAllGroupsReq) (*lalproxy.GetAllGroupsRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.GetAllGroupsRes{}, nil
}
