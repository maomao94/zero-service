package svc

import (
	"context"
	"math"
	"sync"
	"zero-service/app/iecstash/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/facade/streamevent/streamevent"

	"github.com/tidwall/gjson"
	"github.com/zeromicro/go-zero/core/executors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config         config.Config
	StreamEventCli streamevent.StreamEventClient
	AsduPusher     *AsduPusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	streamEventCli := streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
		zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
		// 添加最大消息配置
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	).Conn())
	return &ServiceContext{
		Config:         c,
		StreamEventCli: streamEventCli,
		AsduPusher:     NewAsduPusher(streamEventCli, c.PushAsduChunkBytes),
	}
}

type AsduPusher struct {
	inserter       *executors.ChunkExecutor
	streamEventCli streamevent.StreamEventClient
	writerLock     sync.Mutex
}

func (w *AsduPusher) Write(val string) error {
	w.writerLock.Lock()
	defer w.writerLock.Unlock()
	return w.inserter.Add(val, len(val))
}

func (w *AsduPusher) execute(vals []interface{}) {
	startTime := timex.Now()
	msgBodyList := make([]*streamevent.MsgBody, 0)
	for _, val := range vals {
		s := val.(string)
		result := gjson.Parse(s)
		bodyRaw := result.Get("body").Raw
		typeId := result.Get("typeId").Int()
		msgBody := &streamevent.MsgBody{
			Host:        result.Get("host").String(),
			Port:        int32(result.Get("port").Int()),
			Asdu:        result.Get("asdu").String(),
			TypeId:      int32(typeId),
			DataType:    int32(result.Get("dataType").Int()),
			Coa:         uint32(result.Get("coa").Int()),
			BodyRaw:     bodyRaw,
			Time:        result.Get("time").String(),
			MetaDataRaw: result.Get("metaData").Raw,
		}
		msgBodyList = append(msgBodyList, msgBody)
	}
	if len(msgBodyList) == 0 {
		return
	}
	_, err := w.streamEventCli.PushChunkAsdu(context.Background(), &streamevent.PushChunkAsduReq{
		MsgBody: msgBodyList,
	})
	var invokeflg = "success"
	if err != nil {
		invokeflg = "fail"
	}
	duration := timex.Since(startTime)
	logx.WithDuration(duration).Infof("PushChunkAsdu, asdu size: %d - %s", len(msgBodyList), invokeflg)
}

func NewAsduPusher(streamEventCli streamevent.StreamEventClient, ChunkBytes int) *AsduPusher {
	writer := AsduPusher{}
	writer.inserter = executors.NewChunkExecutor(writer.execute, executors.WithChunkBytes(ChunkBytes))
	writer.streamEventCli = streamEventCli
	return &writer
}
