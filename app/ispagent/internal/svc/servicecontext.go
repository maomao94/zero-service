package svc

import (
	"zero-service/app/ispagent/internal/config"
	ctask "zero-service/app/ispagent/internal/crontask"
	"zero-service/app/ispagent/internal/ispclient"
	"zero-service/common/crontask"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
)

// ServiceContext 为 ispagent 的依赖注入容器，持有配置和 ISP TCP 客户端管理器。
type ServiceContext struct {
	Config    config.Config
	IspClient *ispclient.Manager
	Scheduler *crontask.Scheduler
	Store     crontask.TaskStore
	DB        *gormx.DB
}

// NewServiceContext 创建 ServiceContext 并注册 ISP 客户端关闭回调。
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	var db *gormx.DB
	var store crontask.TaskStore
	if c.DB.DataSource != "" {
		db = gormx.MustOpenWithConf(c.DB)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			db.MustAutoMigrate(&ctask.GormTaskConfig{})
		}
		store = ctask.NewDBStore(db)
	}

	m := ispclient.NewManager(c.IspSetting, store)
	proc.AddShutdownListener(func() { m.Close() })

	svcCtx := &ServiceContext{
		Config:    c,
		IspClient: m,
		Store:     store,
		DB:        db,
	}

	if store != nil {
		handler := NewCronHandler(svcCtx)
		svcCtx.Scheduler = crontask.NewScheduler(store, handler,
			crontask.WithInterval(c.CronTask.Interval),
			crontask.WithLockExpire(c.CronTask.LockExpire),
		)
		svcCtx.Scheduler.Start()
		proc.AddShutdownListener(func() { svcCtx.Scheduler.Stop() })
		logx.Info("[ispagent] cron scheduler started")
	}

	return svcCtx
}
