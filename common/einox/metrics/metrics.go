// Package metrics 为 einox 子系统提供统一的指标接入点。
//
// 同时保留两套实现：
//  1. Prometheus 指标（被 aisolo 主进程通过 metric.RegisterHandler 导出到 /metrics）
//  2. logx fallback（当 prometheus 初始化失败或未启用时, 仅记日志）
//
// 上层调用只认 RecordXxx 方法, 不关心底层实现。
package metrics

import (
	"context"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zeromicro/go-zero/core/logx"
)

const namespace = "einox"

// Metrics 聚合所有埋点指标。
type Metrics struct {
	turnDuration    *prometheus.HistogramVec // mode, status
	agentDuration   *prometheus.HistogramVec // agent_type, status
	toolCallCounter *prometheus.CounterVec   // tool, status
	toolCallLatency *prometheus.HistogramVec // tool, status
	interruptTotal  *prometheus.CounterVec   // kind, tool
	resumeTotal     *prometheus.CounterVec   // kind, status, mode, action (yes|no|unspecified)
	checkpointTotal *prometheus.CounterVec   // op, status

	knowledgeDuration *prometheus.HistogramVec // op, status, backend
}

// Option Metrics 构造选项。
type Option func(*Metrics)

var (
	global     *Metrics
	globalOnce sync.Once
)

// Global 返回全局 Metrics 实例, 首次调用时自动注册 Prometheus collectors。
func Global() *Metrics {
	globalOnce.Do(func() {
		global = NewMetrics()
	})
	return global
}

// NewMetrics 创建指标收集器。已注册过的 collector 会自动复用, 不会重复注册。
func NewMetrics() *Metrics {
	m := &Metrics{
		turnDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "turn",
			Name:      "duration_seconds",
			Help:      "Duration of one agent turn (Run or Resume).",
			Buckets:   prometheus.DefBuckets,
		}, []string{"mode", "status"}),

		agentDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "agent",
			Name:      "duration_seconds",
			Help:      "Duration of a single agent invocation (excluding tool time).",
			Buckets:   prometheus.DefBuckets,
		}, []string{"agent_type", "status"}),

		toolCallCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "tool",
			Name:      "calls_total",
			Help:      "Total number of tool invocations.",
		}, []string{"tool", "status"}),

		toolCallLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "tool",
			Name:      "duration_seconds",
			Help:      "Duration of tool invocations.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"tool", "status"}),

		interruptTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "interrupt",
			Name:      "total",
			Help:      "Total number of interrupts, by kind and tool.",
		}, []string{"kind", "tool"}),

		resumeTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "resume",
			Name:      "total",
			Help:      "Resume outcomes: ok=session idle; error=resume or stream failed; interrupted_again=resume finished but new HITL (session still interrupted). Labels: kind, status, mode, action.",
		}, []string{"kind", "status", "mode", "action"}),

		checkpointTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "checkpoint",
			Name:      "total",
			Help:      "Total number of checkpoint operations.",
		}, []string{"op", "status"}),

		knowledgeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "knowledge",
			Name:      "operation_duration_seconds",
			Help:      "Knowledge ingest/search latency by backend (memory|gorm|redis|milvus).",
			Buckets:   prometheus.DefBuckets,
		}, []string{"op", "status", "backend"}),
	}

	m.turnDuration = registerOrReuse(m.turnDuration)
	m.agentDuration = registerOrReuse(m.agentDuration)
	m.toolCallLatency = registerOrReuse(m.toolCallLatency)
	m.knowledgeDuration = registerOrReuse(m.knowledgeDuration)
	m.toolCallCounter = registerOrReuse(m.toolCallCounter)
	m.interruptTotal = registerOrReuse(m.interruptTotal)
	m.resumeTotal = registerOrReuse(m.resumeTotal)
	m.checkpointTotal = registerOrReuse(m.checkpointTotal)
	return m
}

// =============================================================================
// Record 方法 —— 上层业务的全部埋点入口
// =============================================================================

// promLabelKindOrTool 避免把 Go 里对 int32 枚举误做的 string(enum)（Unicode 控制符）写进 Prometheus 标签。
func promLabelKindOrTool(s string) string {
	if s == "" {
		return ""
	}
	if !utf8.ValidString(s) {
		return "invalid_utf8"
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return "invalid_enum_string"
		}
	}
	return s
}

// RecordTurn 记录一轮对话的耗时。
func (m *Metrics) RecordTurn(ctx context.Context, mode, status string, d time.Duration) {
	m.turnDuration.WithLabelValues(mode, status).Observe(d.Seconds())
	logx.WithContext(ctx).Infof("[einox.metrics] turn mode=%s status=%s dur=%s", mode, status, d)
}

// RecordAgent 记录单个 Agent 调用。
func (m *Metrics) RecordAgent(ctx context.Context, agentType, status string, d time.Duration) {
	m.agentDuration.WithLabelValues(agentType, status).Observe(d.Seconds())
	if d > time.Second {
		logx.WithContext(ctx).Slowf("[einox.metrics] agent type=%s status=%s dur=%s", agentType, status, d)
	}
}

// RecordToolCall 记录一次 Tool 调用。
func (m *Metrics) RecordToolCall(ctx context.Context, tool, status string, d time.Duration) {
	m.toolCallCounter.WithLabelValues(tool, status).Inc()
	m.toolCallLatency.WithLabelValues(tool, status).Observe(d.Seconds())
	if status == "error" {
		logx.WithContext(ctx).Errorf("[einox.metrics] tool=%s error dur=%s", tool, d)
	}
}

// RecordInterrupt 记录一次中断发生。interruptID 仅写入日志，不进 Prometheus 标签（避免高基数）。
func (m *Metrics) RecordInterrupt(ctx context.Context, kind, tool, interruptID string) {
	k, t := promLabelKindOrTool(kind), promLabelKindOrTool(tool)
	m.interruptTotal.WithLabelValues(k, t).Inc()
	if interruptID != "" {
		logx.WithContext(ctx).Infof("[einox.metrics] interrupt interrupt_id=%q kind=%q tool=%q", interruptID, kind, tool)
	} else {
		logx.WithContext(ctx).Infof("[einox.metrics] interrupt kind=%q tool=%q", kind, tool)
	}
}

// RecordResume 记录一次 Resume 结果。resumeAction 为客户端 YES/NO（标签 action）。
func (m *Metrics) RecordResume(ctx context.Context, kind, status, mode, interruptID, resumeAction string, d time.Duration) {
	k := promLabelKindOrTool(kind)
	mod := promLabelKindOrTool(mode)
	act := promLabelKindOrTool(resumeAction)
	if act == "" {
		act = "unspecified"
	}
	if mod == "" {
		mod = "unknown"
	}
	if k != kind {
		logx.WithContext(ctx).Errorf(
			"[einox.metrics] resume kind label sanitized (caller likely used string(int32_enum)); raw=%q status=%s mode=%s action=%s interrupt_id=%q",
			kind, status, mod, act, interruptID,
		)
	}
	m.resumeTotal.WithLabelValues(k, status, mod, act).Inc()
	if interruptID != "" {
		logx.WithContext(ctx).Infof("[einox.metrics] resume interrupt_id=%q kind=%q status=%s mode=%s action=%s dur=%s", interruptID, kind, status, mod, act, d)
	} else {
		logx.WithContext(ctx).Infof("[einox.metrics] resume kind=%q status=%s mode=%s action=%s dur=%s", kind, status, mod, act, d)
	}
}

// RecordCheckPoint 记录 checkpoint 存取操作。
func (m *Metrics) RecordCheckPoint(ctx context.Context, op, status string) {
	m.checkpointTotal.WithLabelValues(op, status).Inc()
	if status == "error" {
		logx.WithContext(ctx).Errorf("[einox.metrics] checkpoint op=%s error", op)
	}
}

// RecordKnowledge 记录知识库 ingest/search 耗时（backend 为 knowledge.EffectiveBackend()）。
func (m *Metrics) RecordKnowledge(ctx context.Context, op, status, backend string, d time.Duration) {
	b := promLabelKindOrTool(backend)
	if b == "" {
		b = "unknown"
	}
	o := promLabelKindOrTool(op)
	st := promLabelKindOrTool(status)
	if st == "" {
		st = "unknown"
	}
	m.knowledgeDuration.WithLabelValues(o, st, b).Observe(d.Seconds())
	if st == "error" {
		logx.WithContext(ctx).Debugf("[einox.metrics] knowledge op=%s backend=%s dur=%s", o, b, d)
	}
}

// =============================================================================
// 注册工具函数
// =============================================================================

func registerOrReuse[T prometheus.Collector](c T) T {
	if err := prometheus.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector.(T)
		}
		logx.Errorf("[einox.metrics] register: %v", err)
	}
	return c
}
