package svc

import (
	"fmt"
	"time"

	"zero-service/app/djicloud/internal/config"
	"zero-service/app/djicloud/internal/hooks"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

const dockOnlineTTL = 60 * time.Second

type ServiceContext struct {
	Config      config.Config
	DjiClient   *djisdk.Client
	DB          *gormx.DB
	OnlineCache *collection.Cache
}

func initDB(c config.Config) *gormx.DB {
	if c.DB.DataSource == "" {
		logx.Must(fmt.Errorf("djicloud db datasource is required"))
	}
	db := gormx.MustOpenWithConf(c.DB)
	if c.Mode == service.DevMode || c.Mode == service.TestMode {
		db.MustAutoMigrate(
			&gormmodel.DjiDevice{},
			&gormmodel.DjiDeviceTopo{},
			&gormmodel.DjiDeviceOsdSnapshot{},
			&gormmodel.DjiDeviceStateSnapshot{},
			&gormmodel.DjiHmsAlert{},
			&gormmodel.DjiFlightTaskProgress{},
			&gormmodel.DjiReturnHomeEvent{},
		)
	}
	return db
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	mqttCli := mqttx.MustNewClient(c.MqttConfig)
	djiCli := djisdk.NewClient(mqttCli,
		djisdk.WithPendingTTL(c.PendingTTL),
		djisdk.WithReplyOptions(djisdk.ReplyOptions{
			EnableEventReply:   c.UpstreamReply.EnableEventsReply,
			EnableStatusReply:  c.UpstreamReply.EnableStatusReply,
			EnableRequestReply: c.UpstreamReply.EnableRequestsReply,
		}),
	)

	onlineCache, err := collection.NewCache(dockOnlineTTL, collection.WithName("dock-online"))
	logx.Must(err)

	db := initDB(c)

	hooks.RegisterDjiClient(djiCli, hooks.RegisterDjiClientOptions{
		DB:          db,
		OnlineCache: onlineCache,
	})

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-cloud] subscribe topics failed: %v", err)
	}

	return &ServiceContext{
		Config:      c,
		DjiClient:   djiCli,
		DB:          db,
		OnlineCache: onlineCache,
	}
}
