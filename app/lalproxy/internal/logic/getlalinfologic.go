package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLalInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLalInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLalInfoLogic {
	return &GetLalInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询服务器基本信息
func (l *GetLalInfoLogic) GetLalInfo(in *lalproxy.GetLalInfoReq) (*lalproxy.GetLalInfoRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.GetLalInfoRes{}, nil
}
