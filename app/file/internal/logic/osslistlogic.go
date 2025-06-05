package logic

import (
	"context"
	"github.com/Masterminds/squirrel"
	"github.com/golang-module/carbon/v2"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
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
	whereBuilder := l.svcCtx.OssModel.SelectBuilder()
	if len(in.TenantId) > 0 {
		whereBuilder = whereBuilder.Where(squirrel.Eq{
			"tenant_id": in.TenantId,
		})
	}
	if in.Category > 0 {
		whereBuilder = whereBuilder.Where(squirrel.Eq{
			"category": in.Category,
		})
	}
	count, err := l.svcCtx.OssModel.FindCount(l.ctx, whereBuilder, "1")
	if err != nil {
		return nil, err
	}
	list, err := l.svcCtx.OssModel.FindPageListByPage(l.ctx, whereBuilder, in.Page, in.PageSize, in.OrderBy)
	if err != nil {
		return nil, err
	}
	var respOss []*file.Oss
	if len(list) > 0 {
		for _, oss := range list {
			var pbOss file.Oss
			_ = copier.Copy(&pbOss, oss)
			pbOss.CreateTime = carbon.CreateFromStdTime(oss.CreateTime).ToDateTimeString()
			pbOss.UpdateTime = carbon.CreateFromStdTime(oss.UpdateTime).ToDateTimeString()
			respOss = append(respOss, &pbOss)
		}
	}
	//copier.Copy(&respOss, list)
	return &file.OssListRes{
		Total: count,
		Oss:   respOss,
	}, nil
}
