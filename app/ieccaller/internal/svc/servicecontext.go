package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"
	"zero-service/app/ieccaller/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/dbx"
	"zero-service/common/executorx"
	"zero-service/common/iec104/client"
	"zero-service/common/iec104/types"
	"zero-service/common/iec104/util"
	"zero-service/common/mqttx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/tidwall/gjson"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config               config.Config
	ClientManager        *client.ClientManager
	KafkaASDUPusher      *kq.Pusher
	KafkaBroadcastPusher *kq.Pusher
	MqttClient           *mqttx.Client
	StreamEventCli       streamevent.StreamEventClient
	ChunkAsduPusher      *executorx.ChunkMessagesPusher

	DevicePointMappingModel model.DevicePointMappingModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config:        c,
		ClientManager: client.NewClientManager(),
	}
	if svcCtx.IsBroadcast() && len(c.KafkaConfig.Brokers) == 0 {
		logx.Must(fmt.Errorf("broadcast is enabled, but kafka config is empty"))
	}
	if len(c.KafkaConfig.Brokers) > 0 {
		svcCtx.KafkaASDUPusher = kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.Topic)
		svcCtx.KafkaBroadcastPusher = kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.BroadcastTopic)
	}
	if len(c.MqttConfig.Broker) > 0 {
		svcCtx.MqttClient = mqttx.MustNewClient(c.MqttConfig.MqttConfig)
	}
	if len(c.StreamEventConf.Endpoints) > 0 || len(c.StreamEventConf.Target) > 0 {
		streamEventCli := streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			// 添加最大消息配置
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
				//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
				//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
			)),
		).Conn())
		svcCtx.StreamEventCli = streamEventCli

		svcCtx.ChunkAsduPusher = executorx.NewChunkMessagesPusher(
			func(messages []string) {
				tid, _ := tool.SimpleUUID()
				msgBodyList := make([]*streamevent.MsgBody, 0, len(messages))
				for _, s := range messages {
					result := gjson.Parse(s)
					bodyRaw := result.Get("body").Raw
					typeId := result.Get("typeId").Int()
					item := &streamevent.MsgBody{
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
						item.Pm = &streamevent.PointMapping{
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
					msgBodyList = append(msgBodyList, item)
				}

				if len(msgBodyList) > 0 {
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
				}
				return
			},
			c.PushAsduChunkBytes,
		)
	}

	if len(c.DB.DataSource) > 0 {
		svcCtx.DevicePointMappingModel = model.NewDevicePointMappingModel(dbx.New(c.DB.DataSource))
	}
	return svcCtx
}

func (svc ServiceContext) PushASDU(ctx context.Context, data *types.MsgBody, ioa uint) error {
	key, _ := data.GetKey()
	data.Time = carbon.Now().ToDateTimeMicroString()

	// 获取 stationId，从上下文或生成
	stationId, ok := ctx.Value("stationId").(string)
	if !ok {
		stationId = util.GenerateStationId(data.Host, data.Port)
		logx.WithContext(ctx).Debugf("stationId not found in context, generated: %s, msgId: %s", stationId, data.MsgId)
	}
	if svc.DevicePointMappingModel != nil {
		query, exist, cacheErr := svc.DevicePointMappingModel.FindCacheOneByTagStationCoaIoa(ctx, stationId, int64(data.Coa), int64(ioa))
		if cacheErr != nil {
			logx.WithContext(ctx).Errorf("cache error %v, msgId: %s", cacheErr, data.MsgId)
			// 继续推送
		} else {
			if !exist {
				logx.WithContext(ctx).Debugf("no mapping for stationId: %s, coa: %d, ioa: %d, msgId: %s", stationId, data.Coa, ioa, data.MsgId)
				// 继续推送
			} else {
				if query.EnablePush != 1 {
					logx.WithContext(ctx).Debugf("push asdu disabled for stationId: %s, coa: %d, ioa: %d, msgId: %s", stationId, data.Coa, ioa, data.MsgId)
					return nil
				}
				data.Pm = &types.PointMapping{
					DeviceId:    query.DeviceId,
					DeviceName:  query.DeviceName,
					TdTableType: query.TdTableType,
					Ext1:        query.Ext1,
					Ext2:        query.Ext2,
					Ext3:        query.Ext3,
					Ext4:        query.Ext4,
					Ext5:        query.Ext5,
				}
			}
		}
	}
	byteData, err := jsonx.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error %v", err)
	}

	mr.FinishVoid(
		// Kafka 推送
		func() {
			if !svc.Config.KafkaConfig.IsPush {
				return
			}
			if svc.KafkaASDUPusher == nil {
				logx.WithContext(ctx).Errorf("kafka asdu pusher is nil, msgId: %s", data.MsgId)
				return
			}
			pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			kafkaErr := svc.KafkaASDUPusher.PushWithKey(pushCtx, key, string(byteData))
			if kafkaErr != nil {
				logx.WithContext(pushCtx).Errorf("failed to push asdu to kafka, msgId: %s, err: %v", data.MsgId, kafkaErr)
			}
		},
		// MQTT 推送
		func() {
			if !svc.Config.MqttConfig.IsPush {
				return
			}
			if svc.MqttClient == nil {
				logx.WithContext(ctx).Errorf("mqtt client is nil, msgId: %s", data.MsgId)
				return
			}

			topics := svc.Config.MqttConfig.Topic
			if len(topics) == 0 {
				return
			}

			for _, topicPattern := range topics {
				pushTopicCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				topic, genErr := util.GenerateTopic(topicPattern, data)
				if genErr != nil {
					logx.WithContext(pushTopicCtx).Debugf("failed to generate mqtt topic, pattern: %s, msgId: %s, err: %v", topicPattern, data.MsgId, genErr)
					continue
				}
				logx.WithContext(pushTopicCtx).Debugf("pushing asdu to mqtt topic: %s, msgId: %s", topic, data.MsgId)
				mqttErr := svc.MqttClient.Publish(pushTopicCtx, topic, byteData)
				if mqttErr != nil {
					logx.WithContext(pushTopicCtx).Errorf("failed to push asdu to mqtt topic: %s, msgId: %s, err: %v", topic, data.MsgId, mqttErr)
					continue
				}
			}
		},
		func() {
			if svc.ChunkAsduPusher != nil {
				if chunkErr := svc.ChunkAsduPusher.Write(string(byteData)); chunkErr != nil {
					logx.WithContext(ctx).Errorf("failed to write asdu to batch pusher, msgId: %s, err: %v", data.MsgId, chunkErr)
				}
				logx.WithContext(ctx).Debugf("write asdu to batch pusher, msgId: %s", data.MsgId)
			}
		},
	)
	return nil
}

func (svc ServiceContext) PushPbBroadcast(ctx context.Context, method string, in any) error {
	if svc.IsBroadcast() {
		pbData, err := json.Marshal(in)
		if err != nil {
			return err
		}
		data := &types.BroadcastBody{
			Method: method,
			Body:   string(pbData),
		}
		err = svc.PushBroadcast(ctx, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc ServiceContext) PushBroadcast(ctx context.Context, data *types.BroadcastBody) error {
	if !svc.IsBroadcast() {
		return nil
	}

	data.BroadcastGroupId = svc.Config.KafkaConfig.BroadcastGroupId
	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error %v", err)
	}

	// Kafka推送
	if svc.KafkaBroadcastPusher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := svc.KafkaBroadcastPusher.Push(ctx, string(byteData)); err != nil {
			logx.WithContext(ctx).Errorf("failed to push broadcast to kafka: %v", err)
			return err
		}
	}
	return nil
}

func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}

// Close 关闭所有资源
func (svc ServiceContext) Close() {
	if svc.KafkaASDUPusher != nil {
		logx.Infof("closing kafka asdu pusher")
		if err := svc.KafkaASDUPusher.Close(); err != nil {
			logx.Errorf("failed to close kafka asdu pusher: %v", err)
		}
	}
	if svc.KafkaBroadcastPusher != nil {
		logx.Infof("closing kafka broadcast pusher")
		if err := svc.KafkaBroadcastPusher.Close(); err != nil {
			logx.Errorf("failed to close kafka broadcast pusher: %v", err)
		}
	}
	if svc.MqttClient != nil {
		logx.Infof("closing mqtt client")
		svc.MqttClient.Close()
	}
	logx.Infof("service context closed")
}
