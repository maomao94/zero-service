package logic

import (
	"context"
	"github.com/golang-module/carbon/v2"
	"github.com/jinzhu/copier"
	"zero-service/file/file"
	"zero-service/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type OssDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOssDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OssDetailLogic {
	return &OssDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *OssDetailLogic) OssDetail(in *file.OssDetailReq) (*file.OssDetailRes, error) {
	oss, err := l.svcCtx.OssModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}
	var respOss file.Oss
	_ = copier.Copy(&respOss, oss)
	respOss.CreateTime = carbon.CreateFromStdTime(oss.CreateTime).ToDateTimeString()
	respOss.UpdateTime = carbon.CreateFromStdTime(oss.UpdateTime).ToDateTimeString()
	return &file.OssDetailRes{Oss: &respOss}, nil
}
