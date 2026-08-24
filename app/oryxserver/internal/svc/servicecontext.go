package svc

import (
	"fmt"
	"time"

	"zero-service/app/oryxserver/internal/config"
	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/mqtt"
	"zero-service/common/gormx"
	"zero-service/common/mqttx"
	"zero-service/common/mqttx/broadcast"
	"zero-service/common/oryxx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

type ServiceContext struct {
	Config     config.Config
	OryxClient *oryxx.Client
	// DB record 生命周期落库
	DB *gormx.DB
	// RelayManager 转推任务管理（sync.Map + FFmpeg 进程）
	RelayManager *relay.Manager
	// MqttClient 可选：cluster 模式跨节点停止转推；standalone 为 nil
	MqttClient mqttx.Client
	// Broadcaster 集群广播客户端（cluster 模式）
	Broadcaster broadcast.Broadcaster

	// broadcast 相关（cluster 模式）
	relayPrefix     string
	relayInstanceId string
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config:       c,
		RelayManager: relay.NewManager(),
	}
	// MQTT 初始化条件：cluster 模式必需，standalone 不依赖
	if svcCtx.IsBroadcast() && len(c.MqttConfig.Broker) == 0 {
		logx.Must(fmt.Errorf("relay broadcast is enabled (deployMode=cluster), but mqtt config is empty"))
	}
	uid, err := tool.SimpleUUID()
	if err != nil {
		logx.Must(fmt.Errorf("generate instance id failed: %w", err))
	}
	svcCtx.relayInstanceId = "oryx-relay-" + uid
	if svcCtx.IsBroadcast() {
		svcCtx.relayPrefix = broadcast.Prefix("oryx", "server")
		ackReplyRouter := broadcast.NewAckReplyRouter(10*time.Second, "mqtt-ack-reply-"+uid)
		cfg := c.MqttConfig.MqttConfig
		cfg.ClientID = svcCtx.relayInstanceId
		cfg.Qos = 1
		svcCtx.MqttClient = mqttx.MustNewClient(cfg, mqttx.WithReplyRouter(
			broadcast.BroadcastAckTopic(svcCtx.relayPrefix, svcCtx.relayInstanceId), ackReplyRouter))
		svcCtx.Broadcaster = broadcast.NewBroadcaster(svcCtx.MqttClient, svcCtx.relayInstanceId,
			broadcast.WithPrefix(svcCtx.relayPrefix))
		// 闭环：注册业务 executor（relay 停止，minimal 依赖不注入 ServiceContext）并挂载广播消费
		// （防回环 + method→executor 路由 + ack 回发由 broadcast SDK 骨架负责）
		mqtt.NewBroadcast(svcCtx.RelayManager).RegisterExecutors(svcCtx.Broadcaster)
		if err := svcCtx.Broadcaster.AddBroadcastHandler(); err != nil {
			logx.Must(err)
		}
	}

	// 数据库连接（record 生命周期落库）
	db := gormx.MustOpenWithConf(c.DB)
	if c.Mode == service.DevMode || c.Mode == service.TestMode {
		db.MustAutoMigrate(&gormmodel.Record{})
	}
	svcCtx.DB = db
	svcCtx.OryxClient = oryxx.NewClient(oryxx.Config{
		Ip:      c.OryxConfig.Ip,
		Port:    c.OryxConfig.Port,
		Timeout: c.OryxConfig.Timeout,
		Secret:  c.OryxConfig.Secret,
	})

	return svcCtx
}

// IsBroadcast 是否为集群部署模式（对齐 ieccaller）
func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}

// BroadcastInstanceId 本实例广播 ID（nacos 元数据注册用）。
func (svc ServiceContext) BroadcastInstanceId() string {
	return svc.relayInstanceId
}

// BroadcastTopic 集群广播主题（nacos 元数据注册用）。
func (svc ServiceContext) BroadcastTopic() string {
	return broadcast.BroadcastTopic(svc.relayPrefix)
}

// BroadcastAckTopic 本实例 ack 主题（nacos 元数据注册用）。
func (svc ServiceContext) BroadcastAckTopic() string {
	return broadcast.BroadcastAckTopic(svc.relayPrefix, svc.relayInstanceId)
}
