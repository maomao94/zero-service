package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type OssListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOssListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OssListLogic {
	return &OssListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *OssListLogic) OssList(in *file.OssListReq) (*file.OssListRes, error) {
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.Oss{})
	if len(in.TenantId) > 0 {
		db = db.Where("tenant_id = ?", in.TenantId)
	}
	if in.Category > 0 {
		db = db.Where("category = ?", in.Category)
	}
	db = db.Order(ossOrderBy(in.OrderBy))

	var list []gormmodel.Oss
	pageResult, err := gormx.QueryPage(db, int(in.Page), int(in.PageSize), &list)
	if err != nil {
		return nil, err
	}
	respOss := make([]*file.Oss, 0, len(list))
	for i := range list {
		respOss = append(respOss, toPbOss(&list[i]))
	}
	return &file.OssListRes{
		Total: pageResult.Total,
		Oss:   respOss,
	}, nil
}
