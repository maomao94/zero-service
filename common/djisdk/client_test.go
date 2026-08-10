package djisdk

import (
	"context"
	"errors"
	"strings"
	"testing"

	"zero-service/common/mqttx"
)

type publishedMessage struct {
	topic   string
	payload []byte
}

type recordingMQTTClient struct {
	published []publishedMessage
	handlers  map[string]func(context.Context, []byte, string, string) error
	err       error
}

func (c *recordingMQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if c.err != nil {
		return c.err
	}
	data := append([]byte(nil), payload...)
	c.published = append(c.published, publishedMessage{topic: topic, payload: data})
	return nil
}

func (c *recordingMQTTClient) AddHandlerFunc(topic string, fn func(context.Context, []byte, string, string) error) error {
	if c.handlers == nil {
		c.handlers = make(map[string]func(context.Context, []byte, string, string) error)
	}
	c.handlers[topic] = fn
	return nil
}

func (c *recordingMQTTClient) AddHandler(topic string, handler mqttx.ConsumeHandler) error {
	return c.AddHandlerFunc(topic, handler.Consume)
}

func (c *recordingMQTTClient) Subscribe(topic string) error {
	return nil
}

func (c *recordingMQTTClient) PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error) {
	return "", c.Publish(ctx, topic, payload)
}

func (c *recordingMQTTClient) Close() {}

func (c *recordingMQTTClient) GetClientID() string {
	return "recording"
}

var errPublishFailed = errors.New("publish failed")

func TestSendCommandRejectsOfflineDeviceBeforePublish(t *testing.T) {
	mqtt := &recordingMQTTClient{}
	client := NewClient(mqtt,
		WithPendingTTL(0),
		WithOnlineChecker(func(gatewaySn string) bool { return gatewaySn == "online-1" }),
	)

	if _, err := client.SendCommand(context.Background(), "offline-1", MethodFlightTaskExecute, map[string]any{}); err == nil {
		t.Fatal("SendCommand() succeeded for offline device, want error")
	} else if !strings.Contains(err.Error(), "device offline") {
		t.Fatalf("SendCommand() error = %v, want device offline message", err)
	}
	if len(mqtt.published) != 0 {
		t.Fatalf("SendCommand() published %d messages, want 0", len(mqtt.published))
	}
}
