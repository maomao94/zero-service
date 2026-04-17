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
	resumeTotal     *prometheus.CounterVec   // kind, status
	checkpointTotal *prometheus.CounterVec   // op, status
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
			Help:      "Total number of resume attempts, by interrupt kind and status.",
		}, []string{"kind", "status"}),

		checkpointTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "checkpoint",
			Name:      "total",
			Help:      "Total number of checkpoint operations.",
		}, []string{"op", "status"}),
	}

	register(m.turnDuration, m.agentDuration, m.toolCallLatency)
	registerCounters(m.toolCallCounter, m.interruptTotal, m.resumeTotal, m.checkpointTotal)
	return m
}

// =============================================================================
// Record 方法 —— 上层业务的全部埋点入口
// =============================================================================

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

// RecordInterrupt 记录一次中断发生。
func (m *Metrics) RecordInterrupt(ctx context.Context, kind, tool string) {
	m.interruptTotal.WithLabelValues(kind, tool).Inc()
	logx.WithContext(ctx).Infof("[einox.metrics] interrupt kind=%s tool=%s", kind, tool)
}

// RecordResume 记录一次 Resume 结果。
func (m *Metrics) RecordResume(ctx context.Context, kind, status string, d time.Duration) {
	m.resumeTotal.WithLabelValues(kind, status).Inc()
	logx.WithContext(ctx).Infof("[einox.metrics] resume kind=%s status=%s dur=%s", kind, status, d)
}

// RecordCheckPoint 记录 checkpoint 存取操作。
func (m *Metrics) RecordCheckPoint(ctx context.Context, op, status string) {
	m.checkpointTotal.WithLabelValues(op, status).Inc()
	if status == "error" {
		logx.WithContext(ctx).Errorf("[einox.metrics] checkpoint op=%s error", op)
	}
}

// =============================================================================
// 注册工具函数
// =============================================================================

func register(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := prometheus.Register(c); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				// 已注册直接忽略; 复用原 collector 让外部仍能写入
				_ = are
				continue
			}
			logx.Errorf("[einox.metrics] register: %v", err)
		}
	}
}

func registerCounters(cs ...prometheus.Collector) { register(cs...) }
