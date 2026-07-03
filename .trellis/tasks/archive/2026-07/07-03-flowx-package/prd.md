# PRD: common/flowx — go-zero 集成的 workflow 封装

## 概述

基于 `github.com/Azure/go-workflow` 框架，在 `common/flowx` 包中提供与 go-zero 框架集成的 DAG 工作流封装。

## 需求

### 核心需求

1. **`NewFlow(opts ...FlowOption) *flow.Workflow`**：构造函数，自动注入日志拦截器
2. **日志拦截器**：打印每个步骤的开始/结束/耗时/错误，日志统一走 go-zero `logx` 体系
3. **Options 模式**：遵循 common 规范，`FlowOption` 写入 `FlowOptions` 构造配置结构体
4. **内置拦截器**：默认注册 `flow.LogStepFields` + `flow.LogAttemptField`（go-workflow 内置的 slog 字段注入）
5. **SubWorkflow 辅助**：提供 `flowx.NewSubFlow(opts ...FlowOption) *flow.Workflow`，内嵌 `flow.Workflow` 并可选设置名称

### 便捷配置选项

- `WithMaxConcurrency(n int)` — 最大并发步骤数
- `WithPanicRecovery()` — 捕获 panic 为 `ErrPanic` 而非崩溃
- `WithStepTimeout(d time.Duration)` — 全局默认步骤超时
- `WithRetry(attempts int)` — 全局默认重试次数

## 验收标准

1. `go build ./common/flowx/...` 编译通过
2. `go test ./common/flowx/...` 全部通过
3. 日志拦截器正确输出步骤开始/结束/错误
4. Options 默认值和自定义值行为正确
5. 包不引入 otel 等额外重依赖
