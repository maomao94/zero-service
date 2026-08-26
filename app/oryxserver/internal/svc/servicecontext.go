package svc

import (
	"context"
	"fmt"
	"time"

	"zero-service/app/oryxserver/internal/config"
	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/mqtt"
	"zero-service/common/asynqx"
	"zero-service/common/ffmpegx"
	"zero-service/common/gormx"
	"zero-service/common/mqttx"
	"zero-service/common/mqttx/broadcast"
	"zero-service/common/oryxx"
	"zero-service/common/tool"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config     config.Config
	OryxClient *oryxx.Client
	// DB record 生命周期落库
	DB            *gormx.DB
	FFmpegManager *ffmpegx.Manager
	RelayRegistry *relay.RelayRegistry
	// MqttClient 可选：cluster 模式跨节点停止转推；standalone 为 nil
	MqttClient mqttx.Client
	// Broadcaster 集群广播客户端（cluster 模式）
	Broadcaster broadcast.Broadcaster

	// broadcast 相关（cluster 模式）
	relayPrefix string

	// 分布式 relay 基础设施（配置 Redis 即启用）
	NodeID         string
	RelayRedis     *redis.Redis
	AsynqServer    *asynq.Server
	AsynqClient    *asynq.Client
	AsynqInspector *asynq.Inspector
	StateStore     *relay.Store
	DistRelay      *relay.DistributedRelay
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	svcCtx := &ServiceContext{
		Config:        c,
		FFmpegManager: ffmpegx.NewManager(),
	}
	svcCtx.RelayRegistry = relay.NewRelayRegistry(svcCtx.FFmpegManager)
	// 节点标识：自动生成（broadcast / relay / nacos 共用）
	uid, err := tool.SimpleUUID()
	if err != nil {
		logx.Must(fmt.Errorf("generate node id failed: %w", err))
	}
	nodeID := "oryx-node-" + uid
	svcCtx.NodeID = nodeID

	// MQTT 初始化条件：cluster 模式必需，standalone 不依赖
	if svcCtx.IsBroadcast() && len(c.MqttConfig.Broker) == 0 {
		logx.Must(fmt.Errorf("relay broadcast is enabled (deployMode=cluster), but mqtt config is empty"))
	}
	if svcCtx.IsBroadcast() {
		svcCtx.relayPrefix = broadcast.Prefix("oryx", "server")
		ackReplyRouter := broadcast.NewAckReplyRouter(10*time.Second, "mqtt-ack-reply-"+nodeID)
		cfg := c.MqttConfig.MqttConfig
		cfg.ClientID = nodeID
		cfg.Qos = 1
		svcCtx.MqttClient = mqttx.MustNewClient(cfg, mqttx.WithReplyRouter(
			broadcast.BroadcastAckTopic(svcCtx.relayPrefix, nodeID), ackReplyRouter))
		svcCtx.Broadcaster = broadcast.NewBroadcaster(svcCtx.MqttClient, nodeID,
			broadcast.WithPrefix(svcCtx.relayPrefix))
		mqtt.NewBroadcast(svcCtx.RelayRegistry).RegisterExecutors(svcCtx.Broadcaster)
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

	// Redis（复用 zrpc.RpcServerConf.Redis，与 trigger 同一配置格式）
	svcCtx.RelayRedis = redis.MustNewRedis(c.Redis.RedisConf)
	// Asynq server / client / inspector（server 只消费 relay 隔离队列）
	svcCtx.AsynqServer = asynqx.NewAsynqServerWithQueue(c.Redis.Host, c.Redis.Pass, c.RedisDB, relay.RelayQueue, c.Concurrency)
	svcCtx.AsynqClient = asynqx.NewAsynqClient(c.Redis.Host, c.Redis.Pass, c.RedisDB)
	svcCtx.AsynqInspector = asynqx.NewAsynqInspector(c.Redis.Host, c.Redis.Pass, c.RedisDB)
	// 状态存储 + 分布式协调器
	svcCtx.StateStore = relay.NewStore(svcCtx.RelayRedis)
	svcCtx.DistRelay = relay.NewDistributedRelay(svcCtx.StateStore, svcCtx.RelayRegistry, nodeID, svcCtx.AsynqClient)
	// 进程级回调：progress 帧 → 续租；异常退出 → 入队补拉
	svcCtx.RelayRegistry.SetProgressHandler(func(ctx context.Context, target string) {
		svcCtx.DistRelay.OnProgress(ctx, target)
	})
	svcCtx.RelayRegistry.SetExitHandler(func(ctx context.Context, target string, result ffmpegx.ExitResult) {
		svcCtx.DistRelay.OnProcessExit(ctx, target, result)
	})
	logx.Infof("relay deps initialized: node_id=%s redis_db=%d queue=%s", nodeID, c.RedisDB, relay.RelayQueue)

	return svcCtx
}

// IsBroadcast 是否为集群部署模式（对齐 ieccaller）
func (svc ServiceContext) IsBroadcast() bool {
	return svc.Config.DeployMode == "cluster"
}

// BroadcastTopic 集群广播主题（nacos 元数据注册用）。
func (svc ServiceContext) BroadcastTopic() string {
	return broadcast.BroadcastTopic(svc.relayPrefix)
}

// BroadcastAckTopic 本实例 ack 主题（nacos 元数据注册用）。
func (svc ServiceContext) BroadcastAckTopic() string {
	return broadcast.BroadcastAckTopic(svc.relayPrefix, svc.NodeID)
}
