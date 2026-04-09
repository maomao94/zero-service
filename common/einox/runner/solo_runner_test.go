package runner

import (
	"bytes"
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestNewSoloRunner(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)
	assert.NotNil(t, runner)
	assert.NotNil(t, runner.GetAgent())
	assert.NotNil(t, runner.GetRunner())
	assert.NotNil(t, runner.GetConfig())
}

func TestNewSoloRunnerWithOptions(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent,
		WithEnableStreaming(true),
		WithEnableHistory(true),
		WithMaxHistory(30),
	)

	assert.NoError(t, err)
	assert.NotNil(t, runner)
	assert.True(t, runner.config.EnableStreaming)
	assert.True(t, runner.config.EnableHistory)
	assert.Equal(t, 30, runner.config.MaxHistory)
}

func TestSoloRunnerQuery(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)

	iter, err := runner.Query(ctx, "hello")
	assert.NoError(t, err)
	assert.NotNil(t, iter)
}

func TestSoloRunnerQueryStream(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)

	messages := []*schema.Message{
		schema.UserMessage("hello"),
	}

	iter, err := runner.QueryStream(ctx, "session-1", messages)
	assert.NoError(t, err)
	assert.NotNil(t, iter)
}

func TestSoloRunnerQueryValidation(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)

	_, err = runner.Query(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input is required")

	_, err = runner.QueryStream(ctx, "session-1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages is required")

	_, err = runner.QueryStream(ctx, "session-1", []*schema.Message{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages is required")
}

func TestSoloRunnerStreamToA2UI(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)

	messages := []*schema.Message{
		schema.UserMessage("hello"),
	}

	_, err = runner.StreamToA2UI(ctx, nil, "session-1", messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer is required")

	var buf bytes.Buffer
	_, err = runner.StreamToA2UI(ctx, &buf, "session-1", messages)
	assert.NoError(t, err)
}

func TestSoloRunnerSetStore(t *testing.T) {
	ctx := context.Background()

	agent, err := createMockAgent(ctx)
	if err != nil {
		t.Skipf("skip test: %v", err)
	}

	runner, err := NewSoloRunner(ctx, agent)
	assert.NoError(t, err)

	store := runner.store
	assert.NotNil(t, store)

	runner.SetStore(nil)
	assert.Nil(t, runner.store)

	runner.SetStore(store)
	assert.NotNil(t, runner.store)
}

func createMockAgent(ctx context.Context) (adk.Agent, error) {
	agentCfg := &adk.ChatModelAgentConfig{
		Name:        "TestAgent",
		Description: "Test Agent for unit testing",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	}

	agent, err := adk.NewChatModelAgent(ctx, agentCfg)
	if err != nil {
		return nil, err
	}

	return agent, nil
}
