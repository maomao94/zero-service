package kqx

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

var testPartitions = []int{0, 1, 2, 3, 4, 5, 6, 7}

func msgWithKey(key string) kafka.Message {
	return kafka.Message{Key: []byte(key)}
}

func optionsOf(opts ...Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func TestNoOrderedOptionKeepsGoQueueDefault(t *testing.T) {
	if o := optionsOf(); o.ordered {
		t.Fatal("ordering should not be enabled by default")
	}
}

func TestWithOrderedKeepsSameKeyOnSamePartition(t *testing.T) {
	if o := optionsOf(WithOrdered()); !o.ordered {
		t.Fatal("WithOrdered did not enable ordered routing")
	}

	balancer := &kafka.Hash{}
	first := balancer.Balance(msgWithKey("10.0.0.1_1_0x000001"), testPartitions...)
	for i := 0; i < 200; i++ {
		if got := balancer.Balance(msgWithKey("10.0.0.1_1_0x000001"), testPartitions...); got != first {
			t.Fatalf("same key routed to different partitions: first=%d, got=%d at iteration %d", first, got, i)
		}
	}
}

func TestWithOrderedHandlesEmptyKeyDeterministically(t *testing.T) {
	// PushWithKey 传入空字符串时是 []byte("")，非 nil，kafka.Hash 仍会做哈希。
	balancer := &kafka.Hash{}
	first := balancer.Balance(msgWithKey(""), testPartitions...)

	for i := 0; i < 200; i++ {
		if got := balancer.Balance(msgWithKey(""), testPartitions...); got != first {
			t.Fatalf("empty key routed to different partitions: first=%d, got=%d at iteration %d", first, got, i)
		}
	}
}

func TestNewPusherAppliesOptions(t *testing.T) {
	pusher := NewPusher([]string{"127.0.0.1:9092"}, "asdu",
		WithOrdered(),
		WithSyncPush(),
		WithAllowAutoTopicCreation(),
		WithChunkSize(64*1024),
		WithFlushInterval(time.Second),
	)
	if pusher == nil {
		t.Fatal("NewPusher returned nil")
	}
	if err := pusher.Close(); err != nil {
		t.Fatalf("close pusher: %v", err)
	}
}

func TestNewPusherWithoutOrderedOption(t *testing.T) {
	pusher := NewPusher([]string{"127.0.0.1:9092"}, "asdu", WithSyncPush())
	if pusher == nil {
		t.Fatal("NewPusher returned nil")
	}
	if err := pusher.Close(); err != nil {
		t.Fatalf("close pusher: %v", err)
	}
}
