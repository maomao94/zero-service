package livekitx

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/server-sdk-go/v2/signalling"
	"github.com/twitchtv/twirp"
	"github.com/zeromicro/go-zero/rest/httpc"
)

// HTTPService 是 go-zero 的 HTTP 服务接口，用于复用其请求客户端和观测配置。
type HTTPService = httpc.Service

// API 聚合 LiveKit 原生 Protocol service 接口。它不复制请求和响应类型。
// v2.18.1 的 LiveKitAPI 无 HTTP client option，因此本包在同一认证基础上
// 直接构造 SDK 使用的生成 service client；HTTP 传输由 Config.HTTPClient
// 决定，未注入时使用 TLS 容错的内部传输（internalTransport）。
//
// MVP 只覆盖快速会议需要的 Room/Participant、Egress、Ingress、SIP 和
// AgentDispatch；Connector、AgentSimulation 和 Cloud Agents 不属于本包能力，
// 未来基于相同认证/HTTP 基础设施作为独立扩展包提供。
type API struct {
	roomService          livekit.RoomService
	egressService        livekit.Egress
	ingressService       livekit.Ingress
	sipService           livekit.SIP
	agentDispatchService livekit.AgentDispatchService
}

// Room 返回房间和参与者管理 service。
func (a *API) Room() livekit.RoomService { return a.roomService }

// Egress 返回录制和导出管理 service。
func (a *API) Egress() livekit.Egress { return a.egressService }

// Ingress 返回输入流管理 service。
func (a *API) Ingress() livekit.Ingress { return a.ingressService }

// SIP 返回 SIP 管理 service。
func (a *API) SIP() livekit.SIP { return a.sipService }

// AgentDispatch 返回 Agent dispatch service。
func (a *API) AgentDispatch() livekit.AgentDispatchService { return a.agentDispatchService }

func newAPI(cfg Config) (*API, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = internalTransport()
	}
	baseURL := signalling.ToHttpURL(cfg.URL)
	opts := []twirp.ClientOption{twirp.WithClientInterceptors(authInterceptor(cfg.APIKey, cfg.APISecret))}
	return &API{
		roomService:          livekit.NewRoomServiceProtobufClient(baseURL, client, opts...),
		egressService:        livekit.NewEgressProtobufClient(baseURL, client, opts...),
		ingressService:       livekit.NewIngressProtobufClient(baseURL, client, opts...),
		sipService:           livekit.NewSIPProtobufClient(baseURL, client, opts...),
		agentDispatchService: livekit.NewAgentDispatchServiceProtobufClient(baseURL, client, opts...),
	}, nil
}

// internalTransport 返回 SDK 内部默认传输：TLS 对自签证书容错，
// http/https 均可直连，调用方无需安装证书（开发/内网环境的主要路径）。
// 返回 *http.Client 时生成的 twirp client 会自动包一层禁重定向处理。
func internalTransport() livekit.HTTPClient {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // SDK 内部传输对内网自签证书容错
		},
	}
}

// serviceHTTPClient 把 go-zero httpc.Service 适配为生成的 twirp client
// 所需的 livekit.HTTPClient，使注入的 httpc.Service 真正执行管理请求。
type serviceHTTPClient struct{ service HTTPService }

func (c serviceHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.service.DoRequest(req)
}

// authInterceptor 为每个管理请求签发临时 Bearer token：
// grant 覆盖管理服务所需的管理权限，请求含房间时把 grant 限定到该房间；
// token 由管理 API 凭据派生，不落日志。
func authInterceptor(key, secret string) twirp.Interceptor {
	return func(next twirp.Method) twirp.Method {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			grant := &auth.VideoGrant{
				RoomCreate: true, RoomList: true, RoomRecord: true, RoomAdmin: true,
				IngressAdmin: true,
			}
			if room := requestRoom(req); room != "" {
				grant.Room = room
			}
			token, err := auth.NewAccessToken(key, secret).SetVideoGrant(grant).SetSIPGrant(&auth.SIPGrant{Admin: true, Call: true}).ToJWT()
			if err != nil {
				return nil, err
			}
			ctx, err = twirp.WithHTTPRequestHeaders(ctx, http.Header{"Authorization": []string{"Bearer " + token}})
			if err != nil {
				return nil, err
			}
			return next(ctx, req)
		}
	}
}

// requestRoom 从常见房间管理请求中提取房间名，用于把管理 token 的
// grant 限定到单个房间；请求类型不在清单内时返回空字符串（不限定）。
func requestRoom(req interface{}) string {
	switch request := req.(type) {
	case *livekit.ListParticipantsRequest:
		return request.Room
	case *livekit.RoomParticipantIdentity:
		return request.Room
	case *livekit.MuteRoomTrackRequest:
		return request.Room
	case *livekit.UpdateParticipantRequest:
		return request.Room
	case *livekit.UpdateSubscriptionsRequest:
		return request.Room
	case *livekit.SendDataRequest:
		return request.Room
	case *livekit.UpdateRoomMetadataRequest:
		return request.Room
	case *livekit.PerformRpcRequest:
		return request.Room
	}
	return ""
}
