// Package modeweb 提供 AgentMode 在 Solo HTTP 网关侧的字符串约定，以及与 Workflow 拓扑、指标标签的映射。
// 放在 internal 外，供 aigtw 与 aisolo/internal/modes 共用。
package modeweb

import (
	"strings"

	"zero-service/aiapp/aisolo/aisolo"
)

// workflowAgentModes 与 aisolo.proto 中 Workflow 家族一致；新增 Workflow 枚举时须同步此处与 Topology。
var workflowAgentModes = []aisolo.AgentMode{
	aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL,
	aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL,
	aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP,
}

// WorkflowAgentModes 返回 Workflow 家族枚举切片副本（勿修改返回值元素以冒充扩展；应改 proto 与本包）。
func WorkflowAgentModes() []aisolo.AgentMode {
	out := make([]aisolo.AgentMode, len(workflowAgentModes))
	copy(out, workflowAgentModes)
	return out
}

// Topology 返回 ADK Workflow 拓扑名；非 Workflow 返回空。
func Topology(m aisolo.AgentMode) string {
	switch m {
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL:
		return "sequential"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL:
		return "parallel"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP:
		return "loop"
	default:
		return ""
	}
}

// IsWorkflow 是否为 Workflow 家族枚举之一。
func IsWorkflow(m aisolo.AgentMode) bool {
	return Topology(m) != ""
}

// TurnMetricTag 供 turn 指标等使用的短标签；非 Workflow 返回空。
func TurnMetricTag(m aisolo.AgentMode) string {
	switch Topology(m) {
	case "sequential":
		return "workflow_sequential"
	case "parallel":
		return "workflow_parallel"
	case "loop":
		return "workflow_loop"
	default:
		return ""
	}
}

// Parse 将 Solo 网关 JSON 中的 mode 字符串解析为 gRPC AgentMode。
func Parse(s string) aisolo.AgentMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "default":
		return aisolo.AgentMode_AGENT_MODE_UNSPECIFIED
	case "agent":
		return aisolo.AgentMode_AGENT_MODE_AGENT
	case "workflow-sequential", "workflow_sequential", "workflow-seq":
		return aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL
	case "workflow-parallel", "workflow_parallel":
		return aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL
	case "workflow-loop", "workflow_loop":
		return aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP
	case "supervisor":
		return aisolo.AgentMode_AGENT_MODE_SUPERVISOR
	case "plan", "plan-execute", "planexecute":
		return aisolo.AgentMode_AGENT_MODE_PLAN
	case "deep", "deepagent", "deep-agent":
		return aisolo.AgentMode_AGENT_MODE_DEEP
	default:
		return aisolo.AgentMode_AGENT_MODE_UNSPECIFIED
	}
}

// ToSoloString 将 AgentMode 转为 Solo 网关 JSON 使用的 mode 字符串。
func ToSoloString(m aisolo.AgentMode) string {
	switch m {
	case aisolo.AgentMode_AGENT_MODE_AGENT:
		return "agent"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL:
		return "workflow-sequential"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL:
		return "workflow-parallel"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP:
		return "workflow-loop"
	case aisolo.AgentMode_AGENT_MODE_SUPERVISOR:
		return "supervisor"
	case aisolo.AgentMode_AGENT_MODE_PLAN:
		return "plan"
	case aisolo.AgentMode_AGENT_MODE_DEEP:
		return "deep"
	default:
		return ""
	}
}
