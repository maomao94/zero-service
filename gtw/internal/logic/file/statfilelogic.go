package file

import (
	"context"
	"github.com/jinzhu/copier"
	"zero-service/app/file/file"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StatFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取文件信息
func NewStatFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatFileLogic {
	return &StatFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StatFileLogic) StatFile(req *types.StatFileRequest) (resp *types.StatFileReply, err error) {
	statFileRes, err := l.svcCtx.FileRpcCLi.StatFile(l.ctx, &file.StatFileReq{
		TenantId:   req.TenantId,
		Code:       req.Code,
		BucketName: req.BucketName,
		Filename:   req.Filename,
		IsSign:     req.IsSign,
		Expires:    req.Expires,
	})
	if err != nil {
		return nil, err
	}
	var ossFile types.OssFile
	_ = copier.Copy(&ossFile, statFileRes.OssFile)
	return &types.StatFileReply{OssFile: ossFile}, nil
}
