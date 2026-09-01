package logic

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"zero-service/app/live/internal/config"
	"zero-service/app/live/internal/svc"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/livekitx"
	"zero-service/common/tool"

	"github.com/alicebob/miniredis/v2"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/proto"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// liveKitMock 记录 Twirp 调用并可配置失败行为。
type liveKitMock struct {
	mu             sync.Mutex
	createdRooms   []string
	deletedRooms   []string
	removed        []string
	sendDataCalls  []string
	createErr      error
	deleteErr      error
	removeErr      error
	sendDataErr    error
	performRpcResp *livekit.PerformRpcResponse
	performRpcErr  error
}

// readProto 读取请求体并按 protobuf 解码。
func readProto(t *testing.T, r *http.Request, msg proto.Message) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.Unmarshal(body, msg); err != nil {
		t.Fatalf("decode %T: %v", msg, err)
	}
}

// writeProto 以 application/protobuf 返回 protobuf 响应。
func writeProto(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	body, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/protobuf")
	_, _ = w.Write(body)
}

// writeTwirpError 以 twirp JSON 错误格式返回错误响应。
func writeTwirpError(t *testing.T, w http.ResponseWriter, err twirp.Error) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(twirp.ServerHTTPStatusFromErrorCode(err.Code()))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code": string(err.Code()),
		"msg":  err.Msg(),
	})
}

// newLiveKitMockServer 构造 mock LiveKit Twirp 服务器。
func newLiveKitMockServer(t *testing.T, mock *liveKitMock) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			t.Error("missing Authorization header")
		} else if verifier, err := auth.ParseAPIToken(raw); err != nil {
			t.Errorf("invalid auth token: %v", err)
		} else if _, _, err := verifier.Verify("secret"); err != nil {
			t.Errorf("auth token verification failed: %v", err)
		}
		switch r.URL.Path {
		case "/twirp/livekit.RoomService/CreateRoom":
			var req livekit.CreateRoomRequest
			readProto(t, r, &req)
			if mock.createErr != nil {
				writeTwirpError(t, w, twirp.Internal.Error(mock.createErr.Error()))
				return
			}
			mock.mu.Lock()
			mock.createdRooms = append(mock.createdRooms, req.GetName())
			mock.mu.Unlock()
			writeProto(t, w, &livekit.Room{Name: req.GetName(), Sid: "RM_mock"})
		case "/twirp/livekit.RoomService/DeleteRoom":
			var req livekit.DeleteRoomRequest
			readProto(t, r, &req)
			if mock.deleteErr != nil {
				writeTwirpError(t, w, twirp.Internal.Error(mock.deleteErr.Error()))
				return
			}
			mock.mu.Lock()
			mock.deletedRooms = append(mock.deletedRooms, req.GetRoom())
			mock.mu.Unlock()
			writeProto(t, w, &livekit.DeleteRoomResponse{})
		case "/twirp/livekit.RoomService/RemoveParticipant":
			var req livekit.RoomParticipantIdentity
			readProto(t, r, &req)
			if mock.removeErr != nil {
				writeTwirpError(t, w, twirp.Internal.Error(mock.removeErr.Error()))
				return
			}
			mock.mu.Lock()
			mock.removed = append(mock.removed, req.GetRoom()+"/"+req.GetIdentity())
			mock.mu.Unlock()
			writeProto(t, w, &livekit.RemoveParticipantResponse{})
		case "/twirp/livekit.RoomService/SendData":
			var req livekit.SendDataRequest
			readProto(t, r, &req)
			if mock.sendDataErr != nil {
				writeTwirpError(t, w, twirp.Internal.Error(mock.sendDataErr.Error()))
				return
			}
			mock.mu.Lock()
			mock.sendDataCalls = append(mock.sendDataCalls, req.GetRoom()+"/"+req.GetTopic())
			mock.mu.Unlock()
			writeProto(t, w, &livekit.SendDataResponse{})
		case "/twirp/livekit.RoomService/PerformRpc":
			if mock.performRpcErr != nil {
				writeTwirpError(t, w, twirp.Internal.Error(mock.performRpcErr.Error()))
				return
			}
			if mock.performRpcResp != nil {
				writeProto(t, w, mock.performRpcResp)
				return
			}
			writeProto(t, w, &livekit.PerformRpcResponse{Payload: "pong"})
		case "/twirp/livekit.RoomService/ListParticipants":
			writeProto(t, w, &livekit.ListParticipantsResponse{})
		case "/twirp/livekit.RoomService/MutePublishedTrack":
			writeProto(t, w, &livekit.MuteRoomTrackResponse{})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// openTestDB 打开内存 sqlite 并迁移模型。
func openTestDB(t *testing.T) *gormx.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_loc=auto"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}
	if err := db.AutoMigrate(&gormmodel.LiveMeeting{}, &gormmodel.LiveMeetingParticipant{}); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	return &gormx.DB{DB: db}
}

// newTestSvcCtx 构造带 mock LiveKit 的 ServiceContext。
func newTestSvcCtx(t *testing.T, mock *liveKitMock) *svc.ServiceContext {
	t.Helper()
	server := newLiveKitMockServer(t, mock)
	lk, err := livekitx.New(
		livekitx.WithURL(server.URL),
		livekitx.WithAPIKey("devkey", "secret"),
	)
	if err != nil {
		t.Fatalf("create livekit client: %v", err)
	}
	t.Cleanup(func() { _ = lk.Close() })
	mr := miniredis.RunT(t)
	return newTestSvcCtxWithRedis(t, lk, redis.New(mr.Addr()))
}

// newTestSvcCtxWithRedis 构造注入指定 Redis client 的 ServiceContext。
func newTestSvcCtxWithRedis(t *testing.T, lk *livekitx.Client, r *redis.Redis) *svc.ServiceContext {
	t.Helper()
	db := openTestDB(t)
	return &svc.ServiceContext{
		Config: config.Config{
			LiveKit: struct {
				Url           string
				ApiKey        string
				ApiSecret     string
				WebhookKey    string
				TokenValidFor time.Duration `json:",default=2h"`
			}{Url: "http://127.0.0.1:7880", ApiKey: "devkey", ApiSecret: "secret", TokenValidFor: 2 * time.Hour},
		},
		LiveKit:     lk,
		DB:          db,
		Redis:       r,
		IdUtil:      tool.NewIdUtil(r),
		MeetingRepo: svc.NewMeetingRepo(db),
	}
}
