package svc

import (
	"context"
	"math"
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
			tid, _ := tool.SimpleUUID()
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
					TraceId:     result.Get("traceId").String(),
					Headers:     gjsonHeadersMap(result.Get("headers")),
				}
				pm := result.Get("pm")
				if pm.Exists() {
					msgBody.Pm = &streamevent.PointMapping{
						DeviceId:    pm.Get("deviceId").String(),
						DeviceName:  pm.Get("deviceName").String(),
						TdTableType: pm.Get("tdTableType").String(),
						Ext1:        pm.Get("ext1").String(),
						Ext2:        pm.Get("ext2").String(),
						Ext3:        pm.Get("ext3").String(),
						Ext4:        pm.Get("ext4").String(),
						Ext5:        pm.Get("ext5").String(),
					}
				}
				msgBodyList = append(msgBodyList, msgBody)
			}

			if len(msgBodyList) > 0 {
				startTime := timex.Now()
				_, err := streamEventCli.PushChunkAsdu(context.Background(), &streamevent.PushChunkAsduReq{
					MsgBody: msgBodyList,
					TId:     tid,
				})
				invokeflg := "success"
				if err != nil {
					invokeflg = "fail"
					logx.Errorf("PushChunkAsdu failed, tId: %s, err: %v", tid, err)
				}
				duration := timex.Since(startTime)
				logx.WithDuration(duration).Infof("PushChunkAsdu, tId: %s, asdu size: %d - %s", tid, len(msgBodyList), invokeflg)
			}
		},
		c.PushAsduChunkBytes,
	)

	return &ServiceContext{
		Config:          c,
		StreamEventCli:  streamEventCli,
		ChunkAsduPusher: chunkAsduPusher,
	}
}

func gjsonHeadersMap(r gjson.Result) map[string]string {
	if !r.Exists() {
		return nil
	}
	m := make(map[string]string)
	for k, v := range r.Map() {
		m[k] = v.String()
	}
	return m
}
