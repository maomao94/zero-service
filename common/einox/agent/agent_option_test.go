package agent

import (
	"testing"

	"github.com/cloudwego/eino/components/model"
)

func TestWithModelOptionStoresRuntimeOptions(t *testing.T) {
	temp := float32(0.25)
	opts := newOptions(WithModelOption(model.WithTemperature(temp)))
	if len(opts.modelOptions) != 1 {
		t.Fatalf("modelOptions len = %d, want 1", len(opts.modelOptions))
	}
}

func TestAgentModelOptionsReturnsCopy(t *testing.T) {
	temp := float32(0.25)
	opts := newOptions(WithModelOption(model.WithTemperature(temp)))
	ag := &Agent{opts: *opts}

	got := ag.ModelOptions()
	if len(got) != 1 {
		t.Fatalf("ModelOptions len = %d, want 1", len(got))
	}
	got = append(got, model.WithTemperature(0.5))
	if len(ag.opts.modelOptions) != 1 {
		t.Fatal("ModelOptions returned internal slice")
	}
}
