package svc

import (
	"context"
	"math"
	"zero-service/app/iecstash/internal/config"
	"zero-service/common/executorx"
	"zero-service/common/grpcx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/msgbody"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config          config.Config
	StreamEventCli  streamevent.StreamEventClient
	ChunkAsduPusher *executorx.ChunkMessagesPusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	streamEventCli := streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor),
		// 添加最大消息配置
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	).Conn())

	chunkAsduPusher := executorx.NewChunkMessagesPusher(
		func(msgs []string) {
			tid, _ := tool.SimpleUUID()
			msgBodyList := msgbody.FromJSONBatch(msgs)
			if len(msgBodyList) == 0 {
				return
			}

			ctx := context.Background()
			logger := logx.WithContext(ctx).WithFields(
				logx.Field("component", "iecstash.pusher"),
				logx.Field("tId", tid),
				logx.Field("asduCount", len(msgBodyList)),
			)
			startTime := timex.Now()
			_, err := streamEventCli.PushChunkAsdu(ctx, &streamevent.PushChunkAsduReq{
				MsgBody: msgBodyList,
				TId:     tid,
			})
			duration := timex.Since(startTime)
			if err != nil {
				logger.WithDuration(duration).Errorf("PushChunkAsdu failed: %v", err)
				return
			}
			logger.WithDuration(duration).Infof("PushChunkAsdu success")
		},
		c.PushAsduChunkBytes,
	)

	return &ServiceContext{
		Config:          c,
		StreamEventCli:  streamEventCli,
		ChunkAsduPusher: chunkAsduPusher,
	}
}
