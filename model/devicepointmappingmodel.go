package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ DevicePointMappingModel = (*customDevicePointMappingModel)(nil)

type (
	// DevicePointMappingModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDevicePointMappingModel.
	DevicePointMappingModel interface {
		devicePointMappingModel
		withSession(session sqlx.Session) DevicePointMappingModel
	}

	customDevicePointMappingModel struct {
		*defaultDevicePointMappingModel
		pointMappingCache *collection.Cache
	}
)

// NewDevicePointMappingModel returns a model for the database table.
func NewDevicePointMappingModel(conn sqlx.SqlConn) DevicePointMappingModel {
	pmc, _ := collection.NewCache(time.Hour*24, collection.WithName("pm-cache"))
	return &customDevicePointMappingModel{
		defaultDevicePointMappingModel: newDevicePointMappingModel(conn),
		pointMappingCache:              pmc,
	}
}

func (m *customDevicePointMappingModel) withSession(session sqlx.Session) DevicePointMappingModel {
	return NewDevicePointMappingModel(sqlx.NewSqlConnFromSession(session))
}

func (s *customDevicePointMappingModel) FindCacheOneByTagStationCoaIoa(ctx context.Context, tagStation string, coa int64, ioa int64) (*DevicePointMapping, bool, error) {
	key := fmt.Sprintf("pm:%s:%d:%d", tagStation, coa, ioa)
	val, err := s.pointMappingCache.Take(key, func() (any, error) {
		v, err := s.FindOneByTagStationCoaIoa(ctx, tagStation, coa, ioa)
		if err != nil {
			if err == ErrNotFound {
				return CacheEntry[DevicePointMapping]{}, nil
			}
			return nil, err
		}
		if v == nil {
			return CacheEntry[DevicePointMapping]{}, nil
		}
		return CacheEntry[DevicePointMapping]{
			Data:  *v,
			Valid: true,
		}, nil
	})
	if err != nil {
		return nil, false, err
	}
	entry := val.(CacheEntry[DevicePointMapping])
	if !entry.Valid {
		return nil, false, nil
	}
	pmValue := entry.Data
	return &pmValue, true, nil
}
