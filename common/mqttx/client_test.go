package mqttx

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/stat"
	"go.opentelemetry.io/otel"
)

// fakeToken 是 paho mqtt.Token 的测试实现，用于模拟超时与错误。
type fakeToken struct {
	err     error
	expired bool
}

func (t *fakeToken) Wait() bool { return true }

func (t *fakeToken) WaitTimeout(_ time.Duration) bool { return !t.expired }

func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (t *fakeToken) Error() error { return t.err }

type fakePublish struct {
	topic    string
	payload  []byte
	qos      byte
	retained bool
}

// fakePahoClient 是 paho mqtt.Client 的测试实现：记录订阅/发布并可按 topic 注入错误。
type fakePahoClient struct {
	mu sync.Mutex

	connected      bool
	connectToken   *fakeToken
	publishToken   *fakeToken
	subscribeToken *fakeToken

	subscribeErrs map[string]error
	subscribes    []string
	callbacks     map[string]mqtt.MessageHandler
	publishes     []fakePublish
	disconnects   int
}

func newFakePahoClient(connected bool) *fakePahoClient {
	return &fakePahoClient{
		connected:     connected,
		subscribeErrs: make(map[string]error),
		callbacks:     make(map[string]mqtt.MessageHandler),
	}
}

func (c *fakePahoClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *fakePahoClient) IsConnectionOpen() bool { return c.IsConnected() }

func (c *fakePahoClient) Connect() mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = true
	if c.connectToken != nil {
		return c.connectToken
	}
	return &fakeToken{}
}

func (c *fakePahoClient) Disconnect(_ uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnects++
	c.connected = false
}

func (c *fakePahoClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, _ := payload.([]byte)
	c.publishes = append(c.publishes, fakePublish{
		topic:    topic,
		payload:  append([]byte(nil), data...),
		qos:      qos,
		retained: retained,
	})
	if c.publishToken != nil {
		return c.publishToken
	}
	return &fakeToken{}
}

func (c *fakePahoClient) Subscribe(topic string, _ byte, callback mqtt.MessageHandler) mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribes = append(c.subscribes, topic)
	if callback != nil {
		c.callbacks[topic] = callback
	}
	if err := c.subscribeErrs[topic]; err != nil {
		return &fakeToken{err: err}
	}
	if c.subscribeToken != nil {
		return c.subscribeToken
	}
	return &fakeToken{}
}

func (c *fakePahoClient) SubscribeMultiple(_ map[string]byte, _ mqtt.MessageHandler) mqtt.Token {
	return &fakeToken{}
}

func (c *fakePahoClient) Unsubscribe(_ ...string) mqtt.Token { return &fakeToken{} }

func (c *fakePahoClient) AddRoute(_ string, _ mqtt.MessageHandler) {}

func (c *fakePahoClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(mqtt.NewClientOptions())
}

func (c *fakePahoClient) snapshot() ([]string, []fakePublish) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.subscribes...), append([]fakePublish(nil), c.publishes...)
}

func (c *fakePahoClient) callback(topic string) mqtt.MessageHandler {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.callbacks[topic]
}

func (c *fakePahoClient) disconnectCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disconnects
}

// fakeMessage 是 paho mqtt.Message 的测试实现。
type fakeMessage struct {
	topic     string
	payload   []byte
	qos       byte
	messageID uint16
	duplicate bool
	retained  bool
}

func (m *fakeMessage) Duplicate() bool   { return m.duplicate }
func (m *fakeMessage) Qos() byte         { return m.qos }
func (m *fakeMessage) Retained() bool    { return m.retained }
func (m *fakeMessage) Topic() string     { return m.topic }
func (m *fakeMessage) MessageID() uint16 { return m.messageID }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              {}

func newTestMqttClient(paho mqtt.Client) *mqttClient {
	cfg := MqttConfig{ClientID: "mqttx-test-client", Timeout: 100, Qos: 1}
	adjustConfig(&cfg)
	manager := newHandlerManager()
	metrics := stat.NewMetrics("mqttx-client-test")
	return &mqttClient{
		client:     paho,
		cfg:        cfg,
		handlerMgr: manager,
		dispatcher: newMessageDispatcher(manager, metrics),
		subscribed: make(map[string]struct{}),
		qos:        cfg.Qos,
		tracer:     otel.Tracer("mqttx-test"),
		metrics:    metrics,
	}
}

func TestAddReplyRouterRegistersHandlerAndSubscribesOnce(t *testing.T) {
	paho := newFakePahoClient(true)
	c := newTestMqttClient(paho)

	router := newTestReplyRouter(func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()

	if err := c.AddReplyRouter("reply/+", router); err != nil {
		t.Fatalf("AddReplyRouter returned error: %v", err)
	}
	if got := c.handlerMgr.getReplyHandler("reply/+"); got != router {
		t.Fatalf("registered reply handler = %v, want the provided router", got)
	}
	if !c.isSubscribed("reply/+") {
		t.Fatal("expected reply topic to be subscribed")
	}

	replacement := newTestReplyRouter(func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "replacement"}, nil
	})
	defer replacement.Close()
	if err := c.AddReplyRouter("reply/+", replacement); err != nil {
		t.Fatalf("second AddReplyRouter returned error: %v", err)
	}
	if got := c.handlerMgr.getReplyHandler("reply/+"); got != replacement {
		t.Fatal("expected the later reply router to replace the previous one")
	}

	topics, _ := paho.snapshot()
	if !reflect.DeepEqual(topics, []string{"reply/+"}) {
		t.Fatalf("subscribe calls = %v, want a single subscription", topics)
	}
}

func TestAddReplyRouterRejectsNilRouter(t *testing.T) {
	c := newTestMqttClient(newFakePahoClient(true))

	err := c.AddReplyRouter("reply/+", nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("AddReplyRouter(nil) error = %v, want nil-router error", err)
	}
	if got := c.handlerMgr.getReplyHandler("reply/+"); got != nil {
		t.Fatalf("nil router must not be registered, got %v", got)
	}
}

func TestAddHandlerRejectsNilHandler(t *testing.T) {
	c := newTestMqttClient(newFakePahoClient(true))

	err := c.AddHandler("device/+", nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("AddHandler(nil) error = %v, want nil-handler error", err)
	}
}

func TestAddReplyRouterDefersSubscriptionUntilConnected(t *testing.T) {
	paho := newFakePahoClient(false)
	c := newTestMqttClient(paho)

	router := newTestReplyRouter(func(context.Context, []byte, string, string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: "tid-1", Value: "ok"}, nil
	})
	defer router.Close()

	if err := c.AddReplyRouter("reply/+", router); err != nil {
		t.Fatalf("AddReplyRouter returned error: %v", err)
	}
	topics, _ := paho.snapshot()
	if len(topics) != 0 {
		t.Fatalf("expected no subscription while disconnected, got %v", topics)
	}
	if c.isSubscribed("reply/+") {
		t.Fatal("topic must not be marked subscribed while disconnected")
	}

	c.onConnect(nil)
	topics, _ = paho.snapshot()
	if !reflect.DeepEqual(topics, []string{"reply/+"}) {
		t.Fatalf("subscribe calls after connect = %v, want [reply/+]", topics)
	}
	if !c.isSubscribed("reply/+") {
		t.Fatal("expected topic to be subscribed after connect")
	}
}

func TestAddHandlerSubscribesAndPropagatesErrors(t *testing.T) {
	t.Run("subscribe error", func(t *testing.T) {
		paho := newFakePahoClient(true)
		subscribeErr := errors.New("subscribe failed")
		paho.subscribeErrs["device/+"] = subscribeErr
		c := newTestMqttClient(paho)

		err := c.AddHandlerFunc("device/+", func(context.Context, []byte, string, string) error { return nil })
		if !errors.Is(err, subscribeErr) {
			t.Fatalf("AddHandlerFunc error = %v, want subscribe error", err)
		}
		if c.isSubscribed("device/+") {
			t.Fatal("failed subscription must not be recorded")
		}
		if len(c.handlerMgr.getHandlers("device/+")) != 1 {
			t.Fatal("expected handler to be registered even when subscribe fails")
		}
	})

	t.Run("subscribe timeout", func(t *testing.T) {
		paho := newFakePahoClient(true)
		paho.subscribeToken = &fakeToken{expired: true}
		c := newTestMqttClient(paho)

		err := c.AddHandlerFunc("device/+", func(context.Context, []byte, string, string) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "subscribe timeout") {
			t.Fatalf("AddHandlerFunc error = %v, want subscribe timeout", err)
		}
		if c.isSubscribed("device/+") {
			t.Fatal("timed-out subscription must not be recorded")
		}
	})

	t.Run("already subscribed", func(t *testing.T) {
		paho := newFakePahoClient(true)
		c := newTestMqttClient(paho)

		for i := 0; i < 2; i++ {
			if err := c.AddHandlerFunc("device/+", func(context.Context, []byte, string, string) error { return nil }); err != nil {
				t.Fatalf("AddHandlerFunc returned error: %v", err)
			}
		}
		topics, _ := paho.snapshot()
		if !reflect.DeepEqual(topics, []string{"device/+"}) {
			t.Fatalf("subscribe calls = %v, want one subscription", topics)
		}
		if got := len(c.handlerMgr.getHandlers("device/+")); got != 2 {
			t.Fatalf("registered handlers = %d, want 2", got)
		}
	})
}

func TestOnConnectRestoresSubscriptionsAndCallsOnReadyOnce(t *testing.T) {
	paho := newFakePahoClient(false)
	c := newTestMqttClient(paho)
	c.cfg.SubscribeTopics = []string{"config/+"}
	readyCalls := 0
	c.onReady = func(Client) { readyCalls++ }
	c.handlerMgr.addHandler("device/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))

	c.onConnect(nil)
	c.onConnect(nil)

	if readyCalls != 1 {
		t.Fatalf("onReady calls = %d, want 1", readyCalls)
	}
	topics, _ := paho.snapshot()
	sort.Strings(topics)
	expected := []string{"config/+", "device/+"}
	if !reflect.DeepEqual(topics, expected) {
		t.Fatalf("subscribe calls = %v, want %v", topics, expected)
	}

	c.onConnectionLost(nil, errors.New("connection lost"))
	if c.isSubscribed("device/+") || c.isSubscribed("config/+") {
		t.Fatal("expected subscriptions to be reset after connection loss")
	}

	c.onConnect(nil)
	topics, _ = paho.snapshot()
	if len(topics) != 4 {
		t.Fatalf("subscribe calls after reconnect = %v, want 4 (2 initial + 2 restored)", topics)
	}
}

func TestRestoreSubscriptionsReportsPartialFailure(t *testing.T) {
	paho := newFakePahoClient(true)
	subscribeErr := errors.New("subscribe failed")
	paho.subscribeErrs["bad/+"] = subscribeErr
	c := newTestMqttClient(paho)
	c.handlerMgr.addHandler("good/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))
	c.handlerMgr.addHandler("bad/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
		return nil
	}))

	result, err := c.restoreSubscriptions()
	if !errors.Is(err, subscribeErr) {
		t.Fatalf("restoreSubscriptions error = %v, want subscribe error", err)
	}
	if result.subscribed != 1 || result.skipped != 0 {
		t.Fatalf("restore result = %+v, want subscribed=1 skipped=0", result)
	}
	if c.isSubscribed("bad/+") {
		t.Fatal("failed topic must not be marked subscribed")
	}
}

type closeCountingHandler struct {
	closed int
}

func (h *closeCountingHandler) Consume(context.Context, []byte, string, string) error { return nil }

func (h *closeCountingHandler) Close() { h.closed++ }

func TestCloseDisconnectsOnceAndClosesReplyHandlers(t *testing.T) {
	paho := newFakePahoClient(true)
	c := newTestMqttClient(paho)
	handler := &closeCountingHandler{}
	if err := c.AddReplyRouter("reply/+", handler); err != nil {
		t.Fatalf("AddReplyRouter returned error: %v", err)
	}

	c.Close()
	c.Close()

	if got := paho.disconnectCount(); got != 1 {
		t.Fatalf("disconnect calls = %d, want 1", got)
	}
	if handler.closed != 1 {
		t.Fatalf("reply handler Close calls = %d, want 1", handler.closed)
	}
	if c.isSubscribed("reply/+") {
		t.Fatal("expected subscriptions to be cleared on close")
	}
}

func TestPublishGuardsConnectionAndReportsTokenErrors(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		c := newTestMqttClient(newFakePahoClient(false))
		err := c.Publish(context.Background(), "device/1", []byte("payload"))
		if err == nil || !strings.Contains(err.Error(), "client not connected") {
			t.Fatalf("Publish error = %v, want not-connected error", err)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		paho := newFakePahoClient(true)
		publishErr := errors.New("publish failed")
		paho.publishToken = &fakeToken{err: publishErr}
		c := newTestMqttClient(paho)

		err := c.Publish(context.Background(), "device/1", []byte("payload"))
		if !errors.Is(err, publishErr) {
			t.Fatalf("Publish error = %v, want publish error", err)
		}
	})

	t.Run("publish timeout", func(t *testing.T) {
		paho := newFakePahoClient(true)
		paho.publishToken = &fakeToken{expired: true}
		c := newTestMqttClient(paho)

		err := c.Publish(context.Background(), "device/1", []byte("payload"))
		if err == nil || !strings.Contains(err.Error(), "publish timeout") {
			t.Fatalf("Publish error = %v, want publish timeout", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		paho := newFakePahoClient(true)
		c := newTestMqttClient(paho)

		if err := c.Publish(context.Background(), "device/1", []byte("payload")); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		_, publishes := paho.snapshot()
		if len(publishes) != 1 {
			t.Fatalf("publish calls = %d, want 1", len(publishes))
		}
		got := publishes[0]
		if got.topic != "device/1" || string(got.payload) != "payload" || got.qos != 1 || got.retained {
			t.Fatalf("unexpected publish call: %+v", got)
		}
	})
}

func TestPublishWithTraceWrapsPayload(t *testing.T) {
	paho := newFakePahoClient(true)
	c := newTestMqttClient(paho)

	if _, err := c.PublishWithTrace(context.Background(), "device/1", []byte("hello")); err != nil {
		t.Fatalf("PublishWithTrace returned error: %v", err)
	}

	_, publishes := paho.snapshot()
	if len(publishes) != 1 {
		t.Fatalf("publish calls = %d, want 1", len(publishes))
	}
	var wrapped Message
	if err := json.Unmarshal(publishes[0].payload, &wrapped); err != nil {
		t.Fatalf("published payload is not a wrapped Message: %v", err)
	}
	if wrapped.Topic != "device/1" || string(wrapped.Payload) != "hello" {
		t.Fatalf("unexpected wrapped message: %+v", wrapped)
	}

	t.Run("publish error", func(t *testing.T) {
		paho := newFakePahoClient(true)
		publishErr := errors.New("publish failed")
		paho.publishToken = &fakeToken{err: publishErr}
		c := newTestMqttClient(paho)

		if _, err := c.PublishWithTrace(context.Background(), "device/1", []byte("hello")); !errors.Is(err, publishErr) {
			t.Fatalf("PublishWithTrace error = %v, want publish error", err)
		}
	})
}

func TestProcessMessageDispatchesUnwrappedPayload(t *testing.T) {
	paho := newFakePahoClient(true)
	c := newTestMqttClient(paho)
	var gotPayload []byte
	var gotTopic, gotTemplate string
	c.handlerMgr.addHandler("device/+", ConsumeHandlerFunc(func(_ context.Context, payload []byte, topic string, topicTemplate string) error {
		gotPayload = append([]byte(nil), payload...)
		gotTopic = topic
		gotTemplate = topicTemplate
		return nil
	}))

	wrapped, err := json.Marshal(&Message{
		Topic:   "device/1",
		Payload: []byte("inner"),
		Headers: map[string]string{"traceparent": "00-trace-span-01"},
	})
	if err != nil {
		t.Fatalf("marshal wrapped message: %v", err)
	}
	c.processMessage(&fakeMessage{topic: "device/1", payload: wrapped, qos: 1, messageID: 7}, "device/+")

	if string(gotPayload) != "inner" {
		t.Fatalf("handler payload = %q, want inner payload", gotPayload)
	}
	if gotTopic != "device/1" || gotTemplate != "device/+" {
		t.Fatalf("handler topic args = (%q, %q), want (device/1, device/+)", gotTopic, gotTemplate)
	}

	t.Run("raw payload passes through", func(t *testing.T) {
		called := false
		c.handlerMgr.addHandler("raw/+", ConsumeHandlerFunc(func(_ context.Context, payload []byte, _ string, _ string) error {
			called = true
			if string(payload) != "plain" {
				t.Fatalf("handler payload = %q, want plain", payload)
			}
			return nil
		}))
		c.processMessage(&fakeMessage{topic: "raw/1", payload: []byte("plain")}, "raw/+")
		if !called {
			t.Fatal("expected raw handler to be called")
		}
	})

	t.Run("empty payload is dropped", func(t *testing.T) {
		called := false
		c.handlerMgr.addHandler("empty/+", ConsumeHandlerFunc(func(context.Context, []byte, string, string) error {
			called = true
			return nil
		}))
		c.processMessage(&fakeMessage{topic: "empty/1", payload: nil}, "empty/+")
		if called {
			t.Fatal("empty payload must not reach handlers")
		}
	})
}

func TestReplyRouterResolvesThroughSubscribedCallback(t *testing.T) {
	paho := newFakePahoClient(true)
	c := newTestMqttClient(paho)
	router := newTestReplyRouter(func(_ context.Context, payload []byte, _ string, _ string) (ReplyMessage[string], error) {
		return ReplyMessage[string]{Tid: string(payload), Value: "ok"}, nil
	})
	defer router.Close()
	if err := c.AddReplyRouter("reply/+", router); err != nil {
		t.Fatalf("AddReplyRouter returned error: %v", err)
	}
	callback := paho.callback("reply/+")
	if callback == nil {
		t.Fatal("expected subscription callback for reply/+")
	}

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := router.RequestReply(context.Background(), "tid-1", func() error { return nil })
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()
	waitUntil(t, func() bool { return router.has("tid-1") })

	callback(nil, &fakeMessage{topic: "reply/1", payload: []byte("tid-1")})

	select {
	case result := <-resultCh:
		if result != "ok" {
			t.Fatalf("RequestReply result = %q, want ok", result)
		}
	case err := <-errCh:
		t.Fatalf("RequestReply returned error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reply")
	}
}

func TestTryUnwrapPayload(t *testing.T) {
	c := &mqttClient{}

	wrapped, err := json.Marshal(&Message{Topic: "device/1", Payload: []byte("inner")})
	if err != nil {
		t.Fatalf("marshal wrapped message: %v", err)
	}
	msg, err := c.tryUnwrapPayload(wrapped)
	if err != nil {
		t.Fatalf("tryUnwrapPayload returned error: %v", err)
	}
	if msg.Topic != "device/1" || string(msg.Payload) != "inner" {
		t.Fatalf("unexpected unwrapped message: %+v", msg)
	}

	if _, err := c.tryUnwrapPayload([]byte("not-json")); err == nil {
		t.Fatal("expected error for raw payload")
	}
	if _, err := c.tryUnwrapPayload([]byte(`{"topic":"device/1"}`)); err == nil {
		t.Fatal("expected error for message without payload")
	}
}

func TestGetClientID(t *testing.T) {
	c := newTestMqttClient(newFakePahoClient(false))
	if got := c.GetClientID(); got != "mqttx-test-client" {
		t.Fatalf("GetClientID = %q, want mqttx-test-client", got)
	}
}

func TestNewClientWithoutBrokerReturnsError(t *testing.T) {
	client, err := NewClient(MqttConfig{})
	if err == nil || !strings.Contains(err.Error(), "no broker") {
		t.Fatalf("NewClient error = %v, want no-broker error", err)
	}
	if client != nil {
		t.Fatalf("NewClient client = %v, want nil", client)
	}
}

func TestNewClientFailsFastWhenBrokerUnreachable(t *testing.T) {
	start := time.Now()
	client, err := NewClient(MqttConfig{
		Broker:  []string{"tcp://192.0.2.1:1883"},
		Timeout: 50,
	})
	if err == nil {
		client.Close()
		t.Fatal("NewClient must fail for an unreachable broker")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("NewClient took %v, want a fast failure", elapsed)
	}
}

func TestConnectReportsTimeoutAndConnectError(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		paho := newFakePahoClient(false)
		paho.connectToken = &fakeToken{expired: true}
		c := newTestMqttClient(paho)

		err := c.connect()
		if err == nil || !strings.Contains(err.Error(), "connect timeout") {
			t.Fatalf("connect error = %v, want connect timeout", err)
		}
	})

	t.Run("connect error", func(t *testing.T) {
		paho := newFakePahoClient(false)
		connectErr := errors.New("connection refused")
		paho.connectToken = &fakeToken{err: connectErr}
		c := newTestMqttClient(paho)

		err := c.connect()
		if !errors.Is(err, connectErr) {
			t.Fatalf("connect error = %v, want connection refused", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		paho := newFakePahoClient(false)
		c := newTestMqttClient(paho)

		if err := c.connect(); err != nil {
			t.Fatalf("connect returned error: %v", err)
		}
	})
}

func TestCreateMqttClientAppliesConfig(t *testing.T) {
	c := newTestMqttClient(nil)
	c.cfg = MqttConfig{
		Broker:    []string{"tcp://127.0.0.1:1883"},
		ClientID:  "cfg-client",
		Username:  "user",
		Password:  "secret",
		Timeout:   1500,
		KeepAlive: 3000,
	}

	paho := c.createMqttClient()
	reader := paho.OptionsReader()
	if got := reader.ClientID(); got != "cfg-client" {
		t.Fatalf("client id = %q, want cfg-client", got)
	}
	if got := reader.Username(); got != "user" {
		t.Fatalf("username = %q, want user", got)
	}
	if got := reader.Password(); got != "secret" {
		t.Fatalf("password = %q, want secret", got)
	}
	if got := reader.KeepAlive(); got != 3*time.Second {
		t.Fatalf("keep alive = %v, want 3s", got)
	}
	if got := reader.ConnectTimeout(); got != 1500*time.Millisecond {
		t.Fatalf("connect timeout = %v, want 1500ms", got)
	}
	if !reader.AutoReconnect() {
		t.Fatal("expected auto reconnect to be enabled")
	}
}

func TestDefaultHandlerDoesNotPanic(t *testing.T) {
	c := newTestMqttClient(nil)
	c.defaultHandler(nil, &fakeMessage{topic: "device/1", payload: []byte("{}")})
}
