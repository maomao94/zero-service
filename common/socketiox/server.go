package socketiox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/doquangtan/socketio/v4"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	EventUp              = "__up__"
	EventJoinRoom        = "__join_room_up__"
	EventLeaveRoom       = "__leave_room_up__"
	EventRoomBroadcast   = "__room_broadcast_up__"
	EventGlobalBroadcast = "__global_broadcast_up__"

	EventStatDown = "__stat_down__"
	EventDown     = "__down__"

	CodeSuccess  = 200
	CodeParamErr = 400
	CodeBizErr   = 500
)

const (
	statInterval = time.Minute
)

type SocketUpReq struct {
	Payload string `json:"payload"`
	ReqId   string `json:"reqId"`
	Room    string `json:"room,omitempty"`
	Event   string `json:"event,omitempty"`
}

type SocketUpRoomReq struct {
	ReqId string `json:"reqId"`
	Room  string `json:"room"`
}

type SocketResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Payload any    `json:"payload,omitempty"`
	SeqId   int64  `json:"seqId,omitempty"`
	ReqId   string `json:"reqId,omitempty"`
}

type SocketDown struct {
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
	SeqId   int64  `json:"seqId,omitempty"`
	ReqId   string `json:"reqId,omitempty"`
}

type StatDown struct {
	SId      string            `json:"sId"`
	Rooms    []string          `json:"rooms"`
	Nps      string            `json:"nps"`
	MetaData map[string]string `json:"metadata,omitempty"`
}

func BuildResp(code int, msg string, payload string, reqId string) []byte {
	resp := SocketResp{
		Code:    code,
		Msg:     msg,
		Payload: payload,
		ReqId:   reqId,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

func BuildDown(event string, payload string, reqId string) []byte {
	down := SocketDown{
		Event:   event,
		Payload: payload,
		ReqId:   reqId,
	}
	bytes, _ := json.Marshal(down)
	return bytes
}

type Session struct {
	id       string
	socket   *socketio.Socket
	lock     sync.Mutex
	metadata map[string]string
}

func (s *Session) Close() error {
	return s.socket.Disconnect()
}

func (s *Session) checkSocketNil() bool {
	return s.socket == nil
}

func (s *Session) ID() string { return s.id }

func (s *Session) GetMetadata(key string) interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.metadata[key]
}

func (s *Session) SetMetadata(key string, val interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if strValue, ok := val.(string); ok {
		s.metadata[key] = strValue
		logx.Debugf("[socketio] set metadata key %q with value %v for conn %s", key, strValue, s.id)
	} else {
		logx.Debugf("[socketio] skipped non-string metadata key %q with value %v (type: %T) for conn %s", key, val, val, s.id)
	}
}

func (s *Session) EmitAny(event string, payload any) error {
	ok := s.checkSocketNil()
	if ok {
		return fmt.Errorf("socket is nil")
	}
	return s.socket.Emit(event, payload)
}

func (s *Session) EmitString(event string, payload string) error {
	ok := s.checkSocketNil()
	if ok {
		return fmt.Errorf("socket is nil")
	}
	return s.socket.Emit(event, payload)
}

func (s *Session) EmitDown(event string, payload string, reqId string) error {
	ok := s.checkSocketNil()
	if ok {
		return fmt.Errorf("socket is nil")
	}
	data := BuildDown(event, payload, reqId)
	if len(event) == 0 {
		return errors.New("event name is empty")
	}
	if event == EventDown {
		return errors.New("event name is not allowed")
	}
	return s.socket.Emit(event, string(data))
}

func (s *Session) EmitEventDown(data string) error {
	return s.EmitString(EventDown, data)
}

func (s *Session) ReplyEventDown(code int, msg string, payload string, reqId string) error {
	data := BuildResp(code, msg, payload, reqId)
	return s.EmitEventDown(string(data))
}

func (s *Session) JoinRoom(room string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	ok := s.checkSocketNil()
	if ok {
		return fmt.Errorf("socket is nil")
	}
	if len(room) == 0 {
		return fmt.Errorf("room cannot be empty")
	}
	for _, r := range s.socket.Rooms() {
		if r == room {
			return nil
		}
	}
	s.socket.Join(room)
	return nil
}

func (s *Session) LeaveRoom(room string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	ok := s.checkSocketNil()
	if ok {
		return fmt.Errorf("socket is nil")
	}
	s.socket.Leave(room)
	return nil
}

type EventHandler func(ctx context.Context, event string, payload *socketio.EventPayload) error

type EventHandlers map[string]EventHandler

type Option func(s *Server)

func WithEventHandlers(handlers EventHandlers) Option {
	return func(s *Server) { s.eventHandlers = handlers }
}

// WithContextKeys 配置从上下文提取的键列表
func WithContextKeys(keys []string) Option {
	return func(s *Server) { s.contextKeys = keys }
}

type Server struct {
	*socketio.Io
	eventHandlers EventHandlers
	sessions      map[string]*Session
	lock          sync.RWMutex
	statInterval  time.Duration
	stopChan      chan struct{}
	contextKeys   []string // 从上下文提取的键列表
}

func MustServer(opts ...Option) *Server {
	srv, err := NewServer(opts...)
	logx.Must(err)
	return srv
}

func NewServer(opts ...Option) (*Server, error) {
	io := socketio.New()
	s := &Server{
		Io:            io,
		eventHandlers: make(EventHandlers),
		sessions:      make(map[string]*Session),
		statInterval:  statInterval,
		stopChan:      make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.bindEvents()
	go s.statLoop()
	return s, nil
}

func (srv *Server) bindEvents() {
	srv.OnConnection(func(socket *socketio.Socket) {
		ctx := socket.Context
		session := &Session{
			id:       socket.Id,
			socket:   socket,
			metadata: make(map[string]string),
		}
		srv.lock.Lock()
		if ctx != nil && len(srv.contextKeys) > 0 {
			for _, key := range srv.contextKeys {
				if value := ctx.Value(key); value != nil {
					session.SetMetadata(key, value)
				}
			}
		}
		srv.sessions[socket.Id] = session
		srv.lock.Unlock()
		logx.Infof("[socketio] new connection established: conn=%s", socket.Id)
		socket.On(EventJoinRoom, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
				logx.Field("SID", payload.SID),
				logx.Field("EVENT", EventJoinRoom),
			)
			var handlerPayload []byte
			if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
				switch data := payload.Data[0].(type) {
				case string:
					handlerPayload = []byte(data)
				default:
					b, err := jsonx.Marshal(data)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", EventUp, socket.Id, err)
						if payload.Ack != nil {
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "")))
						} else {
							_ = session.ReplyEventDown(CodeParamErr, "数据格式错误", "", "")
						}
						return
					}
					handlerPayload = b
				}
			}
			var upReq SocketUpRoomReq
			if err := jsonx.Unmarshal(handlerPayload, &upReq); err != nil {
				logx.WithContext(ctx).Errorf("[socketio] failed to parse request: conn=%s, err=%v, raw_data=%s", socket.Id, err, string(handlerPayload))
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "参数解析失败", "", upReq.ReqId)
				}
				return
			}
			if len(upReq.ReqId) == 0 || len(upReq.Room) == 0 {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s", socket.Id)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|room为必填项", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "reqId|room为必填项", "", upReq.ReqId)
				}
				return
			}
			threading.GoSafe(func() {
				ack := payload.Ack
				session.JoinRoom(upReq.Room)
				if ack != nil {
					ack(string(BuildResp(CodeSuccess, "处理成功", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeSuccess, "处理成功", "", upReq.ReqId)
				}
			})
		})
		socket.On(EventLeaveRoom, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
				logx.Field("SID", payload.SID),
				logx.Field("EVENT", EventLeaveRoom),
			)
			var handlerPayload []byte
			if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
				switch data := payload.Data[0].(type) {
				case string:
					handlerPayload = []byte(data)
				default:
					b, err := jsonx.Marshal(data)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", EventUp, socket.Id, err)
						if payload.Ack != nil {
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "")))
						} else {
							_ = session.ReplyEventDown(CodeParamErr, "数据格式错误", "", "")
						}
						return
					}
					handlerPayload = b
				}
			}
			var upReq SocketUpRoomReq
			if err := jsonx.Unmarshal(handlerPayload, &upReq); err != nil {
				logx.WithContext(ctx).Errorf("[socketio] failed to parse request: conn=%s, err=%v, raw_data=%s", socket.Id, err, string(handlerPayload))
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "参数解析失败", "", upReq.ReqId)
				}
				return
			}
			if len(upReq.ReqId) == 0 || len(upReq.Room) == 0 {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s", socket.Id)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|room为必填项", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "reqId|room为必填项", "", upReq.ReqId)
				}
				return
			}
			threading.GoSafe(func() {
				ack := payload.Ack
				session.LeaveRoom(upReq.Room)
				if ack != nil {
					ack(string(BuildResp(CodeSuccess, "处理成功", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeSuccess, "处理成功", "", upReq.ReqId)
				}
			})
		})
		socket.On(EventUp, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
				logx.Field("SID", payload.SID),
				logx.Field("EVENT", EventUp),
			)
			var handlerPayload []byte
			if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
				switch data := payload.Data[0].(type) {
				case string:
					handlerPayload = []byte(data)
				default:
					b, err := jsonx.Marshal(data)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", EventUp, socket.Id, err)
						if payload.Ack != nil {
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "")))
						} else {
							_ = session.ReplyEventDown(CodeParamErr, "数据格式错误", "", "")
						}
						return
					}
					handlerPayload = b
				}
			}
			logx.WithContext(ctx).Debugf("[socketio] received event: %s from conn: %s, payload: %s", EventUp, socket.Id, string(handlerPayload))
			var upReq SocketUpReq
			if err := jsonx.Unmarshal(handlerPayload, &upReq); err != nil {
				logx.WithContext(ctx).Errorf("[socketio] failed to parse request: conn=%s, err=%v, raw_data=%s", socket.Id, err, string(handlerPayload))
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "参数解析失败", "", upReq.ReqId)
				}
				return
			}
			if len(upReq.ReqId) == 0 || len(upReq.Payload) == 0 {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s", socket.Id)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|payload为必填项", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "reqId|payload为必填项", "", upReq.ReqId)
				}
				return
			}
			logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s", socket.Id)
			threading.GoSafe(func() {
				ack := payload.Ack
				if upHandler := srv.eventHandlers[EventUp]; upHandler != nil {
					err := upHandler(ctx, EventUp, payload)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to process request: conn=%s, err=%v", socket.Id, err)
						if ack != nil {
							ack(string(BuildResp(CodeBizErr, "业务处理失败", "", upReq.ReqId)))
						} else {
							_ = session.ReplyEventDown(CodeBizErr, "业务处理失败", "", upReq.ReqId)
						}
					} else {
						if ack != nil {
							ack(string(BuildResp(CodeSuccess, "处理成功", "", upReq.ReqId)))
						} else {
							_ = session.ReplyEventDown(CodeSuccess, "处理成功", "", upReq.ReqId)
						}
					}
				} else {
					logx.WithContext(ctx).Debugf("[socketio] no handler registered for EventUp: conn=%s", socket.Id)
					if ack != nil {
						ack(string(BuildResp(CodeBizErr, "未配置处理器", "", upReq.ReqId)))
					} else {
						_ = session.ReplyEventDown(CodeBizErr, "未配置处理器", "", upReq.ReqId)
					}
				}
			})
		})
		socket.On(EventRoomBroadcast, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
				logx.Field("SID", payload.SID),
				logx.Field("EVENT", EventRoomBroadcast),
			)
			var handlerPayload []byte
			if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
				switch data := payload.Data[0].(type) {
				case string:
					handlerPayload = []byte(data)
				default:
					b, err := jsonx.Marshal(data)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", EventUp, socket.Id, err)
						if payload.Ack != nil {
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "")))
						} else {
							_ = session.ReplyEventDown(CodeParamErr, "数据格式错误", "", "")
						}
						return
					}
					handlerPayload = b
				}
			}
			logx.WithContext(ctx).Debugf("[socketio] received event: %s from conn: %s, payload: %s", EventUp, socket.Id, string(handlerPayload))
			var upReq SocketUpReq
			if err := jsonx.Unmarshal(handlerPayload, &upReq); err != nil {
				logx.WithContext(ctx).Errorf("[socketio] failed to parse request: conn=%s, err=%v, raw_data=%s", socket.Id, err, string(handlerPayload))
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "参数解析失败", "", upReq.ReqId)
				}
				return
			}
			if len(upReq.ReqId) == 0 || len(upReq.Payload) == 0 || len(upReq.Room) == 0 || len(upReq.Event) == 0 {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s", socket.Id)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|payload|room|event为必填项", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "reqId|payload|room|event为必填项", "", upReq.ReqId)
				}
				return
			}
			logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s")
			threading.GoSafe(func() {
				ack := payload.Ack
				err := srv.BroadcastRoom(upReq.Room, upReq.Event, upReq.Payload, upReq.ReqId)
				if err != nil {
					logx.WithContext(ctx).Errorf("[socketio] failed to process request: conn=%s, err=%v", socket.Id, err)
					if ack != nil {
						ack(string(BuildResp(CodeBizErr, "业务处理失败", "", upReq.ReqId)))
					} else {
						_ = session.ReplyEventDown(CodeBizErr, "业务处理失败", "", upReq.ReqId)
					}
				} else {
					if ack != nil {
						ack(string(BuildResp(CodeSuccess, "处理成功", "", upReq.ReqId)))
					} else {
						_ = session.ReplyEventDown(CodeSuccess, "处理成功", "", upReq.ReqId)
					}
				}
			})
		})
		socket.On(EventGlobalBroadcast, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
				logx.Field("SID", payload.SID),
				logx.Field("EVENT", EventGlobalBroadcast),
			)
			var handlerPayload []byte
			if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
				switch data := payload.Data[0].(type) {
				case string:
					handlerPayload = []byte(data)
				default:
					b, err := jsonx.Marshal(data)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", EventUp, socket.Id, err)
						if payload.Ack != nil {
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "")))
						} else {
							_ = session.ReplyEventDown(CodeParamErr, "数据格式错误", "", "")
						}
						return
					}
					handlerPayload = b
				}
			}
			logx.WithContext(ctx).Debugf("[socketio] received event: %s from conn: %s, payload: %s", EventUp, socket.Id, string(handlerPayload))
			var upReq SocketUpReq
			if err := jsonx.Unmarshal(handlerPayload, &upReq); err != nil {
				logx.WithContext(ctx).Errorf("[socketio] failed to parse request: conn=%s, err=%v, raw_data=%s", socket.Id, err, string(handlerPayload))
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "参数解析失败", "", upReq.ReqId)
				}
				return
			}
			if len(upReq.ReqId) == 0 || len(upReq.Payload) == 0 || len(upReq.Room) == 0 || len(upReq.Event) == 0 {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s", socket.Id)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|payload|event为必填项", "", upReq.ReqId)))
				} else {
					_ = session.ReplyEventDown(CodeParamErr, "reqId|payload|event为必填项", "", upReq.ReqId)
				}
				return
			}
			logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s", socket.Id)
			threading.GoSafe(func() {
				ack := payload.Ack
				err := srv.BroadcastGlobal(upReq.Event, upReq.Payload, upReq.ReqId)
				if err != nil {
					logx.WithContext(ctx).Errorf("[socketio] failed to process request: conn=%s, err=%v", socket.Id, err)
					if ack != nil {
						ack(string(BuildResp(CodeBizErr, "业务处理失败", "", upReq.ReqId)))
					} else {
						_ = session.ReplyEventDown(CodeBizErr, "业务处理失败", "", upReq.ReqId)
					}
				} else {
					if ack != nil {
						ack(string(BuildResp(CodeSuccess, "处理成功", "", upReq.ReqId)))
					} else {
						_ = session.ReplyEventDown(CodeSuccess, "处理成功", "", upReq.ReqId)
					}
				}
			})
		})
		for eventName, handler := range srv.eventHandlers {
			if eventName == EventDown || eventName == EventUp {
				continue
			}
			currentEvent := eventName
			currentHandler := handler
			socket.On(currentEvent, func(payload *socketio.EventPayload) {
				ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID),
					logx.Field("SID", payload.SID),
					logx.Field("EVENT", currentEvent),
				)
				var handlerPayload []byte
				if payload.Data != nil && len(payload.Data) > 0 && payload.Data[0] != nil {
					switch data := payload.Data[0].(type) {
					case string:
						handlerPayload = []byte(data)
					default:
						b, err := json.Marshal(data)
						if err != nil {
							logx.WithContext(ctx).Errorf("[socketio] failed to marshal data for event %s: conn=%s, err=%v", currentEvent, socket.Id, err)
							return
						}
						handlerPayload = b
					}
				}
				logx.WithContext(ctx).Debugf("[socketio] received event: %s from conn: %s, payload length: %d", currentEvent, socket.Id, len(handlerPayload))
				threading.GoSafe(func() {
					currentHandler(ctx, currentEvent, payload)
				})
			})
		}
		socket.On("disconnect", func(payload *socketio.EventPayload) {
			reason := "client disconnect"
			if payload.Data != nil && len(payload.Data) > 0 {
				if r, ok := payload.Data[0].(string); ok {
					reason = r
				}
			}
			logx.Infof("[socketio] disconnecting: conn=%s, reason=%s", socket.Id, reason)
			srv.cleanInvalidSession(socket.Id)
		})
	})
}

func (srv *Server) BroadcastRoom(room, event string, payload string, reqId string) error {
	data := BuildDown(event, payload, reqId)
	if len(event) == 0 {
		return errors.New("event name is empty")
	}
	if event == EventDown {
		return errors.New("event name is not allowed")
	}
	srv.Io.To(room).Emit(event, string(data))
	return nil
}

func (s *Server) BroadcastGlobal(event string, payload string, reqId string) error {
	data := BuildDown(event, payload, reqId)
	if len(event) == 0 {
		return errors.New("event name is empty")
	}
	if event == EventDown {
		return errors.New("event name is not allowed")
	}
	s.Io.Emit(event, string(data))
	return nil
}

func (srv *Server) statLoop() {
	if srv.statInterval <= 0 {
		return
	}
	ticker := time.NewTicker(srv.statInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			srv.lock.RLock()
			sessions := make([]*Session, 0, len(srv.sessions))
			for _, sess := range srv.sessions {
				sessions = append(sessions, sess)
			}
			srv.lock.RUnlock()
			sockets := srv.Io.Sockets()
			if len(sockets) != len(sessions) {
				logx.Errorf("[socketio] session count mismatch: sessions=%d, sockets=%d", len(sessions), len(sockets))
			}
			logx.Statf("[socketio] total sessions: %d", len(sessions))
			for _, sess := range sessions {
				threading.GoSafe(func() {
					stat := StatDown{
						SId:      sess.id,
						Rooms:    sess.socket.Rooms(),
						Nps:      sess.socket.Nps,
						MetaData: sess.metadata,
					}
					payload, _ := jsonx.Marshal(&stat)
					sess.EmitString(EventStatDown, string(payload))
				})
			}
		case <-srv.stopChan:
			return
		}
	}
}

func (srv *Server) cleanInvalidSession(sId string) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	logx.Debugf("[socketio] cleaning invalid session: %s", sId)
	delete(srv.sessions, sId)
}

func (srv *Server) SessionCount() int {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	return len(srv.sessions)
}

func (srv *Server) GetSession(sId string) *Session {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	return srv.sessions[sId]
}

func (srv *Server) GetSessionByDeviceId(deviceId string) ([]*Session, bool) {
	return srv.GetSessionByKey("deviceId", deviceId)
}

func (srv *Server) GetSessionByUserId(userId string) ([]*Session, bool) {
	return srv.GetSessionByKey("userId", userId)
}

func (srv *Server) GetSessionByKey(key, value string) ([]*Session, bool) {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	var sessions []*Session
	for _, sess := range srv.sessions {
		if sess.GetMetadata(key) == value {
			sessions = append(sessions, sess)
		}
	}
	if len(sessions) == 0 {
		return nil, false
	}
	return sessions, true
}

func (srv *Server) JoinRoom(sId string, room string) {
	srv.lock.RLock()
	session, ok := srv.sessions[sId]
	srv.lock.RUnlock()
	if ok {
		session.JoinRoom(room)
	}
}

func (srv *Server) LeaveRoom(sId string, room string) {
	srv.lock.RLock()
	session, ok := srv.sessions[sId]
	srv.lock.RUnlock()
	if ok {
		session.LeaveRoom(room)
	}
}

func (srv *Server) Stop() {
	close(srv.stopChan)
	srv.lock.Lock()
	defer srv.lock.Unlock()
	srv.sessions = make(map[string]*Session)
	srv.Close()
	logx.Info("[socketio] server stopped")
}

func (s *Server) randomUUID() string {
	return uuid.New().String()
}
