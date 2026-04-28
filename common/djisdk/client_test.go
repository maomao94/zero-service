package djisdk

import "context"

type publishedMessage struct {
	topic   string
	payload []byte
}

type recordingMQTTClient struct {
	published []publishedMessage
	handlers  map[string]func(context.Context, []byte, string, string) error
}

func (c *recordingMQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
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
