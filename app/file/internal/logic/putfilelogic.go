package logic

import (
	"context"
	"os"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/imagex"
	"zero-service/common/ossx"

	"github.com/jinzhu/copier"
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
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
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

	head, reader, err := ossx.ReadUploadHead(f)
	if err != nil {
		l.Logger.Errorf("Failed to read file head: %v", err)
		return nil, err
	}
	contentType := resolveContentType(l.ctx, "", head)

	ossFile, err := ossTemplate.PutObject(l.ctx, in.TenantId, in.BucketName, in.Filename, contentType, reader, fInfo.Size(), in.PathPrefix)
	if err != nil {
		return nil, err
	}
	var pbFile file.File
	copier.Copy(&pbFile, ossFile) // nolint:errcheck

	if isImageContentType(contentType) {
		if exifMeta, err := imagex.ExtractImageMeta(in.Path); err == nil {
			var meta file.ImageMeta
			copier.Copy(&meta, &exifMeta) // nolint:errcheck
			pbFile.Meta = &meta
		}
	}
	return &file.PutFileRes{
		File: &pbFile,
	}, nil
}
