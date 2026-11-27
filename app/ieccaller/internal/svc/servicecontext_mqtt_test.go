package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"zero-service/app/ieccaller/internal/config"
	"zero-service/common/iec104/types"
	"zero-service/common/mqttx"
)

const (
	testHost      = "192.168.1.100"
	testPort      = 2404
	testStationID = "SITE_A"
	testIoa       = uint(16385)
)

// publishAttempt 记录一次 MQTT Publish 调用，便于断言 topic 生成结果和上下文超时。
type publishAttempt struct {
	topic       string
	payload     []byte
	timeout     time.Duration
	hasDeadline bool
	err         error
}

// fakeMqttClient 是 mqttx.Client 的测试实现，只记录发布行为。
type fakeMqttClient struct {
	mu        sync.Mutex
	attempts  []publishAttempt
	failTopic map[string]error
}

func (f *fakeMqttClient) AddHandler(string, mqttx.ConsumeHandler) error { return nil }

func (f *fakeMqttClient) AddHandlerFunc(string, func(context.Context, []byte, string, string) error) error {
	return nil
}

func (f *fakeMqttClient) AddReplyRouter(string, mqttx.ConsumeHandler) error { return nil }

func (f *fakeMqttClient) Publish(ctx context.Context, topic string, payload []byte) error {
	attempt := publishAttempt{
		topic:   topic,
		payload: append([]byte(nil), payload...),
	}
	if deadline, ok := ctx.Deadline(); ok {
		attempt.hasDeadline = true
		attempt.timeout = time.Until(deadline)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.failTopic[topic]; ok {
		attempt.err = err
		f.attempts = append(f.attempts, attempt)
		return err
	}
	f.attempts = append(f.attempts, attempt)
	return nil
}

func (f *fakeMqttClient) Close() {}

func (f *fakeMqttClient) GetClientID() string { return "fake-mqtt-client" }

func (f *fakeMqttClient) snapshot() []publishAttempt {
	f.mu.Lock()
	defer f.mu.Unlock()
	attempts := make([]publishAttempt, len(f.attempts))
	copy(attempts, f.attempts)
	return attempts
}

func (f *fakeMqttClient) topics() []string {
	attempts := f.snapshot()
	topics := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		topics = append(topics, attempt.topic)
	}
	return topics
}

// newTestServiceContext 构造只启用 MQTT 推送的 ServiceContext。
func newTestServiceContext(topics []string, isPush bool, client mqttx.Client) ServiceContext {
	var cfg config.Config
	cfg.MqttConfig.Topic = topics
	cfg.MqttConfig.IsPush = isPush
	cfg.MqttConfig.Timeout = 2000
	return ServiceContext{
		Config:     cfg,
		MqttClient: client,
	}
}

func newTestMsgBody() *types.MsgBody {
	return &types.MsgBody{
		MsgId:    "msg-001",
		Host:     testHost,
		Port:     testPort,
		Asdu:     "M_SP_NA_1",
		TypeId:   1,
		DataType: 1,
		Coa:      1,
		Body:     &types.SinglePointInfo{Ioa: testIoa, Value: true},
		MetaData: map[string]any{"stationId": testStationID},
	}
}

func assertTopics(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected publish count: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected topic at index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// 覆盖 PushASDU 中 topic 模板的逐条生成与发布：每条模板各发一次，携带同一份报文。
func TestPushASDUPublishesEachGeneratedTopic(t *testing.T) {
	fake := &fakeMqttClient{}
	svcCtx := newTestServiceContext([]string{
		"iec/asdu",
		"iec/asdu/{{.MetaData.stationId}}/{{.TypeId}}",
		"iec/asdu/{{.Host}}/{{.Port}}",
		"iec/asdu/{{.MetaData.stationId}}/{{.Body.Ioa}}",
	}, true, fake)

	data := newTestMsgBody()
	if err := svcCtx.PushASDU(context.Background(), data, testIoa); err != nil {
		t.Fatalf("PushASDU failed: %v", err)
	}

	assertTopics(t, fake.topics(), []string{
		"iec/asdu",
		"iec/asdu/" + testStationID + "/1",
		"iec/asdu/" + testHost + "/2404",
		"iec/asdu/" + testStationID + "/16385",
	})

	attempts := fake.snapshot()
	for _, attempt := range attempts {
		if !attempt.hasDeadline {
			t.Errorf("topic %q published without deadline", attempt.topic)
			continue
		}
		if attempt.timeout <= 0 || attempt.timeout > 2100*time.Millisecond {
			t.Errorf("topic %q published with unexpected timeout %s", attempt.topic, attempt.timeout)
		}
		if attempt.err != nil {
			t.Errorf("topic %q unexpected publish error: %v", attempt.topic, attempt.err)
		}
	}

	if !bytes.Equal(attempts[0].payload, attempts[1].payload) ||
		!bytes.Equal(attempts[0].payload, attempts[2].payload) ||
		!bytes.Equal(attempts[0].payload, attempts[3].payload) {
		t.Fatalf("payload differs across topics")
	}

	var payload struct {
		MsgId    string `json:"msgId"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		TypeId   int    `json:"typeId"`
		Coa      uint   `json:"coa"`
		Time     string `json:"time"`
		MetaData struct {
			StationID string `json:"stationId"`
		} `json:"metaData"`
		Body struct {
			Ioa   uint `json:"ioa"`
			Value bool `json:"value"`
		} `json:"body"`
	}
	if err := json.Unmarshal(attempts[0].payload, &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if payload.MsgId != "msg-001" || payload.Host != testHost || payload.Port != testPort {
		t.Fatalf("unexpected payload header: %+v", payload)
	}
	if payload.TypeId != 1 || payload.Coa != 1 || payload.Body.Ioa != testIoa || !payload.Body.Value {
		t.Fatalf("unexpected payload body: %+v", payload)
	}
	if payload.MetaData.StationID != testStationID {
		t.Fatalf("unexpected metaData.stationId: %q", payload.MetaData.StationID)
	}
	if payload.Time == "" {
		t.Fatal("payload time should be filled before publish")
	}
}

// 模板生成失败时只跳过该条模板，后续模板仍要发布。
func TestPushASDUSkipsInvalidTopicPatternAndContinues(t *testing.T) {
	fake := &fakeMqttClient{}
	svcCtx := newTestServiceContext([]string{
		"iec/asdu/{{.host}}", // Go 模板字段大小写不匹配
		"iec/asdu//invalid",  // 连续斜杠
		"iec/asdu/+/invalid", // 发布侧禁止通配符
		"iec/asdu/{{.Host}}/{{.Port}}",
	}, true, fake)

	if err := svcCtx.PushASDU(context.Background(), newTestMsgBody(), testIoa); err != nil {
		t.Fatalf("PushASDU failed: %v", err)
	}

	assertTopics(t, fake.topics(), []string{"iec/asdu/" + testHost + "/2404"})
}

// 点位映射模板依赖 data.Pm，未命中映射时只跳过这些模板。
func TestPushASDUDeviceTopicRequiresPointMapping(t *testing.T) {
	t.Run("without point mapping", func(t *testing.T) {
		fake := &fakeMqttClient{}
		svcCtx := newTestServiceContext([]string{
			"iec/asdu/device/{{.Pm.DeviceId}}",
			"iec/asdu/ext1/{{.Pm.Ext1}}",
			"iec/asdu/{{.Host}}",
		}, true, fake)

		data := newTestMsgBody()
		if err := svcCtx.PushASDU(context.Background(), data, testIoa); err != nil {
			t.Fatalf("PushASDU failed: %v", err)
		}

		assertTopics(t, fake.topics(), []string{"iec/asdu/" + testHost})
	})

	t.Run("with point mapping", func(t *testing.T) {
		fake := &fakeMqttClient{}
		svcCtx := newTestServiceContext([]string{
			"iec/asdu/device/{{.Pm.DeviceId}}",
			"iec/asdu/ext1/{{.Pm.Ext1}}",
		}, true, fake)

		data := newTestMsgBody()
		data.Pm = &types.PointMapping{DeviceId: "device-1", Ext1: "alarm"}
		if err := svcCtx.PushASDU(context.Background(), data, testIoa); err != nil {
			t.Fatalf("PushASDU failed: %v", err)
		}

		assertTopics(t, fake.topics(), []string{
			"iec/asdu/device/device-1",
			"iec/asdu/ext1/alarm",
		})
	})
}

// 单条发布失败只记录日志，不能中断后续 topic。
func TestPushASDUContinuesAfterPublishError(t *testing.T) {
	fake := &fakeMqttClient{failTopic: map[string]error{
		"iec/asdu/2": errors.New("broker rejected"),
	}}
	svcCtx := newTestServiceContext([]string{
		"iec/asdu/1",
		"iec/asdu/2",
		"iec/asdu/3",
	}, true, fake)

	if err := svcCtx.PushASDU(context.Background(), newTestMsgBody(), testIoa); err != nil {
		t.Fatalf("PushASDU should not propagate publish error: %v", err)
	}

	assertTopics(t, fake.topics(), []string{"iec/asdu/1", "iec/asdu/2", "iec/asdu/3"})
	if err := fake.snapshot()[1].err; err == nil || err.Error() != "broker rejected" {
		t.Fatalf("unexpected error recorded for failed topic: %v", err)
	}
}

// 关闭 MQTT 推送或未配置主题时不得发起发布。
func TestPushASDUSkipsWhenPushDisabledOrTopicEmpty(t *testing.T) {
	t.Run("push disabled", func(t *testing.T) {
		fake := &fakeMqttClient{}
		svcCtx := newTestServiceContext([]string{"iec/asdu/{{.Host}}"}, false, fake)

		if err := svcCtx.PushASDU(context.Background(), newTestMsgBody(), testIoa); err != nil {
			t.Fatalf("PushASDU failed: %v", err)
		}
		assertTopics(t, fake.topics(), nil)
	})

	t.Run("topic empty", func(t *testing.T) {
		fake := &fakeMqttClient{}
		svcCtx := newTestServiceContext(nil, true, fake)

		if err := svcCtx.PushASDU(context.Background(), newTestMsgBody(), testIoa); err != nil {
			t.Fatalf("PushASDU failed: %v", err)
		}
		assertTopics(t, fake.topics(), nil)
	})
}

// MQTT 客户端未初始化时只记录错误日志，不能 panic。
func TestPushASDUNilMqttClientDoesNotPanic(t *testing.T) {
	svcCtx := newTestServiceContext([]string{"iec/asdu/{{.Host}}"}, true, nil)

	if err := svcCtx.PushASDU(context.Background(), newTestMsgBody(), testIoa); err != nil {
		t.Fatalf("PushASDU failed: %v", err)
	}
}

// 发布上下文超时跟随 MqttConfig.Timeout，未配置时与 mqttx 的 30s 兜底保持一致。
func TestMqttPublishTimeoutFollowsConfig(t *testing.T) {
	var svcCtx ServiceContext
	if got := svcCtx.mqttPublishTimeout(); got != 30*time.Second {
		t.Fatalf("default timeout: got %s, want %s", got, 30*time.Second)
	}

	svcCtx.Config.MqttConfig.Timeout = 2000
	if got := svcCtx.mqttPublishTimeout(); got != 2*time.Second {
		t.Fatalf("configured timeout: got %s, want %s", got, 2*time.Second)
	}
}
