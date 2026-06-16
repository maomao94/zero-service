package svc

import (
	"context"
	"fmt"
	"time"
	"zero-service/common/tool"

	"zero-service/app/djicloud/internal/config"
	"zero-service/app/djicloud/internal/drc"
	"zero-service/app/djicloud/internal/hooks"
	"zero-service/app/djicloud/model/gormmodel"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

const dockOnlineTTL = 60 * time.Second

type ServiceContext struct {
	Config      config.Config
	DjiClient   *djisdk.Client
	DB          *gormx.DB
	OnlineCache *collection.Cache
	DrcManager  *drc.Manager
}

func initDB(c config.Config) *gormx.DB {
	if c.DB.DataSource == "" {
		logx.Must(fmt.Errorf("djicloud db datasource is required"))
	}
	db := gormx.MustOpenWithConf(c.DB)
	if c.Mode == service.DevMode || c.Mode == service.TestMode {
		db.MustAutoMigrate(
			&gormmodel.DjiDevice{},
			&gormmodel.DjiDeviceTopo{},
			&gormmodel.DjiDeviceOsdSnapshot{},
			&gormmodel.DjiDeviceStateSnapshot{},
			&gormmodel.DjiHmsAlert{},
			&gormmodel.DjiDockFlightTask{},
			&gormmodel.DjiDockDeviceFlightTaskState{},
			&gormmodel.DjiFlightTaskReady{},
			&gormmodel.DjiRemoteLogEvent{},
			&gormmodel.DjiReturnHomeEvent{},
		)
	}
	return db
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	djiCli := djisdk.MustNewClient(c.MqttConfig,
		djisdk.WithPendingTTL(c.PendingTTL),
		djisdk.WithReplyOptions(djisdk.ReplyOptions{
			EnableEventReply:   c.UpstreamReply.EnableEventsReply,
			EnableStatusReply:  c.UpstreamReply.EnableStatusReply,
			EnableRequestReply: c.UpstreamReply.EnableRequestsReply,
		}),
	)

	onlineCache, err := collection.NewCache(dockOnlineTTL, collection.WithName("dock-online-cache"))
	logx.Must(err)

	db := initDB(c)

	// 初始化 SocketPush 客户端（可选，未配置时不推送）
	var pushCli socketpush.SocketPushClient
	if len(c.SocketPushConf.Endpoints) > 0 || len(c.SocketPushConf.Target) > 0 {
		pushCli = socketpush.NewSocketPushClient(zrpc.MustNewClient(c.SocketPushConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(50*1024*1024),
			)),
		).Conn())
	}

	// 初始化 DRC 管理器
	var drcOpts []drc.ManagerOption
	if pushCli != nil {
		drcOpts = append(drcOpts, drc.WithOnSessionEnabled(func(gatewaySn, sessionID string) {
			reqId, _ := tool.SimpleUUID()
			room := "drc:heartbeat:" + gatewaySn
			_, err := pushCli.BroadcastRoom(context.Background(), &socketpush.BroadcastRoomReq{
				ReqId:   reqId,
				Room:    room,
				Event:   "drc:session_enabled",
				Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s"}`, gatewaySn, sessionID),
			})
			if err != nil {
				logx.Errorf("[drc-manager] socket push session_enabled failed: sn=%s err=%v", gatewaySn, err)
			}
		}))
		drcOpts = append(drcOpts, drc.WithOnSessionDisabled(func(gatewaySn, sessionID string) {
			reqId, _ := tool.SimpleUUID()
			room := "drc:heartbeat:" + gatewaySn
			_, err := pushCli.BroadcastRoom(context.Background(), &socketpush.BroadcastRoomReq{
				ReqId:   reqId,
				Room:    room,
				Event:   "drc:session_disabled",
				Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s"}`, gatewaySn, sessionID),
			})
			if err != nil {
				logx.Errorf("[drc-manager] socket push session_disabled failed: sn=%s err=%v", gatewaySn, err)
			}
		}))
		drcOpts = append(drcOpts, drc.WithOnSessionExpired(func(gatewaySn, sessionID, reason string) {
			reqId, _ := tool.SimpleUUID()
			room := "drc:heartbeat:" + gatewaySn
			_, err := pushCli.BroadcastRoom(context.Background(), &socketpush.BroadcastRoomReq{
				ReqId:   reqId,
				Room:    room,
				Event:   "drc:session_expired",
				Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s","reason":"%s"}`, gatewaySn, sessionID, reason),
			})
			if err != nil {
				logx.Errorf("[drc-manager] socket push session_expired failed: sn=%s err=%v", gatewaySn, err)
			}
		}))
	}
	drcMgr := drc.NewManager(djiCli, c.DrcConfig, drcOpts...)

	hooks.RegisterDjiClient(djiCli, hooks.RegisterDjiClientOptions{
		DB:                 db,
		OnlineCache:        onlineCache,
		DrcManager:         drcMgr,
		PushCli:            pushCli,
		DisableOsdSQLTrace: c.Telemetry.DisableOsdSQLTrace,
	})

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-cloud] subscribe topics failed: %v", err)
	}

	return &ServiceContext{
		Config:      c,
		DjiClient:   djiCli,
		DB:          db,
		OnlineCache: onlineCache,
		DrcManager:  drcMgr,
	}
}

// Close 释放 ServiceContext 持有的资源。
func (s *ServiceContext) Close() {
	if s.DrcManager != nil {
		s.DrcManager.Close()
	}
	s.DjiClient.Close()
}
