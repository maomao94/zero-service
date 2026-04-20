package modeweb

import (
	"testing"

	"zero-service/aiapp/aisolo/aisolo"
)

func TestParseToSoloStringRoundTrip(t *testing.T) {
	aliases := map[string]aisolo.AgentMode{
		"agent":               aisolo.AgentMode_AGENT_MODE_AGENT,
		"workflow-sequential": aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL,
		"workflow_sequential": aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL,
		"workflow-seq":        aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL,
		"workflow-parallel":   aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL,
		"workflow_parallel":   aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL,
		"workflow-loop":       aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP,
		"workflow_loop":       aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP,
		"supervisor":          aisolo.AgentMode_AGENT_MODE_SUPERVISOR,
		"plan":                aisolo.AgentMode_AGENT_MODE_PLAN,
		"plan-execute":        aisolo.AgentMode_AGENT_MODE_PLAN,
		"deep":                aisolo.AgentMode_AGENT_MODE_DEEP,
		"deep-agent":          aisolo.AgentMode_AGENT_MODE_DEEP,
	}
	for in, want := range aliases {
		in := in
		want := want
		t.Run(in, func(t *testing.T) {
			got := Parse(in)
			if got != want {
				t.Fatalf("Parse(%q)=%v want %v", in, got, want)
			}
			s := ToSoloString(got)
			again := Parse(s)
			if again != got {
				t.Fatalf("roundtrip: %q -> %v -> %q -> %v", in, got, s, again)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	if Parse("") != aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		t.Fatal()
	}
	if Parse("  DEFAULT  ") != aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		t.Fatal()
	}
	if ToSoloString(aisolo.AgentMode_AGENT_MODE_UNSPECIFIED) != "" {
		t.Fatal()
	}
}

func TestWorkflowAgentModesLen(t *testing.T) {
	if n := len(WorkflowAgentModes()); n != 3 {
		t.Fatalf("WorkflowAgentModes: want 3, got %d", n)
	}
}

func TestTopologyAndTurnMetricTag(t *testing.T) {
	if Topology(aisolo.AgentMode_AGENT_MODE_WORKFLOW_SEQUENTIAL) != "sequential" {
		t.Fatal()
	}
	if TurnMetricTag(aisolo.AgentMode_AGENT_MODE_WORKFLOW_PARALLEL) != "workflow_parallel" {
		t.Fatal()
	}
	if IsWorkflow(aisolo.AgentMode_AGENT_MODE_AGENT) {
		t.Fatal()
	}
	if !IsWorkflow(aisolo.AgentMode_AGENT_MODE_WORKFLOW_LOOP) {
		t.Fatal()
	}
}
