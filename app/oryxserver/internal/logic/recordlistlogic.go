package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/gormx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type RecordListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordListLogic {
	return &RecordListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询录制记录（平台级接口）
func (l *RecordListLogic) RecordList(in *oryxserver.RecordListReq) (*oryxserver.RecordListRes, error) {
	page, pageSize := in.Page, in.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.Record{})
	if in.Uuid != "" {
		db = db.Where("uuid = ?", in.Uuid)
	}
	if in.Vhost != "" {
		db = db.Where("vhost = ?", in.Vhost)
	}
	if in.App != "" {
		db = db.Where("app = ?", in.App)
	}
	if in.Stream != "" {
		db = db.Where("stream LIKE ?", "%"+in.Stream+"%")
	}
	if len(in.Status) > 0 {
		db = db.Where("status IN ?", in.Status)
	}
	if in.BeginTimeStart != "" {
		c := carbon.Parse(in.BeginTimeStart)
		if c.IsValid() {
			db = db.Where("begin_time >= ?", c.Time)
		}
	}
	if in.BeginTimeEnd != "" {
		c := carbon.Parse(in.BeginTimeEnd)
		if c.IsValid() {
			db = db.Where("begin_time <= ?", c.Time)
		}
	}
	if in.EndTimeStart != "" {
		c := carbon.Parse(in.EndTimeStart)
		if c.IsValid() {
			db = db.Where("end_time >= ?", c.Time)
		}
	}
	if in.EndTimeEnd != "" {
		c := carbon.Parse(in.EndTimeEnd)
		if c.IsValid() {
			db = db.Where("end_time <= ?", c.Time)
		}
	}

	var records []gormmodel.Record
	res, err := gormx.QueryPage[gormmodel.Record](db.Order("begin_time DESC"), page, pageSize, &records)
	if err != nil {
		l.Logger.Errorf("查询录制记录失败: %v", err)
		return nil, err
	}

	items := make([]*oryxserver.RecordItem, 0, len(records))
	for _, r := range records {
		items = append(items, &oryxserver.RecordItem{
			Uuid:         r.UUID,
			Vhost:        r.Vhost,
			App:          r.App,
			Stream:       r.Stream,
			Opaque:       r.Opaque,
			Status:       r.Status,
			BeginTime:    r.BeginTime.UnixMilli(),
			EndTime:      r.EndTime.UnixMilli(),
			ArtifactCode: r.ArtifactCode,
			ArtifactPath: r.ArtifactPath,
			ArtifactUrl:  r.ArtifactURL,
		})
	}

	return &oryxserver.RecordListRes{
		Records:  items,
		Total:    res.Total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
