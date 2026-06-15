package mqttx

import (
	"sync"
	"sync/atomic"
	"time"
)

// TopicLogSetting 单个 topic 的日志配置
type TopicLogSetting struct {
	// TopicTemplate 主题模板，如 "device/+/data"
	TopicTemplate string `json:"topicTemplate"`
	// LogPayload 是否打印消息内容，默认 false
	LogPayload bool `json:"logPayload"`
	// MinLogInterval 最小日志间隔，避免日志刷屏，默认 5s
	MinLogInterval time.Duration `json:",default=5s"`
}

// TopicLogConfig MQTT 日志配置
// 用于控制消息日志的打印频率和内容
type TopicLogConfig struct {
	// DefaultLogPayload 默认是否打印 payload，默认 false
	DefaultLogPayload bool `json:"defaultLogPayload"`
	// TopicSettings 按 topic 的日志配置列表
	TopicSettings []TopicLogSetting `json:"topicSettings"`
}

// topicLogConfig 内部日志配置
type topicLogConfig struct {
	logPayload     atomic.Bool
	minLogInterval time.Duration
	lastLogUnix    int64
}

// newTopicLogConfig 创建日志配置
func newTopicLogConfig(interval time.Duration) *topicLogConfig {
	if interval == 0 {
		interval = 5 * time.Second
	}
	return &topicLogConfig{
		minLogInterval: interval,
		lastLogUnix:    0,
	}
}

// shouldLog 判断是否应该打印日志（基于时间间隔）
func (c *topicLogConfig) shouldLog() bool {
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&c.lastLogUnix)
	if now-last < c.minLogInterval.Nanoseconds() {
		return false
	}
	return atomic.CompareAndSwapInt64(&c.lastLogUnix, last, now)
}

// shouldLogPayload 是否打印 payload
func (c *topicLogConfig) shouldLogPayload() bool {
	return c.logPayload.Load()
}

// setLogPayload 设置是否打印 payload
func (c *topicLogConfig) setLogPayload(enabled bool) {
	c.logPayload.Store(enabled)
}

// topicLogManagerOption 日志管理器选项
type topicLogManagerOption func(*TopicLogManager)

// TopicLogManager 日志管理器
// 维护各 topic 的日志配置，并控制打印频率
type TopicLogManager struct {
	configs           sync.Map
	defaultLogPayload atomic.Bool
	defaultInterval   time.Duration
}

// NewTopicLogManager 创建日志管理器
func NewTopicLogManager(opts ...topicLogManagerOption) *TopicLogManager {
	m := &TopicLogManager{
		defaultInterval: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithDefaultLogPayload 设置默认是否打印 payload
func WithDefaultLogPayload(enabled bool) topicLogManagerOption {
	return func(m *TopicLogManager) {
		m.defaultLogPayload.Store(enabled)
	}
}

// WithDefaultInterval 设置默认日志间隔
func WithDefaultInterval(interval time.Duration) topicLogManagerOption {
	return func(m *TopicLogManager) {
		m.defaultInterval = interval
	}
}

// ShouldLog 判断 topic 是否应该打印日志
func (m *TopicLogManager) ShouldLog(topic string) bool {
	return m.getConfig(topic).shouldLog()
}

// ShouldLogPayload 判断 topic 是否应该打印 payload
func (m *TopicLogManager) ShouldLogPayload(topic string) bool {
	return m.getConfig(topic).shouldLogPayload()
}

// SetLogPayload 设置 topic 是否打印 payload
func (m *TopicLogManager) SetLogPayload(topic string, enabled bool) {
	m.getConfig(topic).setLogPayload(enabled)
}

// LoadFromConfig 从配置加载日志配置
func (m *TopicLogManager) LoadFromConfig(cfg TopicLogConfig) {
	m.defaultLogPayload.Store(cfg.DefaultLogPayload)
	m.defaultInterval = 5 * time.Second

	for _, setting := range cfg.TopicSettings {
		interval := setting.MinLogInterval
		if interval == 0 {
			interval = m.defaultInterval
		}
		conf := newTopicLogConfig(interval)
		conf.setLogPayload(setting.LogPayload)
		m.configs.Store(setting.TopicTemplate, conf)
	}
}

// getConfig 获取 topic 的日志配置
func (m *TopicLogManager) getConfig(topic string) *topicLogConfig {
	if v, ok := m.configs.Load(topic); ok {
		return v.(*topicLogConfig)
	}
	conf := newTopicLogConfig(m.defaultInterval)
	conf.setLogPayload(m.defaultLogPayload.Load())
	actual, _ := m.configs.LoadOrStore(topic, conf)
	return actual.(*topicLogConfig)
}
