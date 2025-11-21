package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DecodeH3Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecodeH3Logic(ctx context.Context, svcCtx *svc.ServiceContext) *DecodeH3Logic {
	return &DecodeH3Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 解码 h3
func (l *DecodeH3Logic) DecodeH3(in *geo.DecodeH3Req) (*geo.DecodeH3Res, error) {
	// todo: add your logic here and delete this line

	return &geo.DecodeH3Res{}, nil
}
