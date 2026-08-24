package broadcast

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"zero-service/common/antsx"
	"zero-service/common/mqttx"
)

const (
	testPrefix    = "iec"
	testInstance  = "iec-caller-uid"
	testBroadcast = "m"
)

type publishedMessage struct {
	topic   string
	payload []byte
}

// mockMqttClient 手写 mqttx.Client mock：记录发布与 handler 注册。
type mockMqttClient struct {
	mu         sync.Mutex
	handlers   map[string]func(context.Context, []byte, string, string) error
	published  []publishedMessage
	publishErr error
}

func newMockMqttClient() *mockMqttClient {
	return &mockMqttClient{handlers: make(map[string]func(context.Context, []byte, string, string) error)}
}

func (m *mockMqttClient) AddHandler(topicTemplate string, handler mqttx.ConsumeHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topicTemplate] = handler.Consume
	return nil
}

func (m *mockMqttClient) AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topicTemplate] = fn
	return nil
}

func (m *mockMqttClient) Publish(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (m *mockMqttClient) PublishWithTrace(_ context.Context, topic string, payload []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, publishedMessage{topic: topic, payload: payload})
	return "mock-trace-id", m.publishErr
}

func (m *mockMqttClient) Close() {}

func (m *mockMqttClient) GetClientID() string { return "mock-client" }

func (m *mockMqttClient) lastPublished() (publishedMessage, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.published) == 0 {
		return publishedMessage{}, false
	}
	return m.published[len(m.published)-1], true
}

func (m *mockMqttClient) publishedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.published)
}

func (m *mockMqttClient) hasHandler(topicTemplate string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.handlers[topicTemplate]
	return ok
}

func bodyJSON(t *testing.T, body *BroadcastBody) []byte {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return data
}

func ackOf(t *testing.T, msg publishedMessage) *BroadcastAckBody {
	t.Helper()
	ack := &BroadcastAckBody{}
	if err := json.Unmarshal(msg.payload, ack); err != nil {
		t.Fatalf("unmarshal ack payload %q: %v", msg.payload, err)
	}
	return ack
}

func TestConsumeBroadcastLoopbackIgnored(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: BroadcastAckTopic(testPrefix, testInstance), Method: testBroadcast,
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}
	if mock.publishedCount() != 0 {
		t.Fatalf("loopback message must not be acked, published=%d", mock.publishedCount())
	}
}

func TestConsumeBroadcastUnknownMethodAcksFailure(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	reqTopic := "some/other/node/ack"
	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: reqTopic, Method: "no.such.method",
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}

	msg, ok := mock.lastPublished()
	if !ok {
		t.Fatal("expected ack to be published")
	}
	if msg.topic != reqTopic {
		t.Fatalf("ack topic = %q, want %q", msg.topic, reqTopic)
	}
	ack := ackOf(t, msg)
	if ack.Success || ack.Tid != "t1" || ack.Method != "no.such.method" {
		t.Fatalf("unexpected ack: %+v", ack)
	}
	if ack.Error != "unknown method" || ack.ErrorKind != KindUnknown {
		t.Fatalf("unexpected unknown-method ack: %+v", ack)
	}
}

func TestConsumeBroadcastSuccessDefaultAck(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
	b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
		return nil, nil
	})

	reqTopic := BroadcastAckTopic("iec", "other-node")
	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: reqTopic, Method: testBroadcast,
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}
	ack := ackOf(t, mustLast(t, mock))
	if !ack.Success || ack.Tid != "t1" || ack.Method != testBroadcast || ack.ResponseBody != "" {
		t.Fatalf("unexpected success ack: %+v", ack)
	}
}

func TestConsumeBroadcastSuccessCarriesResultBytes(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
	b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
		return []byte(`{"value":true}`), nil
	})

	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: "iec/other/ack", Method: testBroadcast,
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}
	ack := ackOf(t, mustLast(t, mock))
	if !ack.Success || ack.ResponseBody != `{"value":true}` || ack.Tid != "t1" {
		t.Fatalf("unexpected ack: %+v", ack)
	}
}

func TestConsumeBroadcastDeliversMethodAndPayload(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
	gotMethod := ""
	gotPayload := ""
	b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
		gotMethod = method
		gotPayload = string(payload)
		return nil, nil
	})

	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: "iec/other/ack", Method: testBroadcast,
		Body: `{"host":"1.2.3.4","port":2404,"value":true}`,
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}
	if gotMethod != testBroadcast {
		t.Fatalf("executor received method %q, want %q", gotMethod, testBroadcast)
	}
	if gotPayload != `{"host":"1.2.3.4","port":2404,"value":true}` {
		t.Fatalf("executor received payload %q, want body bytes", gotPayload)
	}
}

func TestConsumeBroadcastFailureNormalizesKind(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		mock := newMockMqttClient()
		b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
		b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
			return nil, errors.New("boom")
		})
		if err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
			Tid: "t1", AckTopic: "iec/other/ack", Method: testBroadcast,
		}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix)); err != nil {
			t.Fatalf("ConsumeBroadcast returned error: %v", err)
		}
		ack := ackOf(t, mustLast(t, mock))
		if ack.Success || ack.Error != "boom" || ack.ErrorKind != KindUnknown {
			t.Fatalf("unexpected ack: %+v", ack)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		mock := newMockMqttClient()
		b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
		b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
			return nil, antsx.ErrReplyExpired
		})
		if err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
			Tid: "t1", AckTopic: "iec/other/ack", Method: testBroadcast,
		}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix)); err != nil {
			t.Fatalf("ConsumeBroadcast returned error: %v", err)
		}
		ack := ackOf(t, mustLast(t, mock))
		if ack.Success || ack.ErrorKind != KindTimeout {
			t.Fatalf("unexpected ack: %+v", ack)
		}
	})
}

func TestConsumeBroadcastErrSkipAckNoPublish(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))
	b.AddExecutor(testBroadcast, func(ctx context.Context, method string, payload []byte) ([]byte, error) {
		return nil, ErrSkipAck
	})

	err := b.ConsumeBroadcast(context.Background(), bodyJSON(t, &BroadcastBody{
		Tid: "t1", AckTopic: "iec/other/ack", Method: testBroadcast,
	}), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err != nil {
		t.Fatalf("ConsumeBroadcast returned error: %v", err)
	}
	if mock.publishedCount() != 0 {
		t.Fatalf("ErrSkipAck must not publish ack, published=%d", mock.publishedCount())
	}
}

func TestConsumeBroadcastInvalidPayloadReturnsError(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	err := b.ConsumeBroadcast(context.Background(), []byte(`{`), BroadcastTopic(testPrefix), BroadcastTopicPattern(testPrefix))
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if mock.publishedCount() != 0 {
		t.Fatalf("invalid payload must not publish ack, published=%d", mock.publishedCount())
	}
}

func TestBroadcastFireAndForgetBuildsProtocolBody(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	if err := b.Broadcast(context.Background(), testBroadcast, []byte(`{"taskId":"abc"}`)); err != nil {
		t.Fatalf("Broadcast returned error: %v", err)
	}

	msg, ok := mock.lastPublished()
	if !ok {
		t.Fatal("expected publish")
	}
	if msg.topic != BroadcastTopic(testPrefix) {
		t.Fatalf("topic = %q, want %q", msg.topic, BroadcastTopic(testPrefix))
	}
	sent := &BroadcastBody{}
	if err := json.Unmarshal(msg.payload, sent); err != nil {
		t.Fatalf("unmarshal sent body: %v", err)
	}
	if sent.Tid == "" {
		t.Fatalf("tid must be auto-generated, got empty")
	}
	if sent.Method != testBroadcast {
		t.Fatalf("method = %q, want %q", sent.Method, testBroadcast)
	}
	if sent.AckTopic != BroadcastAckTopic(testPrefix, testInstance) {
		t.Fatalf("ack topic must be auto-filled with own ack topic, got %q", sent.AckTopic)
	}
	if sent.Body != `{"taskId":"abc"}` {
		t.Fatalf("body = %q, want payload bytes", sent.Body)
	}
}

func TestBroadcastReturnsPublishError(t *testing.T) {
	mock := newMockMqttClient()
	mock.publishErr = errors.New("broker down")
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	err := b.Broadcast(context.Background(), testBroadcast, []byte("payload"))
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestBroadcastReplyWithoutRouterFailsGracefully(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	_, err := b.BroadcastReply(context.Background(), testBroadcast, []byte("payload"), 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, mqttx.ErrNoReplyRouter) {
		t.Fatalf("expected ErrNoReplyRouter, got %v", err)
	}
	if mock.publishedCount() != 0 {
		t.Fatalf("no publish must happen without router, published=%d", mock.publishedCount())
	}
}

func TestBroadcastReplyResultMapping(t *testing.T) {
	t.Run("success_returns_bytes", func(t *testing.T) {
		got, err := replyResult(&BroadcastAckBody{Success: true, ResponseBody: `{"value":1}`})
		if err != nil {
			t.Fatalf("replyResult returned error: %v", err)
		}
		if string(got) != `{"value":1}` {
			t.Fatalf("replyResult = %q, want response bytes", got)
		}
	})

	t.Run("failure_restores_kind", func(t *testing.T) {
		_, err := replyResult(&BroadcastAckBody{Success: false, ErrorKind: KindTimeout, Error: "expired"})
		if !errors.Is(err, antsx.ErrReplyExpired) {
			t.Fatalf("replyResult(timeout) = %v, want ErrReplyExpired", err)
		}
		_, err = replyResult(&BroadcastAckBody{Success: false, ErrorKind: KindDuplicate, Error: "dup"})
		if !errors.Is(err, antsx.ErrDuplicateID) {
			t.Fatalf("replyResult(duplicate) = %v, want ErrDuplicateID", err)
		}
		_, err = replyResult(&BroadcastAckBody{Success: false, ErrorKind: KindUnknown, Error: "some text"})
		if err == nil || err.Error() != "some text" {
			t.Fatalf("replyResult(unknown) = %v, want text error", err)
		}
	})
}

func TestAddBroadcastHandlerRegistersPattern(t *testing.T) {
	mock := newMockMqttClient()
	b := NewBroadcaster(mock, testInstance, WithPrefix(testPrefix))

	if err := b.AddBroadcastHandler(); err != nil {
		t.Fatalf("AddBroadcastHandler returned error: %v", err)
	}
	if !mock.hasHandler(BroadcastTopicPattern(testPrefix)) {
		t.Fatalf("expected handler registered on %q", BroadcastTopicPattern(testPrefix))
	}
}

func mustLast(t *testing.T, mock *mockMqttClient) publishedMessage {
	t.Helper()
	msg, ok := mock.lastPublished()
	if !ok {
		t.Fatal("expected publish")
	}
	return msg
}
