package runner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func ExampleSoloRunner() {
	ctx := context.Background()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "My AI Assistant",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	})
	if err != nil {
		fmt.Printf("create agent failed: %v\n", err)
		return
	}

	runner, err := NewSoloRunner(ctx, agent,
		WithEnableStreaming(true),
		WithEnableHistory(true),
		WithMaxHistory(20),
	)
	if err != nil {
		fmt.Printf("create runner failed: %v\n", err)
		return
	}

	_ = runner
}

func ExampleSoloRunner_Query() {
	ctx := context.Background()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "My AI Assistant",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	})
	if err != nil {
		fmt.Printf("create agent failed: %v\n", err)
		return
	}

	runner, err := NewSoloRunner(ctx, agent)
	if err != nil {
		fmt.Printf("create runner failed: %v\n", err)
		return
	}

	iter, err := runner.Query(ctx, "Hello, how are you?")
	if err != nil {
		fmt.Printf("query failed: %v\n", err)
		return
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			fmt.Printf("error: %v\n", event.Err)
			return
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err != nil {
				continue
			}
			fmt.Printf("Response: %s\n", msg.Content)
		}
	}
}

func ExampleSoloRunner_QueryStream() {
	ctx := context.Background()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "My AI Assistant",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	})
	if err != nil {
		fmt.Printf("create agent failed: %v\n", err)
		return
	}

	runner, err := NewSoloRunner(ctx, agent)
	if err != nil {
		fmt.Printf("create runner failed: %v\n", err)
		return
	}

	messages := []*schema.Message{
		schema.UserMessage("Hello!"),
	}

	iter, err := runner.QueryStream(ctx, "session-123", messages)
	if err != nil {
		fmt.Printf("query stream failed: %v\n", err)
		return
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			fmt.Printf("error: %v\n", event.Err)
			return
		}
		if event.Output != nil {
			fmt.Printf("event: %+v\n", event)
		}
	}
}

func ExampleSoloRunner_Chat() {
	ctx := context.Background()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "My AI Assistant",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	})
	if err != nil {
		fmt.Printf("create agent failed: %v\n", err)
		return
	}

	runner, err := NewSoloRunner(ctx, agent,
		WithEnableHistory(true),
		WithMaxHistory(10),
	)
	if err != nil {
		fmt.Printf("create runner failed: %v\n", err)
		return
	}

	result, err := runner.Chat(ctx, "user-1", "session-1", "Hello, how are you?")
	if err != nil {
		fmt.Printf("chat failed: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", result.Response)
}

func ExampleSoloRunner_StreamToA2UI() {
	ctx := context.Background()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MyAgent",
		Description: "My AI Assistant",
		Instruction: "You are a helpful assistant.",
		Model:       nil,
	})
	if err != nil {
		fmt.Printf("create agent failed: %v\n", err)
		return
	}

	runner, err := NewSoloRunner(ctx, agent)
	if err != nil {
		fmt.Printf("create runner failed: %v\n", err)
		return
	}

	messages := []*schema.Message{
		schema.UserMessage("Hello!"),
	}

	var buf bytes.Buffer
	response, interruptID, err := runner.StreamToA2UI(ctx, &buf, "session-123", messages)
	if err != nil {
		fmt.Printf("stream to a2ui failed: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
	fmt.Printf("InterruptID: %s\n", interruptID)
	fmt.Printf("Output: %s\n", buf.String())
}
