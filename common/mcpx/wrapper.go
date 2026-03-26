package mcpx

import (
	"context"
	"encoding/json"
	"fmt"

	"zero-service/common/antsx"
	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
)

// 全局进度事件发射器，按 progressToken 分发进度
var progressEmitter = antsx.NewEventEmitter[progressEvent]()

// progressEvent 进度事件（包内使用，不导出）
type progressEvent struct {
	Token    string
	Progress float64
	Total    float64
	Message  string
	Ctx      context.Context
}

// ctxProgressSenderKey context key for progress sender
type ctxProgressSenderKey struct{}

// ProgressSender 进度发送器，具备发布和订阅能力
// Emit - 发布进度到 emitter
// Start - 订阅并转发到 Client，返回 cancel 函数
type ProgressSender struct {
	token   string
	ctx     context.Context
	session *mcp.ServerSession
	cancel  func()
}

// Emit 发送进度到 emitter，携带 ctx 以传递链路信息
func (p *ProgressSender) Emit(progress, total float64, message string) {
	logx.WithContext(p.ctx).Debugf("[mcpx] progress emit: token=%s, progress=%.0f/%.0f, msg=%s", p.token, progress, total, message)
	progressEmitter.Emit(p.token, progressEvent{
		Token:    p.token,
		Progress: progress,
		Total:    total,
		Message:  message,
		Ctx:      p.ctx,
	})
}

// Start 订阅进度事件并转发到 MCP Client
func (p *ProgressSender) Start() {
	logx.WithContext(p.ctx).Infof("[mcpx] progress start: token=%s", p.token)
	ch, unsub := progressEmitter.Subscribe(p.token)
	p.cancel = unsub
	threading.GoSafe(func() {
		start := timex.Now()
		defer logx.WithContext(p.ctx).WithDuration(timex.Since(start)).Infof("[mcpx] progress stopped chan: token=%s", p.token)
		for event := range ch {
			p.session.NotifyProgress(event.Ctx, &mcp.ProgressNotificationParams{
				ProgressToken: event.Token,
				Progress:      event.Progress,
				Total:         event.Total,
				Message:       event.Message,
			})
		}
	})
}

// Stop 停止订阅
func (p *ProgressSender) Stop() {
	if p.cancel != nil {
		logx.WithContext(p.ctx).Debugf("[mcpx] progress stop: token=%s", p.token)
		p.cancel()
	}
}

// GetProgressSender 从 context 获取进度发送器
func GetProgressSender(ctx context.Context) *ProgressSender {
	if sender, ok := ctx.Value(ctxProgressSenderKey{}).(*ProgressSender); ok {
		return sender
	}
	return nil
}

// wrapperConfig wrapper 配置项
type wrapperConfig struct {
	extractUserCtx bool
}

// Option 函数选项模式
type Option func(*wrapperConfig)

// WithExtractUserCtx 提取用户上下文选项。
// 启用后，会从 _meta 中提取用户身份（user-id, user-name 等）到 context values，
// 业务层可调用 ctxprop.InjectToGrpcMD(ctx) 透传到 gRPC metadata。
func WithExtractUserCtx() Option {
	return func(c *wrapperConfig) {
		c.extractUserCtx = true
	}
}

// CallToolWrapper 简化的 MCP tool handler 包装器。
//
// 设计理念：
// - MCP 层只做 trace 传播和 _meta 透传
// - 不做用户身份鉴权，由业务层自行处理
//
// 上下文传递机制：
//
//	客户端通过 params._meta 传递用户上下文（user_id 等）和链路信息（traceparent）
//	服务端将 _meta 整体存入 ctx，key 为 ctxdata.CtxMetaKey
//	业务层通过 ctxdata.GetMeta(ctx) 获取并自行解析
//
// 信任边界：
//
//	信任边界 1：MCP Server 服务 Token 鉴权（验证是可信 Client）
//	信任边界 2：业务层用户鉴权（从 _meta 中解析用户身份）
//
// 使用方式：
//
//	mcp.AddTool(server, tool, mcpx.CallToolWrapper(handler))
//
// 带用户上下文提取（用于调用 gRPC 服务）：
//
//	mcp.AddTool(server, tool, mcpx.CallToolWrapper(handler, mcpx.WithExtractUserCtx()))
func CallToolWrapper[In, Out any](
	h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
	opts ...Option,
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	var cfg wrapperConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (result *mcp.CallToolResult, out Out, err error) {
		name := req.Params.Name
		start := timex.Now()

		logx.WithContext(ctx).Debugf("[mcpx] call tool wrapper %v", name)

		defer func() {
			if err != nil {
				logx.WithContext(ctx).WithDuration(timex.Since(start)).Errorf("[mcpx] call tool %v failed, args=%s: %v",
					name, marshalArgs(args), err)
			} else {
				logx.WithContext(ctx).WithDuration(timex.Since(start)).Infof("[mcpx] call tool %v success",
					name)
			}
		}()

		var meta map[string]any

		// 从 _meta 中提取链路信息，注入 trace context
		if req.Params != nil {
			meta = req.Params.GetMeta()
			if len(meta) > 0 {
				ctx = ctxprop.ExtractTraceFromMeta(ctx, meta)
			}
		}

		// 将 _meta 整体存入 ctx，供业务层使用
		// 业务层通过 ctxdata.GetMeta(ctx) 获取，自行解析用户身份
		if len(meta) > 0 {
			ctx = context.WithValue(ctx, ctxdata.CtxMetaKey, meta)
		}

		// 可选：从 _meta 中提取用户上下文到 context values，供 gRPC 调用使用
		// 业务层调用 ctxprop.InjectToGrpcMD(ctx) 将用户身份透传到 gRPC metadata
		if cfg.extractUserCtx && len(meta) > 0 {
			ctx = ctxprop.ExtractFromMeta(ctx, meta)
		}

		// 将进度发送器存入 ctx，供业务层使用
		// 业务层通过 GetProgressSender(ctx) 获取，调用 Emit 发送进度
		var progressSender *ProgressSender
		if req.Session != nil {
			if token := req.Params.GetProgressToken(); token != nil {
				tokenStr := fmt.Sprintf("%v", token)
				progressSender = &ProgressSender{
					token:   tokenStr,
					ctx:     ctx,
					session: req.Session,
				}
				progressSender.Start()
				ctx = context.WithValue(ctx, ctxProgressSenderKey{}, progressSender)
			}
		}

		result, out, err = h(ctx, req, args)

		// 工具执行完毕后停止进度订阅
		if progressSender != nil {
			progressSender.Stop()
		}

		return result, out, err
	}
}

// marshalArgs 将 args 序列化为 JSON 字符串。
func marshalArgs(args any) string {
	data, err := json.Marshal(args)
	if err != nil {
		return "<marshal error>"
	}
	return string(data)
}
