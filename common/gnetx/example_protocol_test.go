package gnetx

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"testing"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/stat"
)

// =========================================================================
// 示例协议: [1B msgType][4B seq][4B ack][4B bodyLen][N body]
//
// msgType: 1=req  2=resp  3=push
//
// 此协议演示 gnetx 洋葱模型的完整用法:
//   框架层: NextSendSeq / PacketContextKey / PacketContextProvider / ctx注入
//   协议层(Codec): 解析/构造协议头, seq=NextSendSeq, ack=PacketContext.seq
//   业务层(Serializer): 只管 body 字节
// =========================================================================

const (
	seqMsgTypeReq  = 1
	seqMsgTypeResp = 2
	seqMsgTypePush = 3
)

const seqHeaderLen = 13

// seqPacketCtx 协议头上下文 — 请求级, 通过 PacketContextProvider 传递.
type seqPacketCtx struct {
	Seq uint32
	Ack uint32
}

// seqReqMsg 请求消息.
type seqReqMsg struct {
	Seq    uint32
	Ack    uint32
	TidVal string
	Body   string
}

func (m *seqReqMsg) TID() string        { return m.TidVal }
func (m *seqReqMsg) PacketContext() any { return &seqPacketCtx{Seq: m.Seq, Ack: m.Ack} }

// seqRespMsg 回复消息.
type seqRespMsg struct {
	Seq  uint32
	Ack  uint32
	Body string
}

func (m *seqRespMsg) ResponseTID() string { return strconv.FormatUint(uint64(m.Ack), 10) }
func (m *seqRespMsg) PacketContext() any  { return &seqPacketCtx{Seq: m.Seq, Ack: m.Ack} }

// seqPushMsg 推送消息(不需要回复).
type seqPushMsg struct {
	Seq  uint32
	Body string
}

func (m *seqPushMsg) PacketContext() any { return &seqPacketCtx{Seq: m.Seq, Ack: 0} }

// seqProtocolCodec — 协议层 Codec.
type seqProtocolCodec struct {
	// 无状态: seq 来自 conn.NextSendSeq, ack 来自 ctx.PacketContextKey
}

func (c *seqProtocolCodec) Decode(gconn gnet.Conn, _ Conn) (any, error) {
	hdr, err := gconn.Peek(seqHeaderLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	msgType := hdr[0]
	seq := binary.BigEndian.Uint32(hdr[1:5])
	ack := binary.BigEndian.Uint32(hdr[5:9])
	bodyLen := binary.BigEndian.Uint32(hdr[9:13])

	frameLen := seqHeaderLen + int(bodyLen)
	frame, err := gconn.Peek(frameLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	body := string(frame[seqHeaderLen:])
	if _, err := gconn.Discard(frameLen); err != nil {
		return nil, mapShortBuffer(err)
	}

	switch msgType {
	case seqMsgTypeReq:
		return &seqReqMsg{Seq: seq, Ack: ack, Body: body}, nil
	case seqMsgTypeResp:
		return &seqRespMsg{Seq: seq, Ack: ack, Body: body}, nil
	case seqMsgTypePush:
		return &seqPushMsg{Seq: seq, Body: body}, nil
	}
	return nil, errors.New("unknown msgType")
}

func (c *seqProtocolCodec) Encode(ctx context.Context, msg any, conn Conn) ([]byte, error) {
	seq := uint32(conn.NextSendSeq())
	ack := uint32(0)

	if pc, ok := ctx.Value(PacketContextKey).(*seqPacketCtx); ok {
		ack = pc.Seq // 回复时 ack = 远端 seq
	}

	var msgType byte
	var body string
	switch m := msg.(type) {
	case *seqReqMsg:
		msgType = seqMsgTypeReq
		body = m.Body
		m.Seq = seq
		m.Ack = ack
	case *seqRespMsg:
		msgType = seqMsgTypeResp
		body = m.Body
		m.Seq = seq
		m.Ack = ack
	case *seqPushMsg:
		msgType = seqMsgTypePush
		body = m.Body
		m.Seq = seq
	default:
		return nil, errors.New("unknown message type for encode")
	}

	out := make([]byte, seqHeaderLen+len(body))
	out[0] = msgType
	binary.BigEndian.PutUint32(out[1:5], seq)
	binary.BigEndian.PutUint32(out[5:9], ack)
	binary.BigEndian.PutUint32(out[9:13], uint32(len(body)))
	copy(out[13:], body)
	return out, nil
}

// =========================================================================
// encodeFrame 测试辅助: 用 codec + mockConn 编码一帧.
// =========================================================================
func encodeFrame(t *testing.T, codec Codec, ctx context.Context, msg any, conn Conn) []byte {
	t.Helper()
	out, err := codec.Encode(ctx, msg, conn)
	if err != nil {
		t.Fatalf("encodeFrame: %v", err)
	}
	return out
}

func newSeqSession() *session {
	codec := &seqProtocolCodec{}
	return newSession("test-seq", newMockConn(nil), codec, nil, nil)
}

func newTestServer(opts ServerOptions) *Server {
	return &Server{
		opts:    opts,
		mgr:     NewSessionManager(nil),
		pool:    defaultWorkerPool(),
		tracer:  gnetxTracer(),
		metrics: stat.NewMetrics("test"),
	}
}

// =========================================================================
// 测试: Encode 无 PacketContext — 主动发送场景 (ack=0)
// =========================================================================
func TestEncodeActiveRequestNoPacketContext(t *testing.T) {
	cn := newSeqSession()
	codec := &seqProtocolCodec{}
	ctx := context.Background()

	out := encodeFrame(t, codec, ctx, &seqReqMsg{Body: "hello"}, cn)
	if len(out) < seqHeaderLen {
		t.Fatal("frame too short")
	}

	seq := binary.BigEndian.Uint32(out[1:5])
	ack := binary.BigEndian.Uint32(out[5:9])
	bodyLen := binary.BigEndian.Uint32(out[9:13])

	if seq != 0 {
		t.Fatalf("active request seq = %d, want 0 (first NextSendSeq)", seq)
	}
	if ack != 0 {
		t.Fatalf("active request ack = %d, want 0 (no PacketContext)", ack)
	}
	if bodyLen != 5 {
		t.Fatalf("bodyLen = %d, want 5", bodyLen)
	}
	if string(out[13:]) != "hello" {
		t.Fatalf("body = %q", out[13:])
	}
}

// =========================================================================
// 测试: Encode 有 PacketContext — 回复场景 (ack = 远端 seq)
// 模拟: Server 收到请求(seq=10), handler 返回 reply, dispatch 注入 pc 到 ctx
// =========================================================================
func TestEncodeReplyWithPacketContext(t *testing.T) {
	cn := newSeqSession()
	codec := &seqProtocolCodec{}

	// 模拟 dispatch 把请求的 PacketContext 注入 ctx
	pc := &seqPacketCtx{Seq: 10, Ack: 0}
	ctx := context.WithValue(context.Background(), PacketContextKey, pc)

	out := encodeFrame(t, codec, ctx, &seqRespMsg{Body: "world"}, cn)

	seq := binary.BigEndian.Uint32(out[1:5])
	ack := binary.BigEndian.Uint32(out[5:9])

	// seq 来自 NextSendSeq(首次=0), ack 来自请求的 seq
	if seq != 0 {
		t.Fatalf("reply seq = %d, want 0", seq)
	}
	if ack != 10 {
		t.Fatalf("reply ack = %d, want 10 (远端请求 seq)", ack)
	}
}

// =========================================================================
// 测试: 回复后请求方 Decode 拿到正确的 ack
// =========================================================================
func TestDecodeReplyFrame(t *testing.T) {
	codec := &seqProtocolCodec{}
	cn := newSeqSession()

	// 模拟: 远端收到我们的请求(seq=0)后回复, seq=100, ack=0(我们的请求seq)
	ctx := context.Background()
	resp := &seqRespMsg{Body: "ok"}
	out := encodeFrame(t, codec, ctx, resp, cn)

	mc := newMockConn(out)
	msg, err := codec.Decode(mc, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	r, ok := msg.(*seqRespMsg)
	if !ok {
		t.Fatalf("msg type = %T, want *seqRespMsg", msg)
	}
	// 回复帧 seq=0(NextSendSeq首次), ack=0(主动请求无 pc)
	if r.Seq != 0 {
		t.Fatalf("decoded seq = %d, want 0", r.Seq)
	}
	if r.Ack != 0 {
		t.Fatalf("decoded ack = %d, want 0", r.Ack)
	}
	if r.Body != "ok" {
		t.Fatalf("body = %q", r.Body)
	}
	// ReplyPool 要通过 ResponseTID() 匹配 — 这里 ack=0 → ResponseTID="0"
	if r.ResponseTID() != "0" {
		t.Fatalf("ResponseTID = %q, want \"0\"", r.ResponseTID())
	}
}

// =========================================================================
// 测试: dispatchSync 将消息的 PacketContext 注入 ctx — reply 路径闭环
// =========================================================================
func TestDispatchInjectsPacketContextIntoReplyCtx(t *testing.T) {
	// 构造一对消息, 模拟 server 收到请求 → handler 回复
	req := &seqReqMsg{Seq: 42, Ack: 0, Body: "ping"}
	resp := &seqRespMsg{Body: "pong"}

	var capturedCtx context.Context
	handler := HandlerFunc(func(ctx context.Context, conn Conn, msg any) (any, error) {
		capturedCtx = ctx
		return resp, nil
	})

	codec := &seqProtocolCodec{}
	cn := newSession("test", newMockConn(nil), codec, nil, nil)

	srv := newTestServer(ServerOptions{Codec: codec})
	srv.dispatchSync(cn, req, handler)

	// handler 收到的 ctx 中应有 PacketContext
	if pc, ok := capturedCtx.Value(PacketContextKey).(*seqPacketCtx); !ok || pc.Seq != 42 {
		t.Fatalf("handler ctx missing PacketContext or Seq wrong: %+v", pc)
	}
	// reply 的 Seq 应被 Encode 设置为 NextSendSeq(0), Ack 应为请求的 seq(42)
	if resp.Seq != 0 {
		t.Fatalf("reply Seq = %d, want 0", resp.Seq)
	}
	if resp.Ack != 42 {
		t.Fatalf("reply Ack = %d, want 42 (incoming req seq)", resp.Ack)
	}
}

// =========================================================================
// 测试: 消息不实现 PacketContextProvider — ctx 不带 PacketContext
// =========================================================================
type plainMsg struct{ Body string }

func TestDispatchNoPacketContextProvider(t *testing.T) {
	msg := &plainMsg{Body: "hi"}
	var capturedCtx context.Context
	handler := HandlerFunc(func(ctx context.Context, conn Conn, msg any) (any, error) {
		capturedCtx = ctx
		return nil, nil
	})

	codec := &seqProtocolCodec{}
	cn := newSession("test", newMockConn(nil), codec, nil, nil)

	srv := newTestServer(ServerOptions{Codec: codec})
	srv.dispatchSync(cn, msg, handler)

	if capturedCtx.Value(PacketContextKey) != nil {
		t.Fatal("ctx should NOT have PacketContext when msg doesn't implement PacketContextProvider")
	}
}

// =========================================================================
// 测试: 推送消息 — handler 收到 PacketContext, 但 reply=nil (不回复)
// =========================================================================
func TestPushMessagePacketContext(t *testing.T) {
	push := &seqPushMsg{Seq: 99, Body: "event"}
	var capturedCtx context.Context
	handler := HandlerFunc(func(ctx context.Context, conn Conn, msg any) (any, error) {
		capturedCtx = ctx
		return nil, nil
	})

	codec := &seqProtocolCodec{}
	cn := newSession("test", newMockConn(nil), codec, nil, nil)

	srv := newTestServer(ServerOptions{Codec: codec})
	srv.dispatchSync(cn, push, handler)

	pc, ok := capturedCtx.Value(PacketContextKey).(*seqPacketCtx)
	if !ok {
		t.Fatal("push msg should provide PacketContext")
	}
	if pc.Seq != 99 {
		t.Fatalf("push PacketContext seq = %d, want 99", pc.Seq)
	}
}

// =========================================================================
// 测试: Conn.NextSendSeq 连续递增 — 多次发送
// =========================================================================
func TestNextSendSeqIncrementsAcrossFrames(t *testing.T) {
	codec := &seqProtocolCodec{}
	cn := newSeqSession()

	ctx := context.Background()
	for i := uint32(0); i < 10; i++ {
		out := encodeFrame(t, codec, ctx, &seqReqMsg{Body: "x"}, cn)
		seq := binary.BigEndian.Uint32(out[1:5])
		if seq != i {
			t.Fatalf("frame %d seq = %d, want %d", i, seq, i)
		}
	}
}

// =========================================================================
// 测试: goroutine 间 conn.Send 触发 Encode — ctx 透传
// =========================================================================
func TestSessionSendPassesContextToProtocolCodec(t *testing.T) {
	type ctxKey struct{}
	codec := &seqProtocolCodec{}
	cn := newSession("test-send", newMockConn(nil), codec, nil, nil)

	ctx := context.WithValue(context.Background(), ctxKey{}, "send-ctx")
	if err := cn.Send(ctx, &seqPushMsg{Body: "go"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// mockConn in Encode is ok — we just verify no panic and seq increments
	if seq := cn.NextSendSeq(); seq != 1 {
		t.Fatalf("after Send NextSendSeq = %d, want 1 (one increment for frame seq)", seq)
	}
}

// =========================================================================
// 测试: server 接收 req 后回复 — 完整洋葱链路
// 模拟: mockConn 上放一帧, OnTraffic Decode → dispatch → writeReply → Encode
// =========================================================================
func TestServerOnionEndToEnd(t *testing.T) {
	codec := &seqProtocolCodec{}

	// 构造入站帧: msgType=1(请求), seq=7, ack=0, body="ping"
	reqFrame := make([]byte, seqHeaderLen+4)
	reqFrame[0] = seqMsgTypeReq
	binary.BigEndian.PutUint32(reqFrame[1:5], 7)
	binary.BigEndian.PutUint32(reqFrame[5:9], 0)
	binary.BigEndian.PutUint32(reqFrame[9:13], 4)
	copy(reqFrame[13:], "ping")

	mc := newMockConn(reqFrame)
	mgr := NewSessionManager(nil)
	cn := newSession("end2end", mc, codec, mgr, nil)
	mc.SetContext(cn)

	var handlerAck uint32
	handler := HandlerFunc(func(ctx context.Context, conn Conn, msg any) (any, error) {
		if req, ok := msg.(*seqReqMsg); ok {
			handlerAck = req.Seq
			return &seqRespMsg{Body: "pong"}, nil
		}
		return nil, nil
	})

	srv := newTestServer(ServerOptions{Codec: codec, Handler: handler, BatchReadLimit: 1})
	_ = srv.OnTraffic(mc)

	// handler 应读到 seq=7
	if handlerAck != 7 {
		t.Fatalf("handler ack = %d, want 7", handlerAck)
	}

	// mockConn.buf 应收到回复帧: msgType=2, seq=0(server NextSendSeq), ack=7
	replyBytes := mc.buf.Bytes()
	if len(replyBytes) < seqHeaderLen {
		t.Fatalf("reply frame too short: %d bytes", len(replyBytes))
	}
	replyMsgType := replyBytes[0]
	replySeq := binary.BigEndian.Uint32(replyBytes[1:5])
	replyAck := binary.BigEndian.Uint32(replyBytes[5:9])

	if replyMsgType != seqMsgTypeResp {
		t.Fatalf("reply msgType = %d, want %d", replyMsgType, seqMsgTypeResp)
	}
	if replySeq != 0 {
		t.Fatalf("reply seq = %d, want 0 (server first NextSendSeq)", replySeq)
	}
	if replyAck != 7 {
		t.Fatalf("reply ack = %d, want 7 (incoming req seq)", replyAck)
	}
}

// =========================================================================
// 测试: server 主动发请求(无 PacketContext), client 回复 — 双向 seq/ack
// =========================================================================
func TestBidirectionalSeqAck(t *testing.T) {
	codec := &seqProtocolCodec{}

	// --- 服务端主动发给客户端 ---
	serverCn := newSeqSession()
	// 模拟 server 主动请求: seq=0, ack=0
	reqRaw := encodeFrame(t, codec, context.Background(), &seqReqMsg{Body: "query"}, serverCn)
	// seq=0, ack=0

	// --- 客户端收到, Decode ---
	mc := newMockConn(reqRaw)

	req, err := codec.Decode(mc, nil)
	if err != nil {
		t.Fatalf("client decode: %v", err)
	}
	reqMsg := req.(*seqReqMsg)
	if reqMsg.Seq != 0 {
		t.Fatalf("client received seq = %d, want 0", reqMsg.Seq)
	}

	// --- 客户端回复 ---
	clientCn := newSeqSession()
	respCtx := context.WithValue(context.Background(), PacketContextKey, reqMsg.PacketContext())
	respRaw := encodeFrame(t, codec, respCtx, &seqRespMsg{Body: "result"}, clientCn)

	respSeq := binary.BigEndian.Uint32(respRaw[1:5])
	respAck := binary.BigEndian.Uint32(respRaw[5:9])

	// 客户端 seq 从自己的 NextSendSeq(0), ack=server 的 seq(0)
	if respSeq != 0 {
		t.Fatalf("client reply seq = %d, want 0", respSeq)
	}
	if respAck != 0 {
		t.Fatalf("client reply ack = %d, want 0 (server request seq)", respAck)
	}

	// --- 服务端收到回复, Decode ---
	mc2 := newMockConn(respRaw)
	resp, err := codec.Decode(mc2, nil)
	if err != nil {
		t.Fatalf("server decode reply: %v", err)
	}
	respMsg := resp.(*seqRespMsg)
	if respMsg.Ack != 0 {
		t.Fatalf("server received reply ack = %d, want 0", respMsg.Ack)
	}
	if respMsg.Body != "result" {
		t.Fatalf("body = %q", respMsg.Body)
	}
}

// =========================================================================
// 测试: 直推(双推) — 无回复周期, PacketContext 仅用于 handler 读取
// =========================================================================
func TestPushBidirectional(t *testing.T) {
	codec := &seqProtocolCodec{}

	// Server→Client push
	serverCn := newSeqSession()
	pushRaw := encodeFrame(t, codec, context.Background(), &seqPushMsg{Body: "alert"}, serverCn)
	// seq=0, ack=0(Encode 无 PacketContext)

	// Client 收到 push
	mc := newMockConn(pushRaw)
	push, err := codec.Decode(mc, nil)
	if err != nil {
		t.Fatalf("client decode push: %v", err)
	}
	pushMsg := push.(*seqPushMsg)
	if pushMsg.Seq != 0 {
		t.Fatalf("push seq = %d, want 0", pushMsg.Seq)
	}
	if pushMsg.Body != "alert" {
		t.Fatalf("push body = %q", pushMsg.Body)
	}

	// PacketContext 可读, handler 可记录 seq
	pc := pushMsg.PacketContext().(*seqPacketCtx)
	if pc.Seq != 0 {
		t.Fatalf("PacketContext seq = %d, want 0", pc.Seq)
	}
	if pc.Ack != 0 {
		t.Fatalf("push PacketContext ack should be 0, got %d", pc.Ack)
	}
}

// =========================================================================
// ensure mockConn implements gnet.Conn (compile-time check)
// =========================================================================
var _ gnet.Conn = (*mockConn)(nil)
