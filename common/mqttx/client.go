package mqttx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tool "zero-service/common/tool"
	tracex "zero-service/common/trace"

	"github.com/duke-git/lancet/v2/random"
	"github.com/duke-git/lancet/v2/slice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OpenTelemetry 属性 key
const (
	attrTopicTemplate = "mqtt.topic_template"
	attrClientID      = "mqtt.client_id"
	attrMsgID         = "mqtt.message_id"
	attrQoS           = "mqtt.qos"
	attrAction        = "mqtt.action"
)

// Client is the interface for mqttx operations.
// Callers should depend on this interface for publish, handler registration, and lifecycle management.
// Request/reply is driven by the *ReplyRouter[T] returned by NewReplyRouter:
// register it with WithReplyRouter (before the client is created) or with
// Client.AddReplyRouter (after the client is created), then call
// router.RequestReply from the sender.
type Client interface {
	AddHandler(topicTemplate string, handler ConsumeHandler) error
	AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error
	// AddReplyRouter 注册应答路由：登记分发（优先于普通 handler）并按需订阅模板。
	AddReplyRouter(topicTemplate string, router ConsumeHandler) error
	Publish(ctx context.Context, topic string, payload []byte) error
	PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error)
	Close()
	GetClientID() string
}

// mqttClient MQTT 客户端
// 提供 MQTT 连接、订阅、发布等功能
type mqttClient struct {
	client     mqtt.Client
	cfg        MqttConfig
	handlerMgr *handlerManager
	dispatcher *messageDispatcher
	subscribed map[string]struct{}
	onReady    func(Client)
	ready      atomic.Bool
	closed     atomic.Bool
	qos        byte
	mu         sync.RWMutex
	tracer     oteltrace.Tracer
	metrics    *stat.Metrics
}

// MustNewClient 创建 MQTT 客户端，连接失败时 panic
// 推荐在应用启动时使用，会自动注册关闭监听
func MustNewClient(cfg MqttConfig, opts ...ClientOption) Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	proc.AddWrapUpListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient 创建 MQTT 客户端
// 会自动连接 MQTT 服务器，重连由底层库处理
func NewClient(cfg MqttConfig, opts ...ClientOption) (Client, error) {
	if len(cfg.Broker) == 0 {
		return nil, errors.New("no broker addresses provided")
	}

	// 未指定 ClientID 时自动生成
	if len(cfg.ClientID) == 0 {
		uid, err := random.UUIdV4()
		if err != nil {
			return nil, fmt.Errorf("generate clientID failed: %w", err)
		}
		cfg.ClientID = strings.ReplaceAll(uid, "-", "")
	}

	adjustConfig(&cfg)

	o := &ClientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	c := &mqttClient{
		cfg:        cfg,
		handlerMgr: newHandlerManager(),
		subscribed: make(map[string]struct{}),
		qos:        cfg.Qos,
		onReady:    o.onReady,
		tracer:     otel.Tracer(trace.TraceName),
		metrics:    stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
	}
	c.dispatcher = newMessageDispatcher(c.handlerMgr, c.metrics)

	for _, reg := range o.replyRouters {
		c.handlerMgr.addReplyHandler(reg.topicTemplate, reg.handler)
	}

	client, err := c.createMqttClient()
	if err != nil {
		return nil, err
	}
	c.client = client

	if err := c.connect(); err != nil {
		return nil, err
	}

	return c, nil
}

// adjustConfig 修正配置默认值
func adjustConfig(cfg *MqttConfig) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30000
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60000
	}
	if cfg.Qos > 2 {
		cfg.Qos = 1
	}
}

// createMqttClient 创建底层 MQTT 客户端
func (c *mqttClient) createMqttClient() (mqtt.Client, error) {
	optsMqtt := mqtt.NewClientOptions()
	for _, broker := range c.cfg.Broker {
		optsMqtt.AddBroker(broker)
	}

	optsMqtt.SetClientID(c.cfg.ClientID).
		SetUsername(c.cfg.Username).
		SetPassword(c.cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(c.cfg.Timeout) * time.Millisecond).
		SetKeepAlive(time.Duration(c.cfg.KeepAlive) * time.Millisecond)

	tlsCfg, err := c.buildTLSConfig()
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		optsMqtt.SetTLSConfig(tlsCfg)
	}

	optsMqtt.SetDefaultPublishHandler(c.defaultHandler)
	optsMqtt.OnConnect = c.onConnect
	optsMqtt.OnConnectionLost = c.onConnectionLost

	return mqtt.NewClient(optsMqtt), nil
}

// buildTLSConfig 根据 MqttConfig.TLS 构造 paho 使用的 TLS 配置。
//
// 配置规则：
//   - 未配置任何 TLS 参数时返回 (nil, nil)，不覆盖 paho 的默认行为：
//     Broker 为 ssl:// 等加密协议时使用系统根证书校验，为 tcp:// 时保持明文；
//   - CAFile 用于校验自签服务端证书，配置后替换系统根证书池；
//   - CertFile/KeyFile 用于双向认证，必须成对配置；
//   - ServerName 为空时由 TLS 握手按 Broker 地址自动推导，不必手工填写。
//
// 证书、私钥和 CA 在建立连接前一次性读取，任何配置错误都直接返回 error，
// 避免连接阶段静默降级为明文或不可用状态。
func (c *mqttClient) buildTLSConfig() (*tls.Config, error) {
	if c.cfg.TLS.isEmpty() {
		return nil, nil
	}

	tlsCfg := &tls.Config{
		// 禁用 TLS1.0/1.1，避免旧协议降级；供应商网关普遍支持 TLS1.2 及以上。
		MinVersion:         tls.VersionTLS12,
		ServerName:         c.cfg.TLS.ServerName,
		InsecureSkipVerify: c.cfg.TLS.InsecureSkipVerify,
	}

	// 只配置一半时视为配置错误，避免调用方误以为已启用双向认证。
	hasCert, hasKey := c.cfg.TLS.CertFile != "", c.cfg.TLS.KeyFile != ""
	if hasCert != hasKey {
		return nil, errors.New("mqtt tls: certFile and keyFile must be configured together")
	}
	if hasCert {
		cert, err := tls.LoadX509KeyPair(c.cfg.TLS.CertFile, c.cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("mqtt tls: load client certificate failed: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	// 供应商自签 CA 常见于内网网关；CA 文件读取或解析失败必须报错，
	// 不能退回系统根证书，否则连接会因证书不受信任而失败且难以定位。
	if c.cfg.TLS.CAFile != "" {
		caCert, err := os.ReadFile(c.cfg.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("mqtt tls: read CA file failed: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("mqtt tls: no valid certificate found in CA file %s", c.cfg.TLS.CAFile)
		}
		tlsCfg.RootCAs = caPool
	}

	return tlsCfg, nil
}

func (c *mqttClient) connect() error {
	token := c.client.Connect()

	if !token.WaitTimeout(time.Duration(c.cfg.Timeout) * time.Millisecond) {
		return fmt.Errorf("connect timeout after %dms", c.cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	return nil
}

// onConnect 连接成功回调
//
// 首次连接时 SubscribeAll 之类的调用方通常在 NewClient 返回后立即 AddHandler，
// 那些 Topic 由 subscribeIfNeeded 即时订阅，restoreSubscriptions 此时可能只看到
// 部分（甚至零个）已注册 Topic，这属于正常时序而不是订阅丢失，因此日志用
// phase/total 明确区分「本次恢复动作的范围」和「进程最终订阅的主题」。
func (c *mqttClient) onConnect(_ mqtt.Client) {
	firstConnect := c.ready.CompareAndSwap(false, true)
	logx.Infof("[mqtt] connected client=%s phase=%s", c.cfg.ClientID, connectPhase(firstConnect))

	// 首次连接执行 onReady 回调
	if firstConnect && c.onReady != nil {
		c.onReady(c)
	}

	result, err := c.restoreSubscriptions()
	switch {
	case result.total == 0:
		// 首次连接时处理器往往还没注册，这里刻意不打 0/0，避免误读为恢复失败。
		logx.Infof("[mqtt] restore done total=0 (no topics registered yet, handled by AddHandler)")
	case err != nil:
		logx.Errorf("[mqtt] restore done total=%d ok=%d skip=%d fail=%d err=%v",
			result.total, result.subscribed, result.skipped, result.failed, err)
	default:
		logx.Infof("[mqtt] restore done total=%d ok=%d skip=%d fail=%d",
			result.total, result.subscribed, result.skipped, result.failed)
	}
}

// connectPhase 返回本次连接阶段，用于区分首发连接与断线重连。
func connectPhase(firstConnect bool) string {
	if firstConnect {
		return "first-connect"
	}
	return "reconnect"
}

// onConnectionLost 连接丢失回调
func (c *mqttClient) onConnectionLost(_ mqtt.Client, err error) {
	c.mu.Lock()
	cleared := len(c.subscribed)
	c.subscribed = make(map[string]struct{}) // 重连后需要重新订阅
	c.mu.Unlock()

	logx.Errorf("[mqtt] connection lost, subscriptions cleared count=%d (will restore on reconnect) err=%v", cleared, err)
}

// defaultHandler 默认消息处理器（无自定义处理器时）
func (c *mqttClient) defaultHandler(_ mqtt.Client, msg mqtt.Message) {
	logx.Errorf("[mqtt] no handler topic=%s", msg.Topic())
}

// AddHandler 为订阅主题模板添加消息处理器。
// 如果已连接且 topic 尚未订阅，会自动订阅。
// reply router 请使用 WithReplyRouter（创建前）或 AddReplyRouter（创建后），
// 不要把 ReplyRouter 传入此方法。
func (c *mqttClient) AddHandler(topicTemplate string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("handler cannot be nil")
	}

	c.handlerMgr.addHandler(topicTemplate, handler)
	return c.subscribeIfNeeded(topicTemplate)
}

// AddHandlerFunc 快捷方法：用函数作为消息处理器
func (c *mqttClient) AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error {
	return c.AddHandler(topicTemplate, ConsumeHandlerFunc(fn))
}

// AddReplyRouter 注册应答路由：登记分发（优先于普通 handler）并按需订阅模板。
// 与 WithReplyRouter 的区别是可在 client 创建后调用，构造与注册不必分离。
func (c *mqttClient) AddReplyRouter(topicTemplate string, router ConsumeHandler) error {
	if router == nil {
		return errors.New("reply router cannot be nil")
	}

	c.handlerMgr.addReplyHandler(topicTemplate, router)
	return c.subscribeIfNeeded(topicTemplate)
}

// subscribeIfNeeded 在已连接且模板尚未订阅时立即订阅；未连接时保持静默，
// 由连接回调里的 restoreSubscriptions 统一恢复，避免各注册入口重复实现。
func (c *mqttClient) subscribeIfNeeded(topicTemplate string) error {
	c.mu.Lock()
	needSubscribe := !c.isSubscribed(topicTemplate)
	c.mu.Unlock()

	if needSubscribe && c.client.IsConnected() {
		_, err := c.subscribe(topicTemplate)
		return err
	}
	return nil
}

// subscribe 内部订阅实现
func (c *mqttClient) subscribe(topicTemplate string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isSubscribed(topicTemplate) {
		return false, nil
	}

	token := c.client.Subscribe(topicTemplate, c.qos, c.messageHandler(topicTemplate))
	timeout := time.Duration(c.cfg.Timeout) * time.Millisecond

	if !token.WaitTimeout(timeout) {
		return false, fmt.Errorf("subscribe timeout after %v", timeout)
	}
	if err := token.Error(); err != nil {
		return false, err
	}

	c.subscribed[topicTemplate] = struct{}{}
	logx.Infof("[mqtt] subscribe ok topic=%s", topicTemplate)
	return true, nil
}

// isSubscribed 检查是否已订阅
func (c *mqttClient) isSubscribed(topicTemplate string) bool {
	_, exists := c.subscribed[topicTemplate]
	return exists
}

// restoreSubscriptions 恢复所有订阅（连接重连后）。
//
// 统计口径：total 为本次去重后的 Topic 模板数，恒等于 ok+skip+fail；
// 它只描述「本次恢复动作」，不是进程的订阅总数（见 onConnect 注释）。
func (c *mqttClient) restoreSubscriptions() (restoreSubscriptionsResult, error) {
	topicTemplates := c.handlerMgr.getAllTopicTemplates()
	topicTemplates = append(topicTemplates, c.cfg.SubscribeTopics...)
	topicTemplates = uniqueTopics(topicTemplates)

	var lastErr error
	result := restoreSubscriptionsResult{total: len(topicTemplates)}
	for _, topicTemplate := range topicTemplates {
		subscribed, err := c.subscribe(topicTemplate)
		switch {
		case err != nil:
			logx.Errorf("[mqtt] subscribe failed topic=%s err=%v", topicTemplate, err)
			lastErr = err
			result.failed++
		case subscribed:
			result.subscribed++
		default:
			logx.Infof("[mqtt] subscribe skip topic=%s reason=already-subscribed", topicTemplate)
			result.skipped++
		}
	}
	return result, lastErr
}

type restoreSubscriptionsResult struct {
	// total 本次参与恢复的 Topic 模板数，恒等于 subscribed+skipped+failed。
	total      int
	subscribed int
	skipped    int
	failed     int
}

func uniqueTopics(topics []string) []string {
	return slice.Unique(topics)
}

// messageHandler 消息处理包装器
// topicTemplate 参数为触发当前回调的订阅模板（可能含通配符 +/#），dispatcher 使用该模板精确路由。
func (c *mqttClient) messageHandler(topicTemplate string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		c.processMessage(msg, topicTemplate)
	}
}

// processMessage 处理接收到的消息
// topicTemplate 为触发当前 MQTT 回调的订阅模板。
func (c *mqttClient) processMessage(msg mqtt.Message, topicTemplate string) {
	ctx := context.Background()
	payload := msg.Payload()

	// 尝试解析包装消息（用于链路追踪），只调用一次 msg.Payload()
	if wrapped, err := c.tryUnwrapPayload(payload); err == nil {
		payload = wrapped.Payload
		ctx = c.extractTraceContext(ctx, wrapped)
	}

	// 启动追踪 span
	ctx, span := c.startSpan(ctx, msg, topicTemplate)
	defer span.End()
	ctx = logx.ContextWithFields(ctx,
		logx.Field("client", c.GetClientID()),
		logx.Field("topic", msg.Topic()),
		logx.Field("topic_template", topicTemplate),
		logx.Field("payload_bytes", len(payload)),
		logx.Field("payload_size", tool.DecimalBytes(int64(len(payload)), 1)),
	)

	if len(payload) == 0 {
		logx.WithContext(ctx).Error("[mqtt] empty payload")
		return
	}

	// 分发给 handlers
	c.dispatcher.dispatch(ctx, payload, msg.Topic(), topicTemplate)
}

// tryUnwrapPayload 尝试解析包装消息
// 如果消息是 JSON 格式的 Message 结构，会提取其中的 payload
func (c *mqttClient) tryUnwrapPayload(data []byte) (*Message, error) {
	var wrapped Message
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Payload == nil {
		return nil, errors.New("invalid wrapped payload")
	}
	return &wrapped, nil
}

// extractTraceContext 从消息中提取链路追踪上下文
func (c *mqttClient) extractTraceContext(ctx context.Context, msg *Message) context.Context {
	return tracex.Extract(ctx, tracex.NewCarrier(msg.Headers))
}

// startSpan 启动追踪 span
func (c *mqttClient) startSpan(ctx context.Context, msg mqtt.Message, topicTemplate string) (context.Context, oteltrace.Span) {
	ctx, span := c.tracer.Start(ctx, "mqtt-consume",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	span.SetAttributes(
		attribute.String(attrClientID, c.cfg.ClientID),
		attribute.String(attrTopicTemplate, topicTemplate),
		attribute.String(attrMsgID, fmt.Sprintf("%d", msg.MessageID())),
		attribute.Int(attrQoS, int(msg.Qos())),
		attribute.String(attrAction, "consume"),
	)
	return ctx, span
}

// PublishWithTrace 发布消息并注入 OTel 链路追踪上下文。
func (c *mqttClient) PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error) {
	msg := NewMessage(topic, payload)
	tracex.Inject(ctx, tracex.NewCarrier(msg.Headers))
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal trace message failed: %w", err)
	}
	if err := c.Publish(ctx, topic, jsonBytes); err != nil {
		return "", err
	}
	return trace.TraceIDFromContext(ctx), nil
}

// Publish 发布消息到指定主题
func (c *mqttClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if !c.client.IsConnected() {
		return errors.New("cannot publish: client not connected")
	}

	timeout := time.Duration(c.cfg.Timeout) * time.Millisecond
	token := c.client.Publish(topic, c.qos, false, payload)

	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("publish timeout after %v", timeout)
	}
	return token.Error()
}

// Close 关闭 MQTT 连接
func (c *mqttClient) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		c.client.Disconnect(250)
	}
	c.handlerMgr.closeReplyHandlers()
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] connection closed")
}

// GetClientID 获取客户端 ID
func (c *mqttClient) GetClientID() string {
	return c.cfg.ClientID
}
