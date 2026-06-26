package svc

import (
	"context"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/gormx"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

const deviceOnlineTTL = 60 * time.Second

type DeviceOnlineRefreshCron struct {
	cron *cron.Cron
	db   *gormx.DB
}

func NewDeviceOnlineRefreshCron(db *gormx.DB) *DeviceOnlineRefreshCron {
	return &DeviceOnlineRefreshCron{
		cron: cron.New(cron.WithSeconds()),
		db:   db,
	}
}

func (c *DeviceOnlineRefreshCron) Start() {
	_, err := c.cron.AddFunc("*/15 * * * * *", func() {
		rowsAffected, err := c.refreshExpiredDevicesOnline(context.Background(), time.Now())
		if err != nil {
			return
		}
		logx.Infof("[dji-cloud] refresh expired devices online success, rows affected: %d", rowsAffected)
	})
	logx.Must(err)
	c.cron.Start()
}

func (c *DeviceOnlineRefreshCron) Stop() {
	c.cron.Stop()
}

func (c *DeviceOnlineRefreshCron) refreshExpiredDevicesOnline(ctx context.Context, now time.Time) (int64, error) {
	result := c.db.WithContext(gormx.WithoutSQLTrace(gormx.WithFullSQL(ctx))).Model(&gormmodel.DjiDevice{}).
		Where("is_online = ? AND last_online_at < ?", true, now.Add(-deviceOnlineTTL)).
		Updates(map[string]any{"is_online": false, "update_time": now})
	if result.Error != nil {
		logx.WithContext(ctx).Errorf("[dji-cloud] refresh expired devices online failed: %v", result.Error)
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
