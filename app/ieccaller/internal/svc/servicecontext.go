package svc

import (
	"context"
	"fmt"
	"math"
	"time"
	"zero-service/app/ieccaller/internal/config"
	iecmqtt "zero-service/app/ieccaller/mqtt"
	"zero-service/common/carbonx"
	"zero-service/common/executorx"
	"zero-service/common/gormx"
	"zero-service/common/grpcx"
	"zero-service/common/iec104"
	"zero-service/common/iec104/client"
	"zero-service/common/iec104/types"
	"zero-service/common/iec104/util"
	"zero-service/common/mqttx"
	"zero-service/common/mqttx/broadcast"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model/gormmodel"

	"github.com/tidwall/gjson"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// errKindIECRejected IEC104 从站命令拒绝的错误类别（wire 值）。
// 业务错误类别由 app 注册（不在 mqttx/broadcast SDK 内置），此处保持迁移前 wire 语义不变。
const errKindIECRejected = "iec_rejected"

type ServiceContext struct {
	Config              config.Config
	ClientManager       *client.ClientManager
	KafkaASDUPusher     *kq.Pusher
	MqttClient          mqttx.Client
	StreamEventCli      streamevent.StreamEventClient
	ChunkAsduPusher     *executorx.ChunkMessagesPusher
	Broadcaster         broadcast.Broadcaster
	broadcastPrefix     string
	broadcastInstanceId string

	DB                      *gormx.DB
	DevicePointMappingStore *gormmodel.DevicePointMappingStore
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	svcCtx := &ServiceContext{
		Config:        c,
		ClientManager: client.NewClientManager(),
	}
	if svcCtx.IsBroadcast() && len(c.MqttConfig.Broker) == 0 {
		logx.Must(fmt.Errorf("broadcast is enabled, but mqtt config is empty"))
	}
	uid, err := tool.SimpleUUID()
	if err != nil {
		logx.Must(fmt.Errorf("generate instance id failed: %w", err))
	}
	svcCtx.broadcastInstanceId = "iec-caller-" + uid
	if len(c.KafkaConfig.Brokers) > 0 {
		svcCtx.KafkaASDUPusher = kq.NewPusher(c.KafkaConfig.Brokers, c.KafkaConfig.Topic)
	}
	if svcCtx.IsBroadcast() {
		svcCtx.broadcastPrefix = broadcast.Prefix("iec")
		ackReplyRouter := broadcast.NewAckReplyRouter(10*time.Second, "mqtt-ack-reply-"+uid)
		cfg := c.MqttConfig.MqttConfig
		cfg.ClientID = svcCtx.broadcastInstanceId
		cfg.Qos = 1
		svcCtx.MqttClient = mqttx.MustNewClient(cfg, mqttx.WithReplyRouter(
			broadcast.BroadcastAckTopic(svcCtx.broadcastPrefix, svcCtx.broadcastInstanceId), ackReplyRouter))
		svcCtx.Broadcaster = broadcast.NewBroadcaster(svcCtx.MqttClient, svcCtx.broadcastInstanceId,
			broadcast.WithPrefix(svcCtx.broadcastPrefix))
		// 注册 ieccaller 业务错误类别：IEC104 从站命令拒绝（输出/输入双向映射）
		broadcast.RegisterErrorKind(&client.CommandRejectedError{}, errKindIECRejected, func(msg string) error {
			return &client.CommandRejectedError{
				Cot:        extractCotFromError(msg),
				IsNegative: true,
				Status:     client.AckRejected,
			}
		})
		// 闭环：注册业务 executor（minimal 依赖不注入 ServiceContext）并挂载广播消费
		// （防回环 + method→executor 路由 + ack 回发由 broadcast SDK 骨架负责）
		iecmqtt.NewBroadcast(svcCtx.ClientManager, svcCtx.DevicePointMappingStore).RegisterExecutors(svcCtx.Broadcaster)
		if err := svcCtx.Broadcaster.AddBroadcastHandler(); err != nil {
			logx.Must(err)
		}
	} else if len(c.MqttConfig.Broker) > 0 {
		cfg := c.MqttConfig.MqttConfig
		cfg.ClientID = svcCtx.broadcastInstanceId
		svcCtx.MqttClient = mqttx.MustNewClient(cfg)
	}
	if len(c.StreamEventConf.Endpoints) > 0 || len(c.StreamEventConf.Target) > 0 {
		streamEventCli := streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor),
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
						TraceId:     result.Get("traceId").String(),
						Headers:     gjsonHeadersMap(result.Get("headers")),
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
					ctx, span := iec104.StartForwardSpan(context.Background())
					defer span.End()
					startTime := timex.Now()
					_, err := streamEventCli.PushChunkAsdu(ctx, &streamevent.PushChunkAsduReq{
						MsgBody: msgBodyList,
						TId:     tid,
					})
					invokeflg := "success"
					if err != nil {
						invokeflg = "fail"
						logx.WithContext(ctx).Errorf("PushChunkAsdu failed, tId: %s, err: %v", tid, err)
					}
					duration := timex.Since(startTime)
					logx.WithContext(ctx).WithDuration(duration).Infof("PushChunkAsdu, tId: %s, asdu size: %d - %s", tid, len(msgBodyList), invokeflg)
				}
			},
			c.PushAsduChunkBytes,
		)
	}
	if len(c.DB.DataSource) > 0 {
		db := gormx.MustOpenWithConf(c.DB)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			db.MustAutoMigrate(&gormmodel.GormDevicePointMapping{})
		}
		svcCtx.DB = db
		svcCtx.DevicePointMappingStore = gormmodel.NewDevicePointMappingStore(db)
	}
	return svcCtx
}

func (svc ServiceContext) PushASDU(ctx context.Context, data *types.MsgBody, ioa uint) error {
	key, _ := data.GetKey()
	data.Time = carbonx.NowDateTimeMicro()

	// 获取 stationId，从上下文或生成
	stationId, ok := ctx.Value("stationId").(string)
	if !ok {
		stationId = util.GenerateStationId(data.Host, data.Port)
		logx.WithContext(ctx).Debugf("stationId not found in context, generated: %s, msgId: %s", stationId, data.MsgId)
	}
	if svc.DevicePointMappingStore != nil {
		query, exist, cacheErr := svc.DevicePointMappingStore.FindCacheOneByTagStationCoaIoa(ctx, stationId, int64(data.Coa), int64(ioa))
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
				data.Pm = query.ToPointMapping()
			}
		}
	}
	data.Headers, data.TraceId = iec104.TraceHeaders(ctx)
	byteData, err := jsonx.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	mr.FinishVoid(
		// Kafka 推送
		func() {
			if !svc.Config.KafkaConfig.IsPush {
				return
			}
			kafkaCtx := logx.WithFields(ctx, logx.Field("channel", "kafka"))
			if svc.KafkaASDUPusher == nil {
				logx.WithContext(kafkaCtx).Error("kafka asdu pusher is nil")
				return
			}
			pushKafkaCtx, cancel := context.WithTimeout(kafkaCtx, 10*time.Second)
			defer cancel()
			kafkaErr := svc.KafkaASDUPusher.PushWithKey(pushKafkaCtx, key, string(byteData))
			if kafkaErr != nil {
				logx.WithContext(pushKafkaCtx).Errorf("failed to push asdu to kafka: %v", kafkaErr)
			}
		},
		// MQTT 推送
		func() {
			if !svc.Config.MqttConfig.IsPush {
				return
			}
			mqttCtx := logx.WithFields(ctx, logx.Field("channel", "mqtt"))
			if svc.MqttClient == nil {
				logx.WithContext(mqttCtx).Error("mqtt client is nil")
				return
			}

			topics := svc.Config.MqttConfig.Topic
			if len(topics) == 0 {
				return
			}

			for _, topicPattern := range topics {
				pushMqttCtx, cancel := context.WithTimeout(mqttCtx, 10*time.Second)
				topic, genErr := util.GenerateTopic(topicPattern, data)
				if genErr != nil {
					logx.WithContext(pushMqttCtx).Debugf("failed to generate mqtt topic, pattern: %s, err: %v", topicPattern, genErr)
					cancel()
					continue
				}
				logx.WithContext(pushMqttCtx).Debugf("pushing asdu to mqtt topic: %s", topic)
				mqttErr := svc.MqttClient.Publish(pushMqttCtx, topic, byteData)
				cancel()
				if mqttErr != nil {
					logx.WithContext(pushMqttCtx).Errorf("failed to push asdu to mqtt topic: %s, err: %v", topic, mqttErr)
					continue
				}
			}
		},
		func() {
			if svc.ChunkAsduPusher != nil {
				grpcCtx := logx.WithFields(ctx, logx.Field("channel", "grpc"))
				if chunkErr := svc.ChunkAsduPusher.Write(string(byteData)); chunkErr != nil {
					logx.WithContext(grpcCtx).Errorf("failed to write asdu to batch pusher: %v", chunkErr)
				}
				logx.WithContext(grpcCtx).Debug("write asdu to batch pusher")
			}
		},
	)
	return nil
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

// PushPbBroadcast 以 fire-and-forget 方式向集群广播 protobuf 命令（不等待 ack）。
func (svc ServiceContext) PushPbBroadcast(ctx context.Context, method string, in any) error {
	if !svc.IsBroadcast() {
		return nil
	}
	if svc.Broadcaster == nil {
		return fmt.Errorf("mqtt client is nil")
	}
	pbData, err := protojson.Marshal(in.(proto.Message))
	if err != nil {
		return err
	}
	return svc.Broadcaster.Broadcast(ctx, method, pbData)
}

// PushPbBroadcastWithAck 向集群广播 protobuf 命令并等待执行节点 ack，
// 成功时将 ack.ResponseBody 反序列化到 res，失败时按 errorKind 还原为领域错误。
func (svc ServiceContext) PushPbBroadcastWithAck(ctx context.Context, method string, in any, res any) error {
	if !svc.IsBroadcast() {
		return fmt.Errorf("not in cluster mode")
	}
	if svc.Broadcaster == nil {
		return fmt.Errorf("mqtt client is nil")
	}
	pbData, err := protojson.Marshal(in.(proto.Message))
	if err != nil {
		return err
	}
	respBody, err := svc.Broadcaster.BroadcastReply(ctx, method, pbData, 0)
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(respBody, res.(proto.Message)); err != nil {
		return fmt.Errorf("unmarshal response error: %w", err)
	}
	return nil
}

func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}

func (svc ServiceContext) BroadcastInstanceId() string {
	return svc.broadcastInstanceId
}

func (svc ServiceContext) BroadcastTopic() string {
	return broadcast.BroadcastTopic(svc.broadcastPrefix)
}

func (svc ServiceContext) BroadcastAckTopic() string {
	return broadcast.BroadcastAckTopic(svc.broadcastPrefix, svc.broadcastInstanceId)
}

// Close 关闭所有资源
func (svc ServiceContext) Close() {
	if svc.KafkaASDUPusher != nil {
		logx.Infof("closing kafka asdu pusher")
		if err := svc.KafkaASDUPusher.Close(); err != nil {
			logx.Errorf("failed to close kafka asdu pusher: %v", err)
		}
	}
	if svc.MqttClient != nil {
		logx.Infof("closing mqtt client")
		svc.MqttClient.Close()
	}
	logx.Infof("service context closed")
}

func extractCotFromError(errMsg string) string {
	// Extract COT from error message like "command rejected: cot=UnknownTypeID isNegative=true"
	if idx := indexOf(errMsg, "cot="); idx >= 0 {
		rest := errMsg[idx+4:]
		if endIdx := indexOf(rest, " "); endIdx >= 0 {
			return rest[:endIdx]
		}
		return rest
	}
	return "Unknown"
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
