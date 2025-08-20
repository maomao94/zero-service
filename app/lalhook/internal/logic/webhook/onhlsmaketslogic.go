package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnHlsMakeTsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// HLS 生成每个 ts 分片文件时
func NewOnHlsMakeTsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnHlsMakeTsLogic {
	return &OnHlsMakeTsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnHlsMakeTsLogic) OnHlsMakeTs(req *types.OnHlsMakeTsRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
