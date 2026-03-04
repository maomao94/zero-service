package svc

import (
	"context"
	"math"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/socketiox"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/socketapp/socketgtw/internal/config"
	"zero-service/socketapp/socketgtw/internal/sockethandler"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config         config.Config
	SocketServer   *socketiox.Server
	StreamEventCli streamevent.StreamEventClient
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
	svcCtx := &ServiceContext{
		Config:         c,
		StreamEventCli: streamEventCli,
	}
	svcCtx.SocketServer = socketiox.MustServer(
		socketiox.WithContextKeys(c.SocketMetaData),
		socketiox.WithHandler(socketiox.EventUp, sockethandler.NewSocketUpHandler(svcCtx.StreamEventCli)),
		socketiox.WithTokenValidator(func(token string) bool {
			if c.JwtAuth.AccessSecret == "" {
				return true
			}
			if token == "" {
				return false
			}
			secrets := []string{c.JwtAuth.AccessSecret}
			if len(c.JwtAuth.PrevAccessSecret) > 0 {
				secrets = append(secrets, c.JwtAuth.PrevAccessSecret)
			}
			_, err := tool.ParseToken(token, secrets...)
			if err != nil {
				return false
			}
			return true
		}),
		socketiox.WithTokenValidatorWithClaims(func(token string) (map[string]interface{}, bool) {
			if c.JwtAuth.AccessSecret == "" {
				return nil, true
			}
			if token == "" {
				return nil, false
			}
			secrets := []string{c.JwtAuth.AccessSecret}
			if len(c.JwtAuth.PrevAccessSecret) > 0 {
				secrets = append(secrets, c.JwtAuth.PrevAccessSecret)
			}
			claims, err := tool.ParseToken(token, secrets...)
			if err != nil {
				return nil, false
			}
			return claims, true
		}),
		socketiox.WithConnectHook(func(ctx context.Context, session *socketiox.Session) ([]string, error) {
			reqId, _ := tool.SimpleUUID()
			payloadData := map[string]interface{}{
				"metadata": session.AllMetadata(),
			}
			downJson, _ := jsonx.Marshal(payloadData)
			res, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, &streamevent.UpSocketMessageReq{
				ReqId:   reqId,
				SId:     session.ID(),
				Event:   socketiox.EventConnection,
				Payload: string(downJson),
			})
			if err != nil {
				return nil, err
			}
			var rooms []string
			jsonx.Unmarshal([]byte(res.Payload), &rooms)
			for _, room := range rooms {
				session.JoinRoom(room)
			}
			return nil, nil
		}),
		socketiox.WithDisconnectHook(func(ctx context.Context, session *socketiox.Session, reason string) error {
			reqId, _ := tool.SimpleUUID()
			payloadData := map[string]interface{}{
				"metadata": session.AllMetadata(),
			}
			downJson, _ := jsonx.Marshal(payloadData)
			_, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, &streamevent.UpSocketMessageReq{
				ReqId:   reqId,
				SId:     session.ID(),
				Event:   socketiox.EventDisconnect,
				Payload: string(downJson),
			})
			if err != nil {
				return err
			}
			return nil
		}),
		socketiox.WithPreJoinRoomHook(func(ctx context.Context, session *socketiox.Session, reqId, room string) error {
			payloadData := map[string]interface{}{
				"metadata": session.AllMetadata(),
				"room":     room,
			}
			downJson, _ := jsonx.Marshal(payloadData)
			_, err := svcCtx.StreamEventCli.UpSocketMessage(ctx, &streamevent.UpSocketMessageReq{
				ReqId:   reqId,
				SId:     session.ID(),
				Event:   socketiox.EventJoinRoom,
				Payload: string(downJson),
			})
			if err != nil {
				return err
			}
			return nil
		}),
	)
	return svcCtx
}
