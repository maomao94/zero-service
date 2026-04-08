package einox_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"zero-service/common/einox/a2ui"
	"zero-service/common/einox/agent"
	"zero-service/common/einox/memory"
	"zero-service/common/einox/model"

	"github.com/cloudwego/eino/schema"
)

// 测试配置（从环境变量读取，请设置环境变量）
const (
	testModel = "deepseek-v3-2-251201" // DeepSeek V3 模型
)

// getTestAPIKey 获取测试 API Key（从环境变量）
func getTestAPIKey() string {
	if key := os.Getenv("ARK_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("DEEPSEEK_API_KEY")
}

// =============================================================================
// Test 1: ChatModel 创建测试
// =============================================================================

func TestChatModel_NewChatModel(t *testing.T) {
	ctx := context.Background()

	// 火山 ARK DeepSeek 端点
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 测试使用 Config 创建
	cfg := model.Config{
		Provider:    model.ProviderDeepSeek,
		APIKey:      getTestAPIKey(),
		Model:       testModel,
		BaseURL:     arkBaseURL, // 火山 ARK Base URL
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	chatModel, err := model.NewChatModel(ctx, cfg)
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}
	if chatModel == nil {
		t.Fatal("ChatModel 不能为空")
	}

	log.Printf("✅ TestChatModel_NewChatModel: ChatModel 创建成功 (BaseURL: %s)", arkBaseURL)
}

// =============================================================================
// Test 2: ChatModel Option 模式测试（使用 DeepSeek Provider）
// =============================================================================

func TestChatModel_NewChatModelByOption(t *testing.T) {
	// 火山 ARK DeepSeek 端点
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 测试使用 Option 创建（内部使用 context.Background()）
	chatModel, err := model.NewChatModelByOption(
		model.ProviderDeepSeek,
		model.WithAPIKey(getTestAPIKey()),
		model.WithModel(testModel),
		model.WithBaseURL(arkBaseURL),
		model.WithTemperature(0.7),
		model.WithMaxTokens(2048),
	)
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}
	if chatModel == nil {
		t.Fatal("ChatModel 不能为空")
	}

	log.Printf("✅ TestChatModel_NewChatModelByOption: Option 模式创建成功 (BaseURL: %s)", arkBaseURL)
}

// =============================================================================
// Test 2.1: ARK Provider 专用测试（简化配置）
// =============================================================================

func TestChatModel_ARKProvider(t *testing.T) {
	// 使用 ARK Provider，只需提供 API Key 和模型名称
	// BaseURL 和 Region 自动配置

	// Config 模式
	cfg := model.Config{
		Provider:  model.ProviderArk,
		APIKey:    getTestAPIKey(),
		Model:     testModel,
		ArkRegion: "cn-beijing", // 可选，默认就是 cn-beijing
	}

	chatModel, err := model.NewChatModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("创建 ARK ChatModel 失败: %v", err)
	}
	if chatModel == nil {
		t.Fatal("ARK ChatModel 不能为空")
	}

	log.Printf("✅ TestChatModel_ARKProvider: ARK Provider (Config 模式) 创建成功")

	// Option 模式
	chatModel2, err := model.NewChatModelByOption(
		model.ProviderArk,
		model.WithAPIKey(getTestAPIKey()),
		model.WithModel(testModel),
		// 不需要指定 BaseURL，自动使用 ARK 端点
	)
	if err != nil {
		t.Fatalf("创建 ARK ChatModel (Option) 失败: %v", err)
	}
	if chatModel2 == nil {
		t.Fatal("ARK ChatModel 不能为空")
	}

	log.Printf("✅ TestChatModel_ARKProvider: ARK Provider (Option 模式) 创建成功")
}

// =============================================================================
// Test 3: Agent 基本对话测试
// =============================================================================

func TestAgent_SimpleRun(t *testing.T) {
	ctx := context.Background()
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 1. 创建 ChatModel
	chatModel, err := model.NewChatModel(ctx, model.Config{
		Provider:  model.ProviderDeepSeek,
		APIKey:    getTestAPIKey(),
		Model:     testModel,
		BaseURL:   arkBaseURL,
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 2. 创建 Agent
	a, err := agent.New(ctx,
		agent.WithName("test-agent"),
		agent.WithInstruction("你是一个友好的 AI 助手，用简洁的语言回答问题。"),
		agent.WithModel(chatModel),
		agent.WithStream(true),
	)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	// 3. 运行对话
	result, err := a.Run(ctx, "你好，请介绍一下你自己")
	if err != nil {
		t.Fatalf("Agent 运行失败: %v", err)
	}

	if result == nil {
		t.Fatal("结果不能为空")
	}
	if result.Response == "" {
		t.Fatal("响应内容不能为空")
	}
	if result.Err != nil {
		t.Fatalf("响应包含错误: %v", result.Err)
	}

	log.Printf("✅ TestAgent_SimpleRun: 单轮对话成功")
	log.Printf("   响应: %s", result.Response)
}

// =============================================================================
// Test 4: Agent 带历史消息测试
// =============================================================================

func TestAgent_RunWithHistory(t *testing.T) {
	ctx := context.Background()
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 1. 创建 ChatModel
	chatModel, err := model.NewChatModel(ctx, model.Config{
		Provider:  model.ProviderDeepSeek,
		APIKey:    getTestAPIKey(),
		Model:     testModel,
		BaseURL:   arkBaseURL,
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 2. 创建 Agent（使用 MemoryStorage）
	a, err := agent.New(ctx,
		agent.WithName("history-agent"),
		agent.WithDescription("一个友好的 AI 助手，支持多轮对话"),
		agent.WithInstruction("你是一个友好的 AI 助手，记住之前的对话内容。"),
		agent.WithModel(chatModel),
		agent.WithMemoryStorage(), // 使用默认内存存储
	)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	sessionID := "test-session-001"

	// 3. 第一轮对话
	result1, err := a.RunWithHistory(ctx, "我叫张三，请记住我的名字", agent.WithSessionID(sessionID))
	if err != nil {
		t.Fatalf("第一轮对话失败: %v", err)
	}
	log.Printf("✅ 第一轮对话成功: %s", result1.Response)

	// 4. 第二轮对话（测试是否记住名字）
	result2, err := a.RunWithHistory(ctx, "我叫什么名字？", agent.WithSessionID(sessionID))
	if err != nil {
		t.Fatalf("第二轮对话失败: %v", err)
	}

	// 检查是否记住了名字
	if result2.Err != nil {
		t.Fatalf("第二轮对话包含错误: %v", result2.Err)
	}

	log.Printf("✅ TestAgent_RunWithHistory: 多轮对话成功")
	log.Printf("   第二轮响应: %s", result2.Response)

	// 5. 清理
	_ = a.ClearMemory(ctx, sessionID)
}

// =============================================================================
// Test 5: Agent 流式输出测试
// =============================================================================

func TestAgent_StreamRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 1. 创建 ChatModel
	chatModel, err := model.NewChatModel(ctx, model.Config{
		Provider:  model.ProviderDeepSeek,
		APIKey:    getTestAPIKey(),
		Model:     testModel,
		BaseURL:   arkBaseURL,
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 2. 创建 Agent
	a, err := agent.New(ctx,
		agent.WithName("stream-agent"),
		agent.WithDescription("一个流式输出的 AI 助手"),
		agent.WithInstruction("你是一个友好的 AI 助手，用详细的语言回答问题。"),
		agent.WithModel(chatModel),
		agent.WithStream(true),
	)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	// 3. 流式运行
	stream, err := a.RunStream(ctx, "请给我讲一个关于AI的短故事")
	if err != nil {
		t.Fatalf("流式运行失败: %v", err)
	}

	var fullResponse string
	chunks := 0

	log.Printf("📡 开始接收流式响应...")
	for result := range stream {
		if result.Err != nil {
			t.Fatalf("流式响应错误: %v", result.Err)
		}
		if result.Response != "" {
			fullResponse += result.Response
			chunks++
			// 打印前几个 chunk
			if chunks <= 3 {
				log.Printf("   Chunk %d: %s", chunks, truncate(result.Response, 100))
			}
		}
	}

	log.Printf("✅ TestAgent_StreamRun: 流式输出成功")
	log.Printf("   总 Chunk 数: %d", chunks)
	log.Printf("   完整响应长度: %d 字符", len(fullResponse))
	log.Printf("   响应预览: %s...", truncate(fullResponse, 200))
}

// =============================================================================
// Test 6: Memory Storage 测试
// =============================================================================

func TestMemory_Storage(t *testing.T) {
	ctx := context.Background()

	// 创建带选项的存储
	storage := memory.NewMemoryStorage(
		memory.WithMaxSize(10),
		memory.WithWindowSize(5),
	)

	// 测试保存和获取
	msg := &schema.Message{
		Role:    schema.User,
		Content: "测试消息",
	}

	err := storage.Save(ctx, "session-1", msg)
	if err != nil {
		t.Fatalf("保存消息失败: %v", err)
	}

	msgs, err := storage.Get(ctx, "session-1", 0)
	if err != nil {
		t.Fatalf("获取消息失败: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("期望 1 条消息，实际 %d 条", len(msgs))
	}

	// 测试计数
	count, err := storage.Count(ctx, "session-1")
	if err != nil {
		t.Fatalf("计数失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("期望计数为 1，实际为 %d", count)
	}

	// 测试清除
	err = storage.Clear(ctx, "session-1")
	if err != nil {
		t.Fatalf("清除消息失败: %v", err)
	}

	count, _ = storage.Count(ctx, "session-1")
	if count != 0 {
		t.Fatalf("清除后计数应为 0，实际为 %d", count)
	}

	log.Printf("✅ TestMemory_Storage: 存储操作成功")
}

// =============================================================================
// Test 7: SSE Writer 测试
// =============================================================================

func TestA2UI_SSEWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := a2ui.NewSSEWriter(&buf)

	// 发送多个事件
	events := []*a2ui.Event{
		a2ui.NewStartEvent(),
		a2ui.NewTextEvent("你好", false),
		a2ui.NewTextEvent("世界", true),
		a2ui.NewEndEvent(),
	}

	for _, event := range events {
		err := writer.Write(event)
		if err != nil {
			t.Fatalf("写入事件失败: %v", err)
		}
	}

	// 验证输出
	output := buf.String()
	if output == "" {
		t.Fatal("SSE 输出为空")
	}

	// 解析 SSE 格式
	lines := bytes.Split([]byte(output), []byte("\n"))
	eventCount := 0
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("data: ")) {
			eventCount++
			var event a2ui.Event
			if err := json.Unmarshal(bytes.TrimPrefix(line, []byte("data: ")), &event); err != nil {
				t.Logf("解析事件失败: %v", err)
			}
		}
	}

	log.Printf("✅ TestA2UI_SSEWriter: SSE 写入成功")
	log.Printf("   写入事件数: %d", len(events))
	log.Printf("   输出预览: %s", truncate(output, 200))
}

// =============================================================================
// Test 8: Streamer 测试
// =============================================================================

func TestA2UI_Streamer(t *testing.T) {
	var buf bytes.Buffer
	streamer := a2ui.NewStreamer(&buf)

	// 发送事件
	_ = streamer.EmitStart()
	_ = streamer.EmitText("测试", false)
	_ = streamer.EmitMarkdown("# 标题", false)
	_ = streamer.EmitEnd()

	// 获取所有事件
	events := streamer.GetEvents()
	if len(events) != 4 {
		t.Fatalf("期望 4 个事件，实际 %d 个", len(events))
	}

	log.Printf("✅ TestA2UI_Streamer: Streamer 测试成功")
	log.Printf("   事件数: %d", len(events))
}

// =============================================================================
// Test 9: 完整流程测试（Agent + Streamer + SSE）
// =============================================================================

func TestFullFlow_AgentWithSSE(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	arkBaseURL := "https://ark.cn-beijing.volces.com/api/v3"

	// 1. 创建 ChatModel
	chatModel, err := model.NewChatModel(ctx, model.Config{
		Provider:  model.ProviderDeepSeek,
		APIKey:    getTestAPIKey(),
		Model:     testModel,
		BaseURL:   arkBaseURL,
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatalf("创建 ChatModel 失败: %v", err)
	}

	// 2. 创建 Agent
	a, err := agent.New(ctx,
		agent.WithName("sse-agent"),
		agent.WithDescription("一个支持 SSE 输出的 AI 助手"),
		agent.WithInstruction("你是一个友好的 AI 助手。"),
		agent.WithModel(chatModel),
		agent.WithStream(true),
		agent.WithMemoryStorage(),
	)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	// 3. 创建 SSE Streamer
	var buf bytes.Buffer
	streamer := a2ui.NewStreamer(&buf)

	// 4. 流式运行并通过 SSE 输出
	stream, err := a.RunStream(ctx, "用一句话介绍 Go 语言")
	if err != nil {
		t.Fatalf("流式运行失败: %v", err)
	}

	// 发送开始事件
	_ = streamer.EmitStart()

	// 处理流式响应
	var fullResponse string
	for result := range stream {
		if result.Err != nil {
			_ = streamer.EmitError(result.Err.Error())
			break
		}
		if result.Response != "" {
			fullResponse += result.Response
			// 实时发送 SSE 事件
			_ = streamer.EmitText(result.Response, false)
		}
	}

	// 发送结束事件
	_ = streamer.EmitEnd()
	_ = streamer.Flush()

	log.Printf("✅ TestFullFlow_AgentWithSSE: 完整流程成功")
	log.Printf("   SSE 输出长度: %d 字符", buf.Len())
	log.Printf("   响应内容: %s...", truncate(fullResponse, 200))
}

// =============================================================================
// Test 10: Agent 工具调用测试（可选）
// =============================================================================

// 注意：此测试需要定义工具，实际使用时取消注释
// func TestAgent_WithTools(t *testing.T) {
//     ctx := context.Background()
//
//     // 定义一个简单工具
//     calculator := &tool.BaseTool{
//         Name:        "calculator",
//         Description: "执行数学计算",
//         Parameters:  map[string]any{"expression": "string"},
//     }
//
//     chatModel, _ := model.NewChatModel(ctx, model.Config{...})
//
//     a, _ := agent.New(ctx,
//         agent.WithName("tool-agent"),
//         agent.WithModel(chatModel),
//         agent.WithTools(calculator),
//     )
//
//     result, _ := a.Run(ctx, "计算 2 + 2")
//     log.Printf("工具调用结果: %s", result.Response)
// }

// =============================================================================
// 辅助函数
// =============================================================================

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
