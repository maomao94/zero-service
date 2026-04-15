// Package quick 快速开发包 - 一行代码创建智能体
//
// 提供简洁的接口快速创建 Agent、聊天机器人和工具。
//
// 示例：
//
//	// 1. 创建聊天机器人
//	bot := quick.NewChatBot(ctx, &quick.Config{
//	    Provider: "ark",
//	    APIKey:   "your-api-key",
//	    Model:    "deepseek-v3-2-251201",
//	})
//
//	// 2. 添加工具
//	bot.WithTools(calculatorTool, dateTimeTool)
//
//	// 3. 对话
//	resp, err := bot.Chat(ctx, "1+1等于几？")
//
//	// 或者流式对话
//	bot.ChatStream(ctx, "写一个Hello World", func(chunk string) {
//	    print(chunk)
//	})
package quick
