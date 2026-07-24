package ispclient

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"zero-service/app/ispagent/internal/handler"
	"zero-service/common/crontask"
	"zero-service/common/ftps"
	"zero-service/common/gnetx"
	"zero-service/common/gormx"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const reportTaskConcurrency = 4

// TaskRunFunc 立即执行指定任务。
type TaskRunFunc func(ctx context.Context, taskCode string) error

// IspClient 组合 common/isp.Client 的 TCP 通信能力，并注册 ispagent 私有业务 handler。
//
// 建连、注册、心跳、请求序号和通用应答包装由 common/isp.Client 负责；本类型只保留任务、模型、机器人控制和上报缓存等服务私有逻辑。
type IspClient struct {
	*isp.Client

	taskStore     crontask.TaskStore
	taskRun       atomic.Pointer[TaskRunFunc]
	db            *gormx.DB
	modelUploader *ftps.Uploader
	modelProvider handler.ModelDataProvider
	reports       *reportManager
	reportRunner  *threading.TaskRunner
}

// SetTaskRun 注入立即执行闭包，客户端不感知具体调度器实现。
func (c *IspClient) SetTaskRun(run TaskRunFunc) {
	if run == nil {
		c.taskRun.Store(nil)
		return
	}
	c.taskRun.Store(&run)
}

func (c *IspClient) runTask(ctx context.Context, taskCode string) error {
	run := c.taskRun.Load()
	if run == nil {
		return errors.New("任务调度器未初始化")
	}
	return (*run)(ctx, taskCode)
}

// ClientOptions ISP 客户端构造配置。
type ClientOptions struct {
	ReportOpts []ReportManagerOption
}

// ClientOption ISP 客户端构造选项。
type ClientOption func(*ClientOptions)

// WithReportOption 传入上报管理器构造选项。
func WithReportOption(opts ...ReportManagerOption) ClientOption {
	return func(o *ClientOptions) { o.ReportOpts = append(o.ReportOpts, opts...) }
}

// NewClient 创建 ispagent 业务 ISP 客户端。
func NewClient(cfg isp.ClientConfig, taskStore crontask.TaskStore, db *gormx.DB, uploader *ftps.Uploader, provider handler.ModelDataProvider, opts ...ClientOption) *IspClient {
	cfg.ApplyDefaults()
	o := &ClientOptions{}
	for _, opt := range opts {
		opt(o)
	}
	if provider == nil {
		provider = handler.DefaultModelDataProvider{}
	}
	c := &IspClient{
		taskStore:     taskStore,
		db:            db,
		modelUploader: uploader,
		modelProvider: provider,
		reports:       newReportManager(o.ReportOpts...),
		reportRunner:  threading.NewTaskRunner(reportTaskConcurrency),
	}
	c.Client = isp.MustNewClient(cfg,
		isp.WithClientHandler(c.registerHandlers),
		isp.WithClientOnRegister(c.onRegister),
	)
	go c.reportLoop()
	return c
}

// ---------------------------------------------------------------------------
// 连接
// ---------------------------------------------------------------------------

func (c *IspClient) registerHandlers(router *isp.ClientRouter) {
	// ---- 任务下发 101-1 ----
	router.Handle(isp.MessageIDTaskDispatch, func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		return c.Response(ctx, req, handler.HandleTaskDispatch(ctx, req, c.taskStore), nil), nil
	})

	// ---- 任务控制 41-1/2/3/4 ----
	taskControlHandler := func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		taskPatrolledID, err := handler.HandleTaskControl(ctx, req, c.taskStore, c.runTask, c.db, func(ctx context.Context, code string, items []isp.Item) {
			if _, e := c.Execute(ctx, isp.TypeTaskStatusData, isp.CommandReport, code, items); e != nil {
				logx.Errorf("[ispagent] 任务控制通知发送失败: %v", e)
			}
		})
		if err != nil {
			return c.Response(ctx, req, err, nil), nil
		}
		return c.Response(ctx, req, nil, []isp.Item{{"task_patrolled_id": taskPatrolledID}}), nil
	}
	router.HandlePairs(isp.TaskControlPairs, taskControlHandler)

	// ---- 模型更新上报 36-0 ----
	router.Handle(isp.MessageIDModelUpdateReport, func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		return c.Response(ctx, req, handler.HandleModelUpdateReport(ctx, req), nil), nil
	})

	// ---- 模型同步 61-1~12 ----
	modelSyncHandler := func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		items, err := handler.HandleModelSync(ctx, req, c.modelUploader, c.modelProvider)
		return c.Response(ctx, req, err, items), nil
	}
	router.HandlePairs(isp.ModelSyncPairs, modelSyncHandler)

	// ---- 机器人控制 21~29 ----
	robotControlHandler := func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (*isp.Message, error) {
		return c.Response(ctx, req, handler.HandleRobotControl(ctx, req), nil), nil
	}
	router.HandlePairs(isp.RobotControlPairs, robotControlHandler)
}

func (c *IspClient) onRegister(resp *isp.Message) { c.reports.applyRegistrationIntervals(resp.Items) }

// ---------------------------------------------------------------------------
// 轮询
// ---------------------------------------------------------------------------

func (c *IspClient) reportLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Context().Done():
			return
		case <-ticker.C:
			c.reportTick()
		}
	}
}

func (c *IspClient) reportTick() {
	if !c.IsRegistered() {
		return
	}
	now := time.Now()
	for _, report := range c.reports.dueReports(now) {
		report := report
		c.reportRunner.Schedule(func() {
			c.sendReport(now, report)
		})
	}
	c.reportRunner.Wait()
}

func (c *IspClient) sendReport(now time.Time, report reportSnapshot) {
	typ, cmd := isp.DecodeMessageID(int(report.category))
	reqCtx, cancel := context.WithTimeout(c.Context(), c.RequestTimeout())
	logx.WithContext(reqCtx).Debugf("[ispagent] 定时上报 name=%s code=%s items=%d", categoryMessageName(report.category), report.code, len(report.items))
	_, err := c.Execute(reqCtx, typ, cmd, report.code, report.items)
	cancel()
	if err != nil {
		logx.Errorf("[ispagent] 定时上报失败 name=%s: %v", categoryMessageName(report.category), err)
		return
	}
	c.reports.markSent(report.category, report.code, now, report.snapLastSent)
}
