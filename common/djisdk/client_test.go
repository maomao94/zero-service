package djisdk

import (
	"context"
	"errors"

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
