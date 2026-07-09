package ispclient

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"zero-service/app/ispagent/internal/config"
	"zero-service/app/ispagent/internal/handler"
	"zero-service/common/crontask"
	"zero-service/common/gnetx"
	"zero-service/common/isp"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Manager 管理 ISP TCP 长连接的生命周期：建连、注册、心跳、指令收发。
//
// 采用乐观轮询模式，每 2s 检查连接/注册/心跳状态。
// handler 基于 gnetx.Router 按 messageId 路由入站消息，未匹配的消息自动回复 251-3 通用应答。
type Manager struct {
	cfg       config.IspSetting
	taskStore crontask.TaskStore

	mu            sync.RWMutex
	cli           *gnetx.Client
	ctx           context.Context
	cancel        context.CancelFunc
	receiveCode   string
	registered    bool
	lastSessID    string
	heartbeat     time.Duration
	lastHeartbeat time.Time
	registering   atomic.Bool // 防止并发 doRegister
	heartbeating  atomic.Bool // 防止并发 sendHeartbeat

	lastRecvSeq atomic.Value // recvSeq{sessionID, seq}
}

type recvSeq struct {
	sessionID string
	seq       uint64
}

// NewManager 创建 ISP 客户端管理器。
func NewManager(cfg config.IspSetting, taskStore crontask.TaskStore) *Manager {
	cfg.ApplyDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		cfg:       cfg,
		taskStore: taskStore,
		ctx:       ctx,
		cancel:    cancel,
		heartbeat: cfg.HeartbeatInterval,
	}
	m.lastRecvSeq.Store(recvSeq{})
	_ = m.connect()
	go m.run()
	return m
}

func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	cli := m.cli
	m.cli = nil
	m.mu.Unlock()
	if cli != nil {
		cli.Close()
	}
}

// Execute 发送指令并同步等待响应。未注册时返回 Unavailable。
func (m *Manager) Execute(ctx context.Context, typ, command int32, code string, items []isp.Item) (*isp.Message, error) {
	if typ <= 0 {
		return nil, status.Error(codes.InvalidArgument, "type 必须大于 0")
	}
	if !m.isRegistered() {
		return nil, status.Error(codes.Unavailable, "isp tcp 未注册，请等待连接就绪")
	}
	msg := &isp.Message{
		RootName:      m.cfg.RootName,
		SessionSource: isp.SessionSourceClient,
		SendCode:      m.cfg.SendCode,
		ReceiveCode:   m.currentReceiveCode(),
		Type:          typ,
		Code:          code,
		Command:       command,
		Time:          defaultTime(""),
		Items:         items,
	}
	return m.request(ctx, msg)
}

// ---------------------------------------------------------------------------
// 连接
// ---------------------------------------------------------------------------

func (m *Manager) connect() error {
	codec := isp.NewCodec(m.cfg.RootName, m.cfg.MaxFrameLength, m.cfg.DebugLog)
	router := gnetx.NewRouter(nil)
	// 任务下发指令 (101-1)：打印任务详情 + 回复成功
	gnetx.HandleTyped(router, isp.MessageIDTaskDispatch, func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
		handler.LogInbound(ctx, req)
		err := handler.HandleTaskDispatch(ctx, req, m.taskStore)
		m.trackRecvSeq(req.SendSeq, conn.ID())
		return m.responseWithCode(conn, req, handler.ResponseCode(err)), nil
	})
	// 任务控制指令 (41-1/2/3/4)：服务端控制任务启动/暂停/继续/停止
	// TODO: 异步发送协程后续改为硬件下发指令，等硬件确认后再通知
	taskControlHandler := func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
		handler.LogInbound(ctx, req)
		taskPatrolledID, err := handler.HandleTaskControl(ctx, req, m.taskStore, func(ctx context.Context, code string, items []isp.Item) {
			if _, e := m.Execute(ctx, isp.TypeTaskStatusData, isp.CommandReport, code, items); e != nil {
				logx.Errorf("[ispagent] 任务控制通知发送失败: %v", e)
			}
		})
		m.trackRecvSeq(req.SendSeq, conn.ID())
		if err != nil {
			return m.responseWithCode(conn, req, handler.ResponseCode(err)), nil
		}
		respItems := []isp.Item{{"task_patrolled_id": taskPatrolledID}}
		return m.responseWithItems(conn, req, isp.StatusSuccess, respItems), nil
	}
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeTaskControl, isp.CommandTaskStart), taskControlHandler)
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeTaskControl, isp.CommandTaskPause), taskControlHandler)
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeTaskControl, isp.CommandTaskResume), taskControlHandler)
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeTaskControl, isp.CommandTaskStop), taskControlHandler)

	// 模型更新上报 (11-0)：服务端推送模型文件列表
	gnetx.HandleTyped(router, isp.MessageIDModelUpdateReport, func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
		handler.LogInbound(ctx, req)
		err := handler.HandleModelUpdateReport(ctx, req)
		m.trackRecvSeq(req.SendSeq, conn.ID())
		return m.responseWithCode(conn, req, handler.ResponseCode(err)), nil
	})
	// 模型同步拉取 (61-2/4/9)：服务端主动拉取模型文件
	modelSyncHandler := func(ctx context.Context, conn gnetx.Conn, req *isp.Message) (any, error) {
		handler.LogInbound(ctx, req)
		items, err := handler.HandleModelSync(ctx, req)
		m.trackRecvSeq(req.SendSeq, conn.ID())
		if err != nil || len(items) == 0 {
			return m.responseWithCode(conn, req, handler.ResponseCode(err)), nil
		}
		return m.responseWithItems(conn, req, isp.StatusSuccess, items), nil
	}
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeModelSync, isp.CommandModelRobot), modelSyncHandler) // 61-2
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeModelSync, isp.CommandModelPoint), modelSyncHandler) // 61-4
	gnetx.HandleTyped(router, isp.EncodeMessageID(isp.TypeModelSync, isp.CommandModelMap), modelSyncHandler)   // 61-9
	// 未匹配入站消息：fallback 日志 + 回复 251-3
	router.FallbackFunc(func(ctx context.Context, conn gnetx.Conn, msg any) (any, error) {
		im, ok := msg.(*isp.Message)
		if !ok {
			return nil, nil
		}
		handler.LogFallback(ctx, im)
		m.trackRecvSeq(im.SendSeq, conn.ID())
		return m.defaultResponse(conn, im), nil
	})
	cli, err := gnetx.NewClient(m.cfg.ServerAddr,
		gnetx.WithClientCodec(codec),
		gnetx.WithClientHandler(router),
		gnetx.WithClientMaxFrameLength(m.cfg.MaxFrameLength),
		gnetx.WithClientReconnectInterval(m.cfg.ReconnectInterval),
	)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.cli = cli
	m.mu.Unlock()
	return nil
}

// defaultResponse 构造 251-3 通用应答（无Item），默认成功。
func (m *Manager) defaultResponse(conn gnetx.Conn, req *isp.Message) *isp.Message {
	return m.responseWithCode(conn, req, isp.StatusSuccess)
}

// responseWithCode 构造带指定 Code 的 251-3 通用应答。
func (m *Manager) responseWithCode(conn gnetx.Conn, req *isp.Message, code string) *isp.Message {
	return m.makeResponse(conn, req, code, isp.CommandGenericResponseWithoutItems, nil)
}

// responseWithItems 构造带 Code 和 Items 的 251-4 通用应答。
func (m *Manager) responseWithItems(conn gnetx.Conn, req *isp.Message, code string, items []isp.Item) *isp.Message {
	return m.makeResponse(conn, req, code, isp.CommandGenericResponseWithItems, items)
}

func (m *Manager) makeResponse(conn gnetx.Conn, req *isp.Message, code string, command int32, items []isp.Item) *isp.Message {
	m.mu.RLock()
	rootName := m.cfg.RootName
	sendCode := m.cfg.SendCode
	receiveCode := m.receiveCode
	m.mu.RUnlock()
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

// ---------------------------------------------------------------------------
// 轮询
// ---------------------------------------------------------------------------

func (m *Manager) run() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.tick()
		}
	}
}

func (m *Manager) tick() {
	sess := m.currentSession()
	if sess == nil {
		return
	}
	sessID := sess.ID()

	m.mu.Lock()
	if m.lastSessID != sessID {
		m.lastSessID = sessID
		m.registered = false
		m.lastRecvSeq.Store(recvSeq{})
		m.receiveCode = ""
	}
	registered := m.registered
	interval := m.heartbeat
	elapsed := time.Since(m.lastHeartbeat)
	m.mu.Unlock()

	if !registered {
		m.doRegister()
		return
	}
	if elapsed >= interval {
		m.sendHeartbeat()
	}
}

// ---------------------------------------------------------------------------
// 注册
// ---------------------------------------------------------------------------

// doRegister 发起注册（用 registering CAS 防止并发）
func (m *Manager) doRegister() {
	if !m.registering.CompareAndSwap(false, true) {
		return
	}
	defer m.registering.Store(false)
	msg := &isp.Message{
		RootName:      m.cfg.RootName,
		SessionSource: isp.SessionSourceClient,
		SendCode:      m.cfg.SendCode,
		ReceiveCode:   m.cfg.RegisterReceiveCode,
		Type:          isp.TypeSystem,
		Command:       isp.CommandRegister,
		Time:          defaultTime(""),
	}
	reqCtx, cancel := context.WithTimeout(m.ctx, m.cfg.RequestTimeout)
	defer cancel()

	resp, err := m.request(reqCtx, msg)
	if err != nil {
		logx.Errorf("[ispagent] 注册失败: %v", err)
		m.closeCurrentConn()
		return
	}
	remote := resp.SendCode
	if remote == "" {
		remote = resp.ReceiveCode
	}
	hb := heartbeatFromItems(resp.Items, m.heartbeat)

	m.mu.Lock()
	if remote != "" {
		m.receiveCode = remote
	}
	m.heartbeat = hb
	m.registered = true
	m.lastHeartbeat = time.Now()
	m.mu.Unlock()

	logx.Infof("[ispagent] 注册成功, receiveCode=%s, heartbeat=%s", m.receiveCode, m.heartbeat)
}

// ---------------------------------------------------------------------------
// 心跳
// ---------------------------------------------------------------------------

func (m *Manager) sendHeartbeat() {
	if !m.heartbeating.CompareAndSwap(false, true) {
		return
	}
	defer m.heartbeating.Store(false)

	m.mu.Lock()
	m.lastHeartbeat = time.Now()
	m.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(m.ctx, m.cfg.RequestTimeout)
	defer cancel()
	if _, err := m.Execute(reqCtx, isp.TypeSystem, isp.CommandHeartbeat, "", nil); err != nil {
		logx.Errorf("[ispagent] 心跳失败: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 请求
// ---------------------------------------------------------------------------

func (m *Manager) request(ctx context.Context, msg *isp.Message) (*isp.Message, error) {
	sess := m.currentSession()
	if sess == nil {
		return nil, status.Error(codes.Unavailable, "isp tcp 会话未就绪")
	}
	msg.SendSeq = sess.NextSendSeq()
	rs, _ := m.lastRecvSeq.Load().(recvSeq)
	msg.RecvSeq = rs.seq
	sessID := sess.ID()
	if msg.Time == "" {
		msg.Time = defaultTime("")
	}
	ttl := m.cfg.RequestTimeout
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
	m.trackRecvSeq(resp.SendSeq, sessID)
	return resp, nil
}

func (m *Manager) trackRecvSeq(seq uint64, sessionID string) {
	if seq == 0 || sessionID == "" {
		return
	}
	for {
		old := m.lastRecvSeq.Load().(recvSeq)
		if old.sessionID == sessionID && seq <= old.seq {
			return
		}
		if m.lastRecvSeq.CompareAndSwap(old, recvSeq{sessionID: sessionID, seq: seq}) {
			return
		}
	}
}

// ---------------------------------------------------------------------------
// 工具
// ---------------------------------------------------------------------------

func (m *Manager) currentSession() gnetx.ClientConn {
	m.mu.RLock()
	cli := m.cli
	m.mu.RUnlock()
	if cli == nil {
		return nil
	}
	return cli.Session()
}

func (m *Manager) isRegistered() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registered
}

func (m *Manager) currentReceiveCode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.receiveCode
}

func (m *Manager) closeCurrentConn() {
	m.mu.RLock()
	cli := m.cli
	m.mu.RUnlock()
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

func heartbeatFromItems(items []isp.Item, fallback time.Duration) time.Duration {
	for _, item := range items {
		if raw := strings.TrimSpace(item["heart_beat_interval"]); raw != "" {
			if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
				return time.Duration(sec) * time.Second
			}
		}
	}
	return fallback
}
