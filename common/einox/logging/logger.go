package logging

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// AgentLogger Agent 日志记录器
type AgentLogger struct {
	agentType string
}

// NewAgentLogger 创建 Agent 日志记录器
func NewAgentLogger(agentType string) *AgentLogger {
	return &AgentLogger{agentType: agentType}
}

// LogRequest 记录请求
func (l *AgentLogger) LogRequest(ctx context.Context, userID, sessionID, input string) {
	logx.WithContext(ctx).Info("[Agent:" + l.agentType + "] Request received")
}

// LogResponse 记录响应
func (l *AgentLogger) LogResponse(ctx context.Context, userID, sessionID, output string, duration time.Duration) {
	logx.WithContext(ctx).WithDuration(duration).Info("[Agent:" + l.agentType + "] Response sent")
}

// LogError 记录错误
func (l *AgentLogger) LogError(ctx context.Context, userID, sessionID string, err error) {
	logx.WithContext(ctx).Error("[Agent:" + l.agentType + "] Error: " + err.Error())
}

// LogToolCall 记录工具调用
func (l *AgentLogger) LogToolCall(ctx context.Context, toolName string, args string) {
	logx.WithContext(ctx).Debug("[Agent:" + l.agentType + "] ToolCall: " + toolName)
}

// LogInterrupt 记录中断
func (l *AgentLogger) LogInterrupt(ctx context.Context, userID, sessionID, reason string) {
	logx.WithContext(ctx).Info("[Agent:" + l.agentType + "] Interrupt: " + reason)
}

// LogResume 记录恢复
func (l *AgentLogger) LogResume(ctx context.Context, userID, sessionID string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	logx.WithContext(ctx).Info("[Agent:" + l.agentType + "] Resume: " + status)
}

// RouterLogger Router 日志记录器
type RouterLogger struct{}

// NewRouterLogger 创建 Router 日志记录器
func NewRouterLogger() *RouterLogger {
	return &RouterLogger{}
}

// LogRoute 记录路由决策
func (l *RouterLogger) LogRoute(ctx context.Context, query, intent, agentType string, confidence float64) {
	logx.WithContext(ctx).Info("[Router] Route: intent=" + intent + ", agent=" + agentType)
}

// LogRouteError 记录路由错误
func (l *RouterLogger) LogRouteError(ctx context.Context, query string, err error) {
	logx.WithContext(ctx).Error("[Router] RouteError: " + err.Error())
}

// CheckpointLogger Checkpoint 日志记录器
type CheckpointLogger struct{}

// NewCheckpointLogger 创建 Checkpoint 日志记录器
func NewCheckpointLogger() *CheckpointLogger {
	return &CheckpointLogger{}
}

// LogSave 记录保存
func (l *CheckpointLogger) LogSave(ctx context.Context, checkpointID string, size int) {
	logx.WithContext(ctx).Debug("[Checkpoint] Save: " + checkpointID)
}

// LogLoad 记录加载
func (l *CheckpointLogger) LogLoad(ctx context.Context, checkpointID string, exists bool) {
	logx.WithContext(ctx).Debug("[Checkpoint] Load: " + checkpointID)
}

// LogDelete 记录删除
func (l *CheckpointLogger) LogDelete(ctx context.Context, checkpointID string) {
	logx.WithContext(ctx).Debug("[Checkpoint] Delete: " + checkpointID)
}

// LogError 记录错误
func (l *CheckpointLogger) LogError(ctx context.Context, operation string, err error) {
	logx.WithContext(ctx).Error("[Checkpoint] Error: " + err.Error())
}

// LogSlowRequest 记录慢请求
func LogSlowRequest(ctx context.Context, operation string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		logx.WithContext(ctx).Slow("Slow " + operation + ": duration=" + duration.String())
	}
}
