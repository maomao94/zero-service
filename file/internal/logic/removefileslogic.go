package logic

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveFilesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFilesLogic {
	return &RemoveFilesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveFilesLogic) RemoveFiles(in *file.RemoveFilesReq) (*file.RemoveFileRes, error) {
	// todo: add your logic here and delete this line

	return &file.RemoveFileRes{}, nil
}
