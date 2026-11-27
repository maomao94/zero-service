package mqttx

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
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
	// Publish 发布消息。ctx 携带有效 span 时自动把 traceparent 内联进 JSON payload 的 headers 字段，
	// 与 Kafka 的 record header 语义对齐，调用方无需区分是否跟踪。
	Publish(ctx context.Context, topic string, payload []byte) error
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
// 首次连接时调用方通常紧接着 AddHandler 注册 Topic，订阅动作因此分散在
// 两个 goroutine 中，任何“总数”口径都只是某一瞬间的快照，没有参考价值，
// 所以汇总只保留重连恢复结果；每个 Topic 的真实订阅结果由 subscribe 日志体现。
func (c *mqttClient) onConnect(_ mqtt.Client) {
	firstConnect := c.ready.CompareAndSwap(false, true)
	logx.Infof("[mqtt] connected client=%s phase=%s", c.cfg.ClientID, connectPhase(firstConnect))

	// 首次连接执行 onReady 回调
	if firstConnect && c.onReady != nil {
		c.onReady(c)
	}

	result, err := c.restoreSubscriptions()
	if firstConnect {
		// 首连时没有历史订阅可恢复，失败明细已按 topic 打印，这里不再汇总。
		return
	}
	switch {
	case err != nil:
		logx.Errorf("[mqtt] restore subscriptions failed client=%s restored=%d failed=%d err=%v",
			c.cfg.ClientID, result.subscribed+result.skipped, result.failed, err)
	case result.total > 0:
		logx.Infof("[mqtt] restore subscriptions done client=%s", c.cfg.ClientID)
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
	// paho 不会把 SUBACK 的失败原因码（0x80）放进 token.Error()，只能从授予结果里判断；
	// 不判断会把 Broker 拒绝订阅当成成功，之后一条报文都收不到却没有任何错误日志。
	if subToken, ok := token.(*mqtt.SubscribeToken); ok {
		if err := subscribeGrantedError(topicTemplate, c.qos, subToken.Result()); err != nil {
			return false, err
		}
	}

	c.subscribed[topicTemplate] = struct{}{}
	logx.Infof("[mqtt] subscribe ok topic=%s", topicTemplate)
	return true, nil
}

// subscribeGrantedError 校验 SUBACK 授予结果，0x80 表示 Broker 拒绝订阅。
// MQTT 3.1.1 只有 0x80 表示失败；MQTT 5.0 的失败原因码（0x80~0xA2）均 >= 0x80，统一按阈值判断。
// 一次 SUBSCRIBE 可包含多个 topic filter（SubscribeMultiple），这里汇总所有被拒 topic 再返回，
// 排序保证日志顺序稳定；授予 QoS 低于请求 QoS 属于降级而非失败，只记录日志。
// 空结果或结果里缺少本次订阅的 topic，说明 SUBACK 异常，按失败处理，避免静默当成订阅成功。
func subscribeGrantedError(topic string, requested byte, result map[string]byte) error {
	if len(result) == 0 {
		return fmt.Errorf("subscribe rejected by broker: empty SUBACK result")
	}

	topics := make([]string, 0, len(result))
	for grantedTopic := range result {
		topics = append(topics, grantedTopic)
	}
	slices.Sort(topics)

	var failures, downgraded []string
	for _, grantedTopic := range topics {
		granted := result[grantedTopic]
		if granted >= 0x80 {
			failures = append(failures, fmt.Sprintf("topic=%s granted=%#x", grantedTopic, granted))
			continue
		}
		if granted < requested {
			downgraded = append(downgraded, fmt.Sprintf("topic=%s requested=%d granted=%d", grantedTopic, requested, granted))
		}
	}
	if _, ok := result[topic]; !ok {
		failures = append(failures, fmt.Sprintf("topic=%s missing from SUBACK", topic))
	}

	if len(downgraded) > 0 {
		logx.Infof("[mqtt] subscribe downgraded: %s", strings.Join(downgraded, "; "))
	}
	if len(failures) == 0 {
		return nil
	}
	// 单行分隔而不是 errors.Join：go-zero 日志按行切分，错误信息里带换行会把一条日志拆成多行。
	return fmt.Errorf("subscribe rejected by broker: %s", strings.Join(failures, "; "))
}

// isSubscribed 检查是否已订阅
func (c *mqttClient) isSubscribed(topicTemplate string) bool {
	_, exists := c.subscribed[topicTemplate]
	return exists
}

// restoreSubscriptions 恢复所有订阅（连接重连后）。
//
// 只负责把订阅补回来并把失败放大成 error；每个 Topic 订阅成功会单独打 info，
// 已订阅跳过属于去重细节，只在 debug 输出。
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
			logx.Debugf("[mqtt] subscribe skipped topic=%s reason=already-subscribed", topicTemplate)
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

	// trace 头内联在 payload 顶层 headers 字段，与 Kafka record header 语义一致
	ctx = c.extractInlineTraceContext(ctx, payload)

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

// extractInlineTraceContext 从 payload 顶层的 headers 字段恢复链路上下文。
// Publish 自动注入写入的就是该字段；解析失败或无 headers 时原样返回。
func (c *mqttClient) extractInlineTraceContext(ctx context.Context, payload []byte) context.Context {
	// 快速路径：绝大多数消息不带 trace 头（如业务自发的点位数据），不必解析
	if !bytes.Contains(payload, traceparentMarker) {
		return ctx
	}

	var envelope struct {
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || len(envelope.Headers) == 0 {
		return ctx
	}
	return tracex.Extract(ctx, tracex.NewCarrier(envelope.Headers))
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

// traceparentMarker 用于快速判断 payload 是否已携带 trace 头（ASDU 每条点位报文都自带）。
var traceparentMarker = []byte(`"traceparent"`)

// injectTraceHeaders 在 payload 是 JSON 对象且尚未携带 trace 头时，把 traceparent 内联到顶层 headers。
// MQTT 3.1.1 没有自定义 header 位，这是与 Kafka record header 对齐的等价载体：
// 只在末尾追加字段，原有字节原样保留。
//
// 过滤顺序按成本递增：无 span → 非对象形态 → 已含 traceparent → JSON 结构探测。
// 已含 traceparent 的判定是字面量扫描，可能被业务数据里的同名文本误命中断而放弃注入，
// 这是为高频 ASDU 保留的零分配快速路径。
func injectTraceHeaders(ctx context.Context, payload []byte) ([]byte, bool) {
	if ctx == nil {
		return nil, false
	}
	// 绝大多数消息没有 span，先挡掉，避免任何解析开销
	if !oteltrace.SpanContextFromContext(ctx).IsValid() {
		return nil, false
	}

	// 便宜的形态判断：不是 JSON 对象（二进制、数组、标量、截断）直接原样发送
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil, false
	}

	// 快速路径：已带 traceparent 的消息（如 ASDU）跳过解析，避免逐字段扫描
	if bytes.Contains(trimmed, traceparentMarker) {
		return nil, false
	}

	// 一次遍历同时完成两件事：payload 是否完整合法、顶层是否已占用 headers
	var probe struct {
		Headers json.RawMessage `json:"headers"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		// 非法 JSON：不追加任何字段，坏数据原样发出，便于上游排查
		return nil, false
	}
	// 业务已自带 headers（无论类型是否合法），不覆盖也不合并，避免破坏业务字段
	if probe.Headers != nil {
		return nil, false
	}

	headers := make(map[string]string)
	tracex.Inject(ctx, tracex.NewCarrier(headers))
	return appendJSONHeaders(trimmed, headers)
}

// appendJSONHeaders 在 JSON 对象末尾追加 headers 字段，原有字节原样保留。
// payload 不是 JSON 对象（二进制、数组、标量）时返回 false。
func appendJSONHeaders(payload []byte, headers map[string]string) ([]byte, bool) {
	if len(headers) == 0 {
		return nil, false
	}

	trimmed := bytes.TrimRight(payload, " \t\r\n")
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return nil, false
	}

	encoded, err := json.Marshal(headers)
	if err != nil {
		return nil, false
	}

	out := make([]byte, 0, len(trimmed)+len(encoded)+16)
	out = append(out, trimmed[:len(trimmed)-1]...)
	if len(trimmed) > 2 {
		out = append(out, ',')
	}
	out = append(out, `"headers":`...)
	out = append(out, encoded...)
	out = append(out, '}')
	return out, true
}

// Publish 发布消息到指定主题。
// ctx 携带有效 span 时自动注入 trace 头（见 injectTraceHeaders），无 span 或 payload 不可注入时原样发送。
func (c *mqttClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if !c.client.IsConnected() {
		return errors.New("cannot publish: client not connected")
	}

	if traced, ok := injectTraceHeaders(ctx, payload); ok {
		payload = traced
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
