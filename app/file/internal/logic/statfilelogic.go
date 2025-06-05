package logic

import (
	"context"
	"github.com/golang-module/carbon/v2"
	"github.com/jinzhu/copier"
	"time"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model"
	"zero-service/ossx"

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
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	ossFile, err := ossTemplate.StatFile(l.ctx, in.TenantId, in.BucketName, in.Filename)
	if err != nil {
		return nil, err
	}
	var resOssFile file.OssFile
	_ = copier.Copy(&resOssFile, ossFile)
	resOssFile.PutTime = carbon.CreateFromStdTime(ossFile.PutTime).ToDateTimeString()
	//l.Infof("time %s", time.Unix(ossFile.PutTime.Unix(), 0).Format("2006-01-02 15:04:05"))
	if in.IsSign {
		expires := 60 * time.Minute // 1 小时
		if in.Expires > 0 {
			expires = time.Duration(in.Expires) * time.Minute
		}
		signUrl, err := ossTemplate.SignUrl(l.ctx, in.TenantId, in.BucketName, in.Filename, expires)
		if err != nil {
			return nil, err
		}
		resOssFile.SignUrl = signUrl
	}
	return &file.StatFileRes{
		OssFile: &resOssFile,
	}, nil
}
