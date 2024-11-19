package logic

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StatFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStatFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatFileLogic {
	return &StatFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StatFileLogic) StatFile(in *file.StatFileReq) (*file.StatFileRes, error) {
	// todo: add your logic here and delete this line

	return &file.StatFileRes{}, nil
}
