package isp

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"zero-service/app/ispagent/internal/config"
	"zero-service/app/ispagent/internal/handler"
	"zero-service/common/crontask"
	"zero-service/common/ftps"
	"zero-service/common/gnetx"
	"zero-service/common/gormx"
	"zero-service/common/isp"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client 管理 ISP TCP 长连接的生命周期：建连、注册、心跳、指令收发。
//
// 采用乐观轮询模式，每 2s 检查连接/注册/心跳状态。
// handler 基于 gnetx.Router 按 messageId 路由入站消息，未匹配的消息自动回复 251-3 通用应答。
type Client struct {
	cfg           config.IspSetting
	taskStore     crontask.TaskStore
	db            *gormx.DB
	modelUploader *ftps.Uploader
	modelProvider handler.ModelDataProvider

	mu            sync.RWMutex
	cli           *gnetx.Client
	ctx           context.Context
	cancel        context.CancelFunc
	receiveCode   string
	registered    bool
	lastSessID    string
	heartbeat     time.Duration
	lastHeartbeat time.Time
	reports       *reportManager
	registering   atomic.Bool
	heartbeating  atomic.Bool

	lastRecvSeq atomic.Value
}

type recvSeq struct {
	sessionID string
	seq       uint64
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

func NewClient(cfg config.IspSetting, taskStore crontask.TaskStore, db *gormx.DB, uploader *ftps.Uploader, provider handler.ModelDataProvider, opts ...ClientOption) *Client {
	cfg.ApplyDefaults()
	o := &ClientOptions{}
	for _, opt := range opts {
		opt(o)
	}
	if provider == nil {
		provider = handler.DefaultModelDataProvider{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		cfg:           cfg,
		taskStore:     taskStore,
		db:            db,
		modelUploader: uploader,
		modelProvider: provider,
		ctx:           ctx,
		cancel:        cancel,
		heartbeat:     cfg.HeartbeatInterval,
		reports:       newReportManager(o.ReportOpts...),
	}
	c.lastRecvSeq.Store(recvSeq{})
	_ = c.connect()
	go c.run()
	return c
}

func (c *Client) Close() {
	c.cancel()
	c.mu.Lock()
	cli := c.cli
	c.cli = nil
	c.mu.Unlock()
	if cli != nil {
		cli.Close()
	}
}

// Execute 发送指令并同步等待响应。未注册时返回 Unavailable。
func (c *Client) Execute(ctx context.Context, typ, command int32, code string, items []isp.Item) (*isp.Message, error) {
	if typ <= 0 {
		return nil, status.Error(codes.InvalidArgument, "type 必须大于 0")
	}
	if !c.isRegistered() {
		return nil, status.Error(codes.Unavailable, "isp tcp 未注册，请等待连接就绪")
	}
	msg := &isp.Message{
		RootName:      c.cfg.RootName,
		SessionSource: isp.SessionSourceClient,
		SendCode:      c.cfg.SendCode,
		ReceiveCode:   c.ReceiveCode(),
		Type:          typ,
		Code:          code,
		Command:       command,
		Time:          defaultTime(""),
		Items:         items,
	}
	return c.request(ctx, msg)
}

// ---------------------------------------------------------------------------
// 连接
// ---------------------------------------------------------------------------

func (c *Client) connect() error {
	codec := isp.NewCodec(c.cfg.RootName, c.cfg.MaxFrameLength, c.cfg.DebugLog)
	router := gnetx.NewRouter()

	// ---- 任务下发 101-1 ----
	gnetx.HandleTypedAsync(router, isp.MessageIDTaskDispatch, c.wrap(func(ctx context.Context, req *isp.Message) ([]isp.Item, error) {
		return nil, handler.HandleTaskDispatch(ctx, req, c.taskStore)
	}))

	// ---- 任务控制 41-1/2/3/4 ----
	taskControlHandler := c.wrap(func(ctx context.Context, req *isp.Message) ([]isp.Item, error) {
		taskPatrolledID, err := handler.HandleTaskControl(ctx, req, c.taskStore, c.db, c.cfg.SendCode, c.ReceiveCode(), func(ctx context.Context, code string, items []isp.Item) {
			if _, e := c.Execute(ctx, isp.TypeTaskStatusData, isp.CommandReport, code, items); e != nil {
				logx.Errorf("[ispagent] 任务控制通知发送失败: %v", e)
			}
		})
		if err != nil {
			return nil, err
		}
		return []isp.Item{{"task_patrolled_id": taskPatrolledID}}, nil
	})
	for _, pair := range isp.TaskControlPairs {
		gnetx.HandleTypedAsync(router, isp.EncodeMessageID(pair.Type, pair.Cmd), taskControlHandler)
	}

	// ---- 模型更新上报 36-0 ----
	gnetx.HandleTypedAsync(router, isp.MessageIDModelUpdateReport, c.wrap(func(ctx context.Context, req *isp.Message) ([]isp.Item, error) {
		return nil, handler.HandleModelUpdateReport(ctx, req)
	}))

	// ---- 模型同步 61-1~12 ----
	modelSyncHandler := c.wrap(func(ctx context.Context, req *isp.Message) ([]isp.Item, error) {
		return handler.HandleModelSync(ctx, req, c.modelUploader, c.modelProvider)
	})
	for _, pair := range isp.ModelSyncPairs {
		gnetx.HandleTypedAsync(router, isp.EncodeMessageID(pair.Type, pair.Cmd), modelSyncHandler)
	}

	// ---- 机器人控制 21~29 ----
	robotControlHandler := c.wrap(func(ctx context.Context, req *isp.Message) ([]isp.Item, error) {
		return nil, handler.HandleRobotControl(ctx, req)
	})
	for _, pair := range isp.RobotControlPairs {
		gnetx.HandleTypedAsync(router, isp.EncodeMessageID(pair.Type, pair.Cmd), robotControlHandler)
	}

	// ---- 未匹配消息 ----
	router.FallbackFuncAsync(func(ctx context.Context, conn gnetx.Conn, msg any) (any, error) {
		im, ok := msg.(*isp.Message)
		if !ok {
			return nil, nil
		}
		handler.LogFallback(ctx, im)
		c.trackRecvSeq(im.SendSeq, conn.ID())
		return c.response(ctx, conn, im, nil), nil
	})
	cli, err := gnetx.NewClient(c.cfg.ServerAddr,
		gnetx.WithClientCodec(codec),
		gnetx.WithClientHandler(router),
		gnetx.WithClientMaxFrameLength(c.cfg.MaxFrameLength),
		gnetx.WithClientReconnectInterval(c.cfg.ReconnectInterval),
	)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cli = cli
	c.mu.Unlock()
	return nil
}

func (c *Client) response(ctx context.Context, conn gnetx.Conn, req *isp.Message, err error) *isp.Message {
	code := isp.ResponseCode(err)
	c.auditError(ctx, code, req, err)
	return c.newResponse(conn, req, code, isp.CommandGenericResponseWithoutItems, nil)
}

func (c *Client) responseWithItems(ctx context.Context, conn gnetx.Conn, req *isp.Message, items []isp.Item) *isp.Message {
	return c.newResponse(conn, req, isp.StatusSuccess, isp.CommandGenericResponseWithItems, items)
}

func (c *Client) newResponse(conn gnetx.Conn, req *isp.Message, code string, command int32, items []isp.Item) *isp.Message {
	c.mu.RLock()
	rootName := c.cfg.RootName
	sendCode := c.cfg.SendCode
	receiveCode := c.receiveCode
	c.mu.RUnlock()
	return &isp.Message{
		RootName:      rootName,
		SessionSource: isp.SessionSourceClient,
		SendCode:      sendCode,
		ReceiveCode:   receiveCode,
		Type:          isp.TypeSystem,
		Code:          code,
		Command:       command,
		SendSeq:       conn.NextSendSeq(),
		RecvSeq:       req.SendSeq,
		Time:          defaultTime(""),
		Items:         items,
	}
}

func (c *Client) wrap(h func(ctx context.Context, req *isp.Message) ([]isp.Item, error)) func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
	return func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
		handler.LogInbound(ctx, req)
		items, err := h(ctx, req)
		c.trackRecvSeq(req.SendSeq, conn.ID())
		if err != nil || len(items) == 0 {
			return c.response(ctx, conn, req, err), nil
		}
		return c.responseWithItems(ctx, conn, req, items), nil
	}
}

func (c *Client) auditError(ctx context.Context, code string, req *isp.Message, err error) {
	if code == isp.StatusSuccess || code == isp.StatusRetry {
		return
	}
	name := code
	switch code {
	case isp.StatusReject:
		name = "拒绝(400)"
	case isp.StatusError:
		name = "错误(500)"
	}
	var ie *isp.IspError
	if errors.As(err, &ie) {
		logx.WithContext(ctx).Errorf("[ispagent] 回复%s type=%d command=%d reqCode=%s msg=%s", name, req.Type, req.Command, req.Code, ie.Msg)
	} else if err != nil {
		logx.WithContext(ctx).Errorf("[ispagent] 回复%s type=%d command=%d reqCode=%s err=%v", name, req.Type, req.Command, req.Code, err)
	}
}

// ---------------------------------------------------------------------------
// 轮询
// ---------------------------------------------------------------------------

func (c *Client) run() {
	go c.reportLoop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.tick()
		}
	}
}

func (c *Client) reportLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.reportTick()
		}
	}
}

func (c *Client) tick() {
	sess := c.currentSession()
	if sess == nil {
		return
	}
	sessID := sess.ID()

	c.mu.Lock()
	if c.lastSessID != sessID {
		c.lastSessID = sessID
		c.registered = false
		c.lastRecvSeq.Store(recvSeq{})
		c.receiveCode = ""
	}
	registered := c.registered
	interval := c.heartbeat
	elapsed := time.Since(c.lastHeartbeat)
	c.mu.Unlock()

	if !registered {
		c.doRegister()
		return
	}
	if elapsed >= interval {
		c.sendHeartbeat()
	}
}

// ---------------------------------------------------------------------------
// 注册
// ---------------------------------------------------------------------------

func (c *Client) doRegister() {
	if !c.registering.CompareAndSwap(false, true) {
		return
	}
	defer c.registering.Store(false)
	msg := &isp.Message{
		RootName:      c.cfg.RootName,
		SessionSource: isp.SessionSourceClient,
		SendCode:      c.cfg.SendCode,
		ReceiveCode:   c.cfg.RegisterReceiveCode,
		Type:          isp.TypeSystem,
		Command:       isp.CommandRegister,
		Time:          defaultTime(""),
	}
	reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
	defer cancel()

	resp, err := c.request(reqCtx, msg)
	if err != nil {
		logx.Errorf("[ispagent] 注册失败: %v", err)
		c.closeCurrentConn()
		return
	}
	hb := c.heartbeat
	if len(resp.Items) > 0 {
		hb = parseItemInterval(resp.Items[0], "heart_beat_interval", hb)
	}

	c.mu.Lock()
	if resp.SendCode != "" {
		c.receiveCode = resp.SendCode
	}
	c.heartbeat = hb
	c.registered = true
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()

	c.reports.applyRegistrationIntervals(resp.Items)

	logx.Infof("[ispagent] 注册成功, receiveCode=%s, heartbeat=%s", c.receiveCode, c.heartbeat)
}

func (c *Client) reportTick() {
	if !c.isRegistered() {
		return
	}
	now := time.Now()
	for _, report := range c.reports.dueReports(now) {
		typ, cmd := isp.DecodeMessageID(int(report.category))
		reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
		logx.WithContext(reqCtx).Debugf("[ispagent] 定时上报 name=%s code=%s items=%d", categoryMessageName(report.category), report.code, len(report.items))
		_, err := c.Execute(reqCtx, typ, cmd, report.code, report.items)
		cancel()
		if err != nil {
			logx.Errorf("[ispagent] 定时上报失败 name=%s: %v", categoryMessageName(report.category), err)
			continue
		}
		c.reports.markSent(report.category, report.code, now, report.snapLastSent)
	}
}

// ---------------------------------------------------------------------------
// 心跳
// ---------------------------------------------------------------------------

func (c *Client) sendHeartbeat() {
	if !c.heartbeating.CompareAndSwap(false, true) {
		return
	}
	defer c.heartbeating.Store(false)

	c.mu.Lock()
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(c.ctx, c.cfg.RequestTimeout)
	defer cancel()
	if _, err := c.Execute(reqCtx, isp.TypeSystem, isp.CommandHeartbeat, "", nil); err != nil {
		logx.Errorf("[ispagent] 心跳失败: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 请求
// ---------------------------------------------------------------------------

func (c *Client) request(ctx context.Context, msg *isp.Message) (*isp.Message, error) {
	sess := c.currentSession()
	if sess == nil {
		return nil, status.Error(codes.Unavailable, "isp tcp 会话未就绪")
	}
	msg.SendSeq = sess.NextSendSeq()
	rs, _ := c.lastRecvSeq.Load().(recvSeq)
	msg.RecvSeq = rs.seq
	sessID := sess.ID()
	if msg.Time == "" {
		msg.Time = defaultTime("")
	}
	ttl := c.cfg.RequestTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d > 0 && d < ttl {
			ttl = d
		}
	}
	handler.LogOutbound(ctx, msg)
	respAny, err := sess.Request(ctx, msg, ttl)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "isp tcp 请求失败: %v", err)
	}
	resp, ok := respAny.(*isp.Message)
	if !ok {
		return nil, status.Errorf(codes.Internal, "isp tcp 响应类型异常: %T", respAny)
	}
	c.trackRecvSeq(resp.SendSeq, sessID)
	return resp, nil
}

func (c *Client) trackRecvSeq(seq uint64, sessionID string) {
	if seq == 0 || sessionID == "" {
		return
	}
	for {
		old := c.lastRecvSeq.Load().(recvSeq)
		if old.sessionID == sessionID && seq <= old.seq {
			return
		}
		if c.lastRecvSeq.CompareAndSwap(old, recvSeq{sessionID: sessionID, seq: seq}) {
			return
		}
	}
}

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------

func (c *Client) currentSession() gnetx.ClientConn {
	c.mu.RLock()
	cli := c.cli
	c.mu.RUnlock()
	if cli == nil {
		return nil
	}
	return cli.Session()
}

func (c *Client) isRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

func (c *Client) Connected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cli != nil && c.cli.Session() != nil && c.registered
}

func (c *Client) ReceiveCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.receiveCode
}

func (c *Client) closeCurrentConn() {
	c.mu.RLock()
	cli := c.cli
	c.mu.RUnlock()
	if cli == nil {
		return
	}
	sess := cli.Session()
	if sess != nil {
		_ = sess.Close()
	}
}

func defaultTime(v string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return carbon.Now().ToDateTimeString()
}
