package svc

import (
	"context"
	"math"
	"time"
	"zero-service/app/iecstash/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/executorx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"

	"github.com/tidwall/gjson"
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
		zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
		// 添加最大消息配置
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	).Conn())

	chunkAsduPusher := executorx.NewChunkMessagesPusher(
		func(msgs []string) {
			// 转换为streamevent.MsgBody列表
			msgBodyList := make([]*streamevent.MsgBody, 0, len(msgs))
			for _, s := range msgs {
				result := gjson.Parse(s)
				bodyRaw := result.Get("body").Raw
				typeId := result.Get("typeId").Int()
				msgBody := &streamevent.MsgBody{
					MsgId:       result.Get("msgId").String(),
					Host:        result.Get("host").String(),
					Port:        int32(result.Get("port").Int()),
					Asdu:        result.Get("asdu").String(),
					TypeId:      int32(typeId),
					DataType:    int32(result.Get("dataType").Int()),
					Coa:         uint32(result.Get("coa").Int()),
					BodyRaw:     bodyRaw,
					Time:        result.Get("time").String(),
					MetaDataRaw: result.Get("metaData").String(),
				}
				pm := result.Get("pm")
				if pm.Exists() {
					msgBody.Pm = &streamevent.PointMapping{
						DeviceId:    pm.Get("deviceId").String(),
						DeviceName:  pm.Get("deviceName").String(),
						TdTableType: pm.Get("tdTableType").String(),
					}
				}
				msgBodyList = append(msgBodyList, msgBody)
			}

			// 调用gRPC推送
			tid, _ := tool.SimpleUUID()
			pushGrpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			startTime := timex.Now()
			_, err := streamEventCli.PushChunkAsdu(pushGrpcCtx, &streamevent.PushChunkAsduReq{
				MsgBody: msgBodyList,
				TId:     tid,
			})
			var invokeflg = "success"
			if err != nil {
				invokeflg = "fail"
				logx.WithContext(pushGrpcCtx).Errorf("PushChunkAsdu failed, tId: %s, err: %v", tid, err)
			}
			duration := timex.Since(startTime)
			logx.WithContext(pushGrpcCtx).WithDuration(duration).Infof("PushChunkAsdu, tId: %s, asdu size: %d - %s", tid, len(msgBodyList), invokeflg)
			return
		},
		c.PushAsduChunkBytes,
	)

	return &ServiceContext{
		Config:          c,
		StreamEventCli:  streamEventCli,
		ChunkAsduPusher: chunkAsduPusher,
	}
}
