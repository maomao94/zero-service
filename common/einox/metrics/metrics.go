package metrics

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	namespace = "einox"
)

// AgentLabels Agent 指标标签
type AgentLabels struct {
	AgentType string
	Status    string
}

// RouterLabels Router 指标标签
type RouterLabels struct {
	Intent string
	Status string
}

// CheckpointLabels Checkpoint 指标标签
type CheckpointLabels struct {
	Operation string
	Status    string
}

// Metrics 日志式指标收集器（避免依赖 prometheus）
type Metrics struct{}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordAgentRequest 记录 Agent 请求
func (m *Metrics) RecordAgentRequest(ctx context.Context, agentType, status string, duration time.Duration) {
	logx.WithContext(ctx).Slow("agent request: type=" + agentType + ", status=" + status + ", duration=" + duration.String())
}

// RecordRouterRequest 记录 Router 请求
func (m *Metrics) RecordRouterRequest(ctx context.Context, intent, status string, duration time.Duration) {
	logx.WithContext(ctx).Slow("router request: intent=" + intent + ", status=" + status)
}

// RecordCheckPointOperation 记录 Checkpoint 操作
func (m *Metrics) RecordCheckPointOperation(ctx context.Context, operation, status string, duration time.Duration) {
	logx.WithContext(ctx).Slow("checkpoint operation: op=" + operation + ", status=" + status)
}

// RecordInterrupt 记录中断
func (m *Metrics) RecordInterrupt(ctx context.Context, interruptType, status string) {
	logx.WithContext(ctx).Info("interrupt: type=" + interruptType + ", status=" + status)
}

// RecordResume 记录恢复
func (m *Metrics) RecordResume(ctx context.Context, status string, duration time.Duration) {
	logx.WithContext(ctx).Info("resume: status=" + status + ", duration=" + duration.String())
}

// GlobalMetrics 全局指标实例
var GlobalMetrics = NewMetrics()
