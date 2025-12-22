package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"text/template"
	"time"
	"zero-service/app/ieccaller/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/dbx"
	"zero-service/common/iec104/iec104client"
	"zero-service/common/iec104/types"
	"zero-service/common/iec104/util"
	"zero-service/common/mqttx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config               config.Config
	ClientManager        *iec104client.ClientManager
	KafkaASDUPusher      *kq.Pusher
	KafkaBroadcastPusher *kq.Pusher
	MqttClient           *mqttx.Client
	StreamEventCli       streamevent.StreamEventClient

	SqliteConn              sqlx.SqlConn
	DevicePointMappingModel model.DevicePointMappingModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	if c.DisableStmtLog {
		sqlx.DisableStmtLog()
	}
	svcCtx := &ServiceContext{
		Config:        c,
		ClientManager: iec104client.NewClientManager(),
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
	}

	if len(c.SqliteDB.DataSource) > 0 {
		svcCtx.SqliteConn = dbx.NewSqlite(c.SqliteDB.DataSource)
		svcCtx.DevicePointMappingModel = model.NewDevicePointMappingModel(svcCtx.SqliteConn)
	}
	return svcCtx
}

// generateTopic 根据配置的topic规则和报文值生成最终的topic
func generateTopic(topicPattern string, data *types.MsgBody) (string, error) {
	if data == nil {
		return "", errors.New("msg body is nil")
	}

	tmpl, err := template.New("topic").Parse(topicPattern)
	if err != nil {
		return "", errors.New("failed to parse topic template")
	}

	// Execute the template directly with the MsgBody struct
	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", errors.New("failed to execute topic template")
	}

	topic := result.String()

	if strings.Contains(topic, "{{") && strings.Contains(topic, "}}") {
		return "", errors.New("unresolved placeholders in topic")
	}

	if strings.Contains(topic, "+") || strings.Contains(topic, "#") {
		return "", errors.New("invalid topic pattern")
	}

	return topic, nil
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
	if svc.SqliteConn != nil {
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
	byteData, err := json.Marshal(data)
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
				topic, genErr := generateTopic(topicPattern, data)
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
		// gRPC 推送
		func() {
			if svc.StreamEventCli == nil {
				return
			}
			tid, _ := tool.SimpleUUID()
			bodyRaw, rpcErr := jsonx.MarshalToString(data.Body)
			if rpcErr != nil {
				logx.WithContext(ctx).Errorf("failed to marshal body to json, msgId: %s, err: %v", data.MsgId, rpcErr)
				return
			}
			metaDataRaw, rpcErr := jsonx.MarshalToString(data.MetaData)
			if rpcErr != nil {
				logx.WithContext(ctx).Errorf("failed to marshal metaData to json, msgId: %s, err: %v", data.MsgId, rpcErr)
				return
			}
			item := &streamevent.MsgBody{
				MsgId:       data.MsgId,
				Host:        data.Host,
				Port:        int32(data.Port),
				Asdu:        data.Asdu,
				TypeId:      int32(data.TypeId),
				DataType:    int32(data.DataType),
				Coa:         uint32(data.Coa),
				BodyRaw:     bodyRaw,
				Time:        data.Time,
				MetaDataRaw: metaDataRaw,
			}
			if data.Pm != nil {
				item.Pm = &streamevent.PointMapping{
					DeviceId:    data.Pm.DeviceId,
					DeviceName:  data.Pm.DeviceName,
					TdTableType: data.Pm.TdTableType,
					Ext1:        data.Pm.Ext1,
					Ext2:        data.Pm.Ext2,
					Ext3:        data.Pm.Ext3,
					Ext4:        data.Pm.Ext4,
					Ext5:        data.Pm.Ext5,
				}
			}
			msgBodyList := []*streamevent.MsgBody{
				item,
			}
			logx.WithContext(ctx).Debugf("pushing asdu to grpc, msgId: %s, tid: %s", data.MsgId, tid)
			pushGrpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			_, rpcErr = svc.StreamEventCli.PushChunkAsdu(pushGrpcCtx, &streamevent.PushChunkAsduReq{
				MsgBody: msgBodyList,
				TId:     tid,
			})
			if rpcErr != nil {
				logx.WithContext(pushGrpcCtx).Errorf("failed to push asdu to grpc, msgId: %s, tid: %s, err: %v", data.MsgId, tid, rpcErr)
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
