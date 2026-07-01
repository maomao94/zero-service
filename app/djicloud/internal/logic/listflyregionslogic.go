package logic

import (
	"context"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type ListFlyRegionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFlyRegionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFlyRegionsLogic {
	return &ListFlyRegionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListFlyRegions 分页查询飞行区配置记录。
// sign_url 为 true 时并发生成 OSS 签名下载地址。
func (l *ListFlyRegionsLogic) ListFlyRegions(in *djicloud.ListFlyRegionsReq) (*djicloud.ListFlyRegionsRes, error) {
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiFlyRegion{})
	if in.GetGatewaySn() != "" {
		db = db.Where("gateway_sn = ?", in.GetGatewaySn())
	}

	var regions []gormmodel.DjiFlyRegion
	pageResult, err := gormx.QueryPage(db.Order("id DESC"), int(in.GetPage()), int(in.GetPageSize()), &regions)
	if err != nil {
		return nil, err
	}

	needSign := in.GetSignUrl() && l.svcCtx.OssTemplate != nil

	list := make([]*djicloud.FlyRegionInfo, len(regions))
	fns := make([]func() error, len(regions))
	for i := range regions {
		i := i
		r := regions[i]
		fns[i] = func() error {
			info := &djicloud.FlyRegionInfo{
				Id:         r.Id,
				GatewaySn:  r.GatewaySn,
				Name:       r.Name,
				FileId:     r.FileId,
				FileName:   r.FileName,
				FileSize:   r.FileSize,
				Checksum:   r.Checksum,
				CreateTime: r.CreateTime.UnixMilli(),
			}
			if needSign && r.BucketName != "" {
				u, err := l.svcCtx.OssTemplate.SignUrl(l.ctx, "", r.BucketName, r.FileName, 7*24*time.Hour)
				if err != nil {
					logx.WithContext(l.ctx).Errorf("[dji-cloud] ListFlyRegions: sign url failed: %v", err)
				} else {
					info.Url = u
				}
			}
			list[i] = info
			return nil
		}
	}
	err = mr.Finish(fns...)
	if err != nil {
		return nil, err
	}

	return &djicloud.ListFlyRegionsRes{
		Total: int64(pageResult.Total),
		List:  list,
	}, nil
}
