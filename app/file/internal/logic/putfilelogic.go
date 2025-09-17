package logic

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/imagex"
	"zero-service/common/ossx"
	"zero-service/model"

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
	// 读取文件的前 512 字节
	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		l.Logger.Errorf("Failed to read file: %v", err)
		return nil, err
	}
	// 检测内容类型
	contentType := http.DetectContentType(buffer)
	// 重置文件指针，继续从文件开始读取
	_, err = f.Seek(0, io.SeekStart)
	ossFile, err := ossTemplate.PutObject(context.Background(), in.TenantId, in.BucketName, in.Filename, contentType, f, fInfo.Size())
	if err != nil {
		return nil, err
	}
	var pbFile file.File
	_ = copier.Copy(&pbFile, ossFile)
	meta := file.ImageMeta{}
	if strings.HasPrefix(contentType, "image/") {
		exifMeta, err := imagex.ExtractImageMeta(in.Path)
		if err == nil {
			_ = copier.Copy(&meta, &exifMeta)
			pbFile.Meta = &meta
		}
	}
	return &file.PutFileRes{
		File: &pbFile,
	}, nil
}
