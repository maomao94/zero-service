package svc

import (
	"zero-service/app/ispagent/internal/config"
	ctask "zero-service/app/ispagent/internal/crontask"
	ispclient "zero-service/app/ispagent/internal/isp"
	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/ftps"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
)

// ServiceContext 为 ispagent 的依赖注入容器，持有配置和 ISP TCP 客户端管理器。
type ServiceContext struct {
	Config        config.Config
	IspClient     *ispclient.Client
	Scheduler     *crontask.Scheduler
	ModelUploader *ftps.Uploader
	Store         crontask.TaskStore
	DB            *gormx.DB
}

// NewServiceContext 创建 ServiceContext。
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	var db *gormx.DB
	var store crontask.TaskStore
	if c.DB.DataSource != "" {
		db = gormx.MustOpenWithConf(c.DB)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			db.MustAutoMigrate(&gormmodel.GormTaskConfig{}, &gormmodel.GormIspPatrolTask{})
		}
		store = ctask.NewDBStore(db)
	}

	modelUploader := ftps.NewUploader(c.ModelSync.FTPS.ToFTPSConfig())
	m := ispclient.NewClient(c.IspSetting, store, db, modelUploader, nil,
		ispclient.WithReportOption(ispclient.WithNoFreshCheck(
			ispclient.ReportCategoryPatrolDeviceCoordinates,
			ispclient.ReportCategoryPatrolDeviceRunData,
			ispclient.ReportCategoryPatrolDeviceStatusData,
		)),
	)
	proc.AddShutdownListener(func() { m.Close() })

	svcCtx := &ServiceContext{
		Config:        c,
		IspClient:     m,
		ModelUploader: modelUploader,
		Store:         store,
		DB:            db,
	}

	if store != nil {
		handler := NewCronHandler(svcCtx)
		svcCtx.Scheduler = crontask.NewScheduler(store, handler,
			crontask.WithInterval(c.CronTask.Interval),
			crontask.WithLockExpire(c.CronTask.LockExpire),
			crontask.WithMaxDelay(c.CronTask.MaxDelay),
			crontask.WithInvalidTimeFilter(ctask.NewInvalidTimeFilter()),
		)
	}

	return svcCtx
}
