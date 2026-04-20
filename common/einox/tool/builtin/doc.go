// Package builtin 注册 einox 内置工具（按 CapCompute / CapIO / CapHuman 分类）。
//
// # 软失败约定（避免 ADK 整轮 NodeRunError）
//
// 对「可预期的坏输入」（解析失败、参数不合法等），工具实现不要对 Invoke 返回 (nil, goErr)，
// 否则流式工具节点会失败并中断会话。应改为返回成功，并在 JSON 结果里带非空字符串字段 error
// （可选字段 result）。协议层会把该 JSON 映射到 tool.call.end 的 error 字段，前端用告警样式展示。
//
// 当前采用此约定的工具：calculator。其它工具仅在真实异常（如 rand.Read 失败）时使用 (nil, err)。
package builtin
