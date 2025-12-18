package svc

import (
	"context"
	"fmt"
	"time"
	"zero-service/common/dbx"
	"zero-service/facade/streamevent/internal/config"
	"zero-service/model"

	_ "github.com/taosdata/driver-go/v3/taosWS"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var emptyDevicePointMapping = &model.DevicePointMapping{}

type ServiceContext struct {
	Config                  config.Config
	TaosConn                sqlx.SqlConn
	SqliteConn              sqlx.SqlConn
	DevicePointMappingModel model.DevicePointMappingModel
	pointMappingCache       *collection.Cache
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config: c,
	}
	if len(c.TaosDSN) > 0 {
		svcCtx.TaosConn = dbx.NewTaos(c.TaosDSN)
	}
	if len(c.Sqlite) > 0 {
		svcCtx.SqliteConn = dbx.NewSqlite(c.Sqlite)
		svcCtx.DevicePointMappingModel = model.NewDevicePointMappingModel(svcCtx.SqliteConn)
	}
	svcCtx.pointMappingCache, _ = collection.NewCache(time.Hour*24, collection.WithName("pm-cache"))
	return svcCtx
}

func (s *ServiceContext) FindOneByTagStationCoaIoa(ctx context.Context, tagStation string, coa int64, ioa int64) (pm *model.DevicePointMapping, valid bool, err error) {
	key := fmt.Sprintf("pm:%s:%d:%d", tagStation, coa, ioa)
	val, err := s.pointMappingCache.Take(key, func() (any, error) {
		v, err := s.DevicePointMappingModel.
			FindOneByTagStationCoaIoa(ctx, tagStation, coa, ioa)
		if err != nil {
			if err == model.ErrNotFound {
				return model.CacheEntry[model.DevicePointMapping]{}, nil
			}
			return nil, err
		}
		if v == nil {
			return model.CacheEntry[model.DevicePointMapping]{}, nil
		}
		return model.CacheEntry[model.DevicePointMapping]{
			Data:  *v,
			Valid: true,
		}, nil
	})
	if err != nil {
		return nil, false, err
	}
	entry := val.(model.CacheEntry[model.DevicePointMapping])
	if !entry.Valid {
		return nil, false, nil
	}
	pmValue := entry.Data
	return &pmValue, true, nil
}
