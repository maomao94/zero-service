package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"os"
	"zero-service/model"
	"zero-service/ossx"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PutFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPutFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PutFileLogic {
	return &PutFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PutFileLogic) PutFile(in *file.PutFileReq) (*file.PutFileRes, error) {
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	f, err := os.Open(in.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	ossFile, err := ossTemplate.PutObject(in.TenantId, in.BucketName, in.Filename, in.ContentType, f, fInfo.Size())
	if err != nil {
		return nil, err
	}
	var pbFile file.File
	_ = copier.Copy(&pbFile, ossFile)
	return &file.PutFileRes{
		File: &pbFile,
	}, nil
}
