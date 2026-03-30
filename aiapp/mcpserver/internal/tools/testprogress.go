package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"zero-service/common/mcpx"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

// TestProgressArgs 测试进度通知参数
type TestProgressArgs struct {
	Steps    int    `json:"steps" jsonschema:"总步数"`
	Interval int    `json:"interval" jsonschema:"每步间隔毫秒"`
	Message  string `json:"message,omitempty" jsonschema:"可选消息"`
}

// RegisterTestProgress 注册测试进度通知工具
func RegisterTestProgress(server *sdkmcp.Server) {
	progressTool := &sdkmcp.Tool{
		Name:        "test_progress",
		Description: "测试进度通知功能，模拟长时间运行的任务并发送进度更新, Interval 是毫秒单位, 这是一个非常耗时的任务,需要开启进度通知",
	}

	progressHandler := func(ctx context.Context, req *sdkmcp.CallToolRequest, args TestProgressArgs) (*sdkmcp.CallToolResult, any, error) {
		steps := args.Steps
		if steps <= 0 {
			steps = 5
		}
		interval := args.Interval
		if interval <= 0 {
			interval = 500 // 默认 500ms
		}

		// 获取进度发送器（包含 ctx，带 trace 信息）
		sender := mcpx.GetProgressSender(ctx)

		logx.WithContext(ctx).Infof("[test_progress] 开始测试, steps=%d, interval=%dms", steps, interval)

		threading.RunSafe(func() {
			for i := 1; i <= steps; i++ {
				progress := float64(i)
				total := float64(steps)
				msg := args.Message
				if msg == "" {
					msg = "处理中...当前 step: " + strconv.Itoa(i)
				}

				// 发送进度通知
				if sender != nil {
					sender.Emit(progress, total, msg)
				} else {
					logx.WithContext(ctx).Debugf("[test_progress] 进度: %.0f/%.0f - %s", progress, total, msg)
				}

				// 模拟耗时操作
				time.Sleep(time.Duration(interval) * time.Millisecond)
			}
		})
		result := "测试进度执行中,共 %d 步,token:%s"
		sendToken := ""
		if sender != nil {
			sendToken = sender.GetToken()
		}
		textcontext := sdkmcp.TextContent{Text: fmt.Sprintf(result, steps, sendToken)}
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&textcontext,
			},
		}, nil, nil
	}

	sdkmcp.AddTool(server, progressTool, mcpx.CallToolWrapper(progressHandler, mcpx.WithExtractUserCtx()))
}
