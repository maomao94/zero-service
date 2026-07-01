package svc

import (
	"context"
	"fmt"
	"time"
	"zero-service/common/tool"

	"zero-service/app/djicloud/internal/config"
	"zero-service/app/djicloud/internal/hooks"
	"zero-service/app/djicloud/model/gormmodel"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/common/ossx"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

const dockOnlineTTL = 60 * time.Second

type ServiceContext struct {
	Config      config.Config
	DjiClient   *djisdk.Client
	DB          *gormx.DB
	OnlineCache *collection.Cache
	OssTemplate ossx.OssTemplate
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
			&gormmodel.DjiFlyRegion{},
			&gormmodel.DjiFlyRegionSyncStatus{},
		)
	}
	return db
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	db := initDB(c)

	onlineCache, err := collection.NewCache(dockOnlineTTL, collection.WithName("dock-online-cache"))
	logx.Must(err)

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

	var handlerOpts []djisdk.ClientOption
	if pushCli != nil {
		handlerOpts = append(handlerOpts,
			djisdk.WithDrcSessionEnabled(func(ctx context.Context, gatewaySn, sessionID string) {
				threading.GoSafe(func() {
					ctx := context.WithoutCancel(ctx)
					reqId, _ := tool.SimpleUUID()
					room := "drc:heartbeat:" + gatewaySn
					_, err := pushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
						ReqId:   reqId,
						Room:    room,
						Event:   "drc:session_enabled",
						Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s"}`, gatewaySn, sessionID),
					})
					if err != nil {
						logx.Errorw("[dji-sdk] drc_manager socket push session_enabled failed: "+err.Error(), logx.Field("gateway_sn", gatewaySn))
					}
				})
			}),
			djisdk.WithDrcSessionDisabled(func(ctx context.Context, gatewaySn, sessionID string) {
				threading.GoSafe(func() {
					ctx := context.WithoutCancel(ctx)
					reqId, _ := tool.SimpleUUID()
					room := "drc:heartbeat:" + gatewaySn
					_, err := pushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
						ReqId:   reqId,
						Room:    room,
						Event:   "drc:session_disabled",
						Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s"}`, gatewaySn, sessionID),
					})
					if err != nil {
						logx.Errorw("[dji-sdk] drc_manager socket push session_disabled failed: "+err.Error(), logx.Field("gateway_sn", gatewaySn))
					}
				})
			}),
			djisdk.WithDrcSessionExpired(func(ctx context.Context, gatewaySn, sessionID, reason string) {
				threading.GoSafe(func() {
					ctx := context.WithoutCancel(ctx)
					reqId, _ := tool.SimpleUUID()
					room := "drc:heartbeat:" + gatewaySn
					_, err := pushCli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
						ReqId:   reqId,
						Room:    room,
						Event:   "drc:session_expired",
						Payload: fmt.Sprintf(`{"gateway_sn":"%s","session_id":"%s","reason":"%s"}`, gatewaySn, sessionID, reason),
					})
					if err != nil {
						logx.Errorw("[dji-sdk] drc_manager socket push session_expired failed: "+err.Error(), logx.Field("gateway_sn", gatewaySn))
					}
				})
			}),
		)
	}

	// 初始化 OSS（可选）
	var ossTemplate ossx.OssTemplate
	if c.Oss != nil {
		ossTemplate = ossx.MustNewTemplate(&ossx.Config{
			Category:   c.Oss.Category,
			Endpoint:   c.Oss.Endpoint,
			AccessKey:  c.Oss.AccessKey,
			SecretKey:  c.Oss.SecretKey,
			BucketName: c.Oss.BucketName,
			Region:     c.Oss.Region,
		}, ossx.OssRule{})
	}

	handlerOpts = append(handlerOpts, hooks.WithDjiClientOptions(hooks.RegisterDjiClientOptions{
		DB:                 db,
		OnlineCache:        onlineCache,
		PushCli:            pushCli,
		DisableOsdSQLTrace: c.Telemetry.DisableOsdSQLTrace,
		OssTemplate:        ossTemplate,
	})...)

	djiCli := djisdk.MustNewClient(c.Dji, handlerOpts...)

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-cloud] subscribe topics failed: %v", err)
	}

	return &ServiceContext{
		Config:      c,
		DjiClient:   djiCli,
		DB:          db,
		OnlineCache: onlineCache,
		OssTemplate: ossTemplate,
	}
}

func (s *ServiceContext) Close() {
	s.DjiClient.Close()
}
