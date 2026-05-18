package runtime

import (
	"context"
	"fmt"
	"sort"
	"time"

	ctool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"zero-service/common/einox/metrics"
	"zero-service/common/einox/protocol"
)

type ToolRegistry struct {
	tools map[string]ctool.BaseTool
	order []string
}

func NewToolRegistry(tools ...ctool.BaseTool) (*ToolRegistry, error) {
	r := &ToolRegistry{tools: make(map[string]ctool.BaseTool)}
	for _, tool := range tools {
		if err := r.Register(context.Background(), tool); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *ToolRegistry) Register(ctx context.Context, tool ctool.BaseTool) error {
	if tool == nil {
		return fmt.Errorf("runtime: nil tool")
	}
	info, err := tool.Info(ctx)
	if err != nil {
		return err
	}
	if info == nil || info.Name == "" {
		return fmt.Errorf("runtime: tool info name is empty")
	}
	if _, ok := r.tools[info.Name]; ok {
		return fmt.Errorf("runtime: tool %q already registered", info.Name)
	}
	r.tools[info.Name] = tool
	r.order = append(r.order, info.Name)
	return nil
}

func (r *ToolRegistry) Infos(ctx context.Context) ([]*schema.ToolInfo, error) {
	infos := make([]*schema.ToolInfo, 0, len(r.tools))
	for _, name := range r.names() {
		tool := r.tools[name]
		info, err := tool.Info(ctx)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (r *ToolRegistry) names() []string {
	if len(r.order) == len(r.tools) {
		return append([]string(nil), r.order...)
	}
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *ToolRegistry) Run(ctx context.Context, name, args string) (string, error) {
	return r.run(ctx, name, args)
}

func (r *ToolRegistry) RunWithEmitter(ctx context.Context, em *protocol.Emitter, name, args string) (string, error) {
	return r.RunWithEmitterCallID(ctx, em, uuid.NewString(), name, args)
}

func (r *ToolRegistry) RunWithEmitterCallID(ctx context.Context, em *protocol.Emitter, callID, name, args string) (string, error) {
	if callID == "" {
		callID = uuid.NewString()
	}
	if em != nil {
		_ = em.Emit(protocol.EventToolCallStart, protocol.ToolCallStartData{CallID: callID, Tool: name, ArgsJSON: args})
	}
	result, err := r.run(ctx, name, args)
	if em != nil {
		data := protocol.ToolCallEndData{CallID: callID, Tool: name, Result: result}
		if err != nil {
			data.Error = err.Error()
			data.Result = ""
		}
		_ = em.Emit(protocol.EventToolCallEnd, data)
	}
	return result, err
}

func (r *ToolRegistry) run(ctx context.Context, name, args string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("runtime: tool %q not registered", name)
	}
	invokable, ok := tool.(ctool.InvokableTool)
	if !ok {
		return "", fmt.Errorf("runtime: tool %q is not invokable", name)
	}
	start := time.Now()
	result, err := invokable.InvokableRun(ctx, args)
	if err != nil {
		metrics.Global().RecordToolCall(ctx, name, "error", time.Since(start))
		return "", err
	}
	metrics.Global().RecordToolCall(ctx, name, "ok", time.Since(start))
	return result, nil
}

type StaticTool struct {
	Name   string
	Desc   string
	Result string
	Err    error
	Calls  *ToolCalls
}

type ToolCalls struct {
	Args []string
}

func (t StaticTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.Name, Desc: t.Desc}, nil
}

func (t StaticTool) InvokableRun(_ context.Context, args string, opts ...ctool.Option) (string, error) {
	_ = opts
	if t.Err != nil {
		return "", t.Err
	}
	if t.Calls != nil {
		t.Calls.Args = append(t.Calls.Args, args)
	}
	return t.Result, nil
}

var _ ctool.InvokableTool = StaticTool{}
