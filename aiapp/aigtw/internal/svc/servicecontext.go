// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"time"

	"zero-service/aiapp/aigtw/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/antsx"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/zrpc"
)

// ChunkEvent 流式事件结构，用于 EventEmitter 传输 SSE chunk 数据
type ChunkEvent struct {
	Data  any    // ChatCompletionChunk 或其他结构体，将被 JSON 序列化
	Error error  // 非 nil 表示出错，流应终止
	Done  bool   // true 表示流结束
}

type ServiceContext struct {
	Config     config.Config
	ZeroRpcCli zerorpc.ZerorpcClient
	Emitter    *antsx.EventEmitter[ChunkEvent]
	PendingReg *antsx.PendingRegistry[string]
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		ZeroRpcCli: zerorpc.NewZerorpcClient(zrpc.MustNewClient(c.ZeroRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
		Emitter:    antsx.NewEventEmitter[ChunkEvent](),
		PendingReg: antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(60 * time.Second)),
	}
}
