package socketio

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/doquangtan/socketio/v4"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	// 事件名称常量
	EventUp      = "__up__"       // 客户端上行事件
	EventDown    = "__down__"     // 服务器下行事件
	EventSeqSync = "__seq_down__" // 序列号同步事件

	// 状态码常量
	CodeSuccess  = 200
	CodeParamErr = 400
	CodeBizErr   = 500
)

const (
	SeqKeyUser   = "__user__"   // 单推专属KEY
	SeqKeyGlobal = "__global__" // 全局广播专属KEY
)

type SocketUpReq struct {
	Topic   string `json:"topic"`
	Method  string `json:"method"`
	Payload any    `json:"payload"`
	ReqId   string `json:"reqId"`
}

type SocketResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Topic   string `json:"topic,omitempty"`
	Method  string `json:"method,omitempty"`
	Payload any    `json:"payload,omitempty"`
	SeqId   int64  `json:"seqId,omitempty"`
	ReqId   string `json:"reqId,omitempty"`
	SId     string `json:"sId,omitempty"`
}

type SocketDown struct {
	Payload any    `json:"payload,omitempty"`
	SeqId   int64  `json:"seqId,omitempty"`
	ReqId   string `json:"reqId,omitempty"`
	SId     string `json:"sId,omitempty"`
}

func BuildResp(code int, msg string, topic, method string, payload any, seqId int64, reqId string, sId string) []byte {
	resp := SocketResp{
		Code:    code,
		Msg:     msg,
		Topic:   topic,
		Method:  method,
		Payload: payload,
		SeqId:   seqId,
		ReqId:   reqId,
		SId:     sId,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

func BuildDown(payload any, seqId int64, reqId string, sId string) []byte {
	down := SocketDown{
		Payload: payload,
		SeqId:   seqId,
		ReqId:   reqId,
		SId:     sId,
	}
	bytes, _ := json.Marshal(down)
	return bytes
}

type Session struct {
	id       string
	socket   *socketio.Socket
	seqNum   map[string]*int64
	lock     sync.Mutex
	metadata map[string]interface{}
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
	s.metadata[key] = val
}

func (s *Session) incrSeq(key string) int64 {
	if _, ok := s.seqNum[key]; !ok {
		s.seqNum[key] = new(int64)
	}
	return atomic.AddInt64(s.seqNum[key], 1)
}

func (s *Session) EmitString(event string, payload string) error {
	return s.socket.Emit(event, payload)
}

func (s *Session) EmitDown(data string) error {
	return s.EmitString(EventDown, data)
}

func (s *Session) ReplyDown(code int, msg string, topic, method string, payload any, reqId string) error {
	s.lock.Lock()
	seq := s.incrSeq(SeqKeyUser)
	s.lock.Unlock()
	data := BuildResp(code, msg, topic, method, payload, seq, reqId, s.id)
	return s.EmitDown(string(data))
}

func (s *Session) JoinRoom(roomID string) {
	s.socket.Join(roomID)
}

func (s *Session) LeaveRoom(roomID string) {
	s.socket.Leave(roomID)
	s.lock.Lock()
	delete(s.seqNum, roomID)
	s.lock.Unlock()
}

type EventHandler func(ctx context.Context, event string, payload *socketio.EventPayload) error

type EventHandlers map[string]EventHandler

type Option func(s *Server)

func WithEventHandlers(handlers EventHandlers) Option {
	return func(s *Server) { s.eventHandlers = handlers }
}

func WithSeqSyncInterval(interval time.Duration) Option {
	return func(s *Server) { s.seqSyncInterval = interval }
}

type Server struct {
	*socketio.Io
	eventHandlers   EventHandlers
	sessions        map[string]*Session
	lock            sync.RWMutex
	seqSyncInterval time.Duration
	stopChan        chan struct{}
}

func MustServer(opts ...Option) *Server {
	srv, err := NewServer(opts...)
	logx.Must(err)
	return srv
}

func NewServer(opts ...Option) (*Server, error) {
	io := socketio.New()
	s := &Server{
		Io:              io,
		eventHandlers:   make(EventHandlers),
		sessions:        make(map[string]*Session),
		seqSyncInterval: 0,
		stopChan:        make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.bindEvents()
	go s.StartSeqSync()
	return s, nil
}

func (s *Server) bindEvents() {
	s.OnConnection(func(socket *socketio.Socket) {
		session := &Session{
			id:       socket.Id,
			socket:   socket,
			seqNum:   make(map[string]*int64),
			metadata: make(map[string]interface{}),
		}
		s.lock.Lock()
		s.sessions[socket.Id] = session
		s.lock.Unlock()
		logx.Infof("[socketio] new connection established: conn=%s, total=%d", socket.Id, s.SessionCount())
		socket.On(EventUp, func(payload *socketio.EventPayload) {
			ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID), logx.Field("SID", payload.SID))
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
							payload.Ack(string(BuildResp(CodeParamErr, "数据格式错误", "", "", nil, 0, "", payload.SID)))
						} else {
							_ = session.ReplyDown(CodeParamErr, "数据格式错误", "", "", nil, "")
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
					payload.Ack(string(BuildResp(CodeParamErr, "参数解析失败", "", "", nil, 0, upReq.ReqId, payload.SID)))
				} else {
					_ = session.ReplyDown(CodeParamErr, "参数解析失败", "", "", nil, upReq.ReqId)
				}
				return
			}
			if upReq.ReqId == "" || upReq.Topic == "" || upReq.Method == "" || upReq.Payload == nil {
				logx.WithContext(ctx).Errorf("[socketio] missing required fields: conn=%s, topic=%q, method=%q", socket.Id, upReq.Topic, upReq.Method)
				if payload.Ack != nil {
					payload.Ack(string(BuildResp(CodeParamErr, "reqId|topic|method|payload为必填项", "", "", nil, 0, upReq.ReqId, payload.SID)))
				} else {
					_ = session.ReplyDown(CodeParamErr, "reqId|topic|method|payload为必填项", "", "", nil, upReq.ReqId)
				}
				return
			}
			logx.WithContext(ctx).Debugf("[socketio] processing request: conn=%s, topic=%q, method=%q", socket.Id, upReq.Topic, upReq.Method)
			replyTopic := fmt.Sprintf("%s_%s", upReq.Topic, "reply")
			threading.GoSafe(func() {
				ack := payload.Ack
				if upHandler := s.eventHandlers[EventUp]; upHandler != nil {
					err := upHandler(ctx, EventUp, payload)
					if err != nil {
						logx.WithContext(ctx).Errorf("[socketio] failed to process request: conn=%s, err=%v", socket.Id, err)
						if ack != nil {
							ack(string(BuildResp(CodeBizErr, "业务处理失败", replyTopic, upReq.Method, nil, 0, upReq.ReqId, payload.SID)))
						} else {
							_ = session.ReplyDown(CodeBizErr, "业务处理失败", replyTopic, upReq.Method, nil, upReq.ReqId)
						}
					} else {
						if ack != nil {
							ack(string(BuildResp(CodeSuccess, "处理成功", replyTopic, upReq.Method, nil, 0, upReq.ReqId, payload.SID)))
						} else {
							_ = session.ReplyDown(CodeSuccess, "处理成功", replyTopic, upReq.Method, nil, upReq.ReqId)
						}
					}
				} else {
					logx.WithContext(ctx).Debugf("[socketio] no handler registered for EventUp: conn=%s", socket.Id)
					if ack != nil {
						ack(string(BuildResp(CodeBizErr, "未配置处理器", replyTopic, upReq.Method, nil, 0, upReq.ReqId, payload.SID)))
					} else {
						_ = session.ReplyDown(CodeBizErr, "未配置处理器", replyTopic, upReq.Method, nil, upReq.ReqId)
					}
				}
			})
		})
		for eventName, handler := range s.eventHandlers {
			if eventName == EventDown || eventName == EventUp {
				continue
			}
			currentEvent := eventName
			currentHandler := handler
			socket.On(currentEvent, func(payload *socketio.EventPayload) {
				ctx := logx.WithFields(context.WithValue(context.Background(), "SID", payload.SID), logx.Field("SID", payload.SID))
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
			logx.Infof("[socketio] disconnecting: conn=%s, reason=%s, total=%d", socket.Id, reason, s.SessionCount())
			s.cleanInvalidSession(socket.Id)
		})
	})
}

func (s *Server) StartSeqSync() {
	if s.seqSyncInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.seqSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logx.Debugf("[socketio] starting sequence synchronization")
			s.lock.RLock()
			sessions := make([]*Session, 0, len(s.sessions))
			for _, sess := range s.sessions {
				sessions = append(sessions, sess)
			}
			s.lock.RUnlock()
			sockets := s.Io.Sockets()
			if len(sockets) != len(sessions) {
				logx.Errorf("[socketio] session count mismatch: sessions=%d, sockets=%d", len(sessions), len(sockets))
			}
			logx.Statf("[socketio] total sessions: %d", len(sessions))
			for _, sess := range sessions {
				threading.GoSafe(func() {
					seqData := make(map[string]int64)
					for seqKey, seqPtr := range sess.seqNum {
						seqData[seqKey] = atomic.LoadInt64(seqPtr)
					}
					data := map[string]any{
						"sId":    sess.id,
						"seqKey": seqData,
					}
					payload, _ := jsonx.Marshal(data)
					logx.Debugf("[socketio] sending sequence synchronization: %s", string(payload))
					sess.EmitString(EventSeqSync, string(payload))
				})
			}
		case <-s.stopChan:
			logx.Infof("[socketio] sequence synchronization stopped")
			return
		}
	}
}

func (s *Server) cleanInvalidSession(sId string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	logx.Debugf("[socketio] cleaning invalid session: %s", sId)
	delete(s.sessions, sId)
}

func (s *Server) SessionCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.sessions)
}

func (s *Server) GetSession(sId string) *Session {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.sessions[sId]
}

func (s *Server) GetSessionByDeviceID(deviceID string) *Session {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, sess := range s.sessions {
		if sess.GetMetadata("deviceId") == deviceID {
			return sess
		}
	}
	return nil
}

func (s *Server) JoinRoom(sId string, roomId string) {
	s.lock.RLock()
	session, ok := s.sessions[sId]
	s.lock.RUnlock()
	if ok {
		session.JoinRoom(roomId)
	}
}

func (s *Server) LeaveRoom(sId string, roomId string) {
	s.lock.RLock()
	session, ok := s.sessions[sId]
	s.lock.RUnlock()
	if ok {
		session.LeaveRoom(roomId)
	}
}

func (s *Server) Stop() {
	close(s.stopChan)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.sessions = make(map[string]*Session)
	s.Close()
	logx.Info("[socketio] server stopped")
}
