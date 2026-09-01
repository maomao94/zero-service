package meeting

import (
	"context"
	"encoding/base64"
	"testing"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/config"
	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"

	"google.golang.org/grpc"
)

// fakeLiveRpcCli 记录调用参数并返回可配置结果。
type fakeLiveRpcCli struct {
	createReq   *live.CreateMeetingReq
	joinReq     *live.JoinMeetingReq
	joinRes     *live.JoinMeetingRes
	getReq      *live.GetMeetingReq
	listReq     *live.ListMeetingsReq
	endReq      *live.EndMeetingReq
	kickReq     *live.KickParticipantReq
	muteReq     *live.MuteParticipantReq
	listPartReq *live.ListParticipantsReq
	sendReq     *live.SendMeetingDataReq
	rpcReq      *live.PerformMeetingRpcReq
	rpcRes      *live.PerformMeetingRpcRes
	err         error
}

func (f *fakeLiveRpcCli) CreateMeeting(ctx context.Context, in *live.CreateMeetingReq, opts ...grpc.CallOption) (*live.CreateMeetingRes, error) {
	f.createReq = in
	return &live.CreateMeetingRes{Meeting: &live.MeetingInfo{MeetingNo: "M1", Title: in.GetTitle()}}, f.err
}

func (f *fakeLiveRpcCli) JoinMeeting(ctx context.Context, in *live.JoinMeetingReq, opts ...grpc.CallOption) (*live.JoinMeetingRes, error) {
	f.joinReq = in
	if f.joinRes != nil {
		return f.joinRes, f.err
	}
	return &live.JoinMeetingRes{Token: "tok", WsUrl: "ws://x"}, f.err
}

func (f *fakeLiveRpcCli) GetMeeting(ctx context.Context, in *live.GetMeetingReq, opts ...grpc.CallOption) (*live.GetMeetingRes, error) {
	f.getReq = in
	return &live.GetMeetingRes{Meeting: &live.MeetingInfo{MeetingNo: in.GetMeetingNo()}}, f.err
}

func (f *fakeLiveRpcCli) ListMeetings(ctx context.Context, in *live.ListMeetingsReq, opts ...grpc.CallOption) (*live.ListMeetingsRes, error) {
	f.listReq = in
	return &live.ListMeetingsRes{}, f.err
}

func (f *fakeLiveRpcCli) EndMeeting(ctx context.Context, in *live.EndMeetingReq, opts ...grpc.CallOption) (*live.EndMeetingRes, error) {
	f.endReq = in
	return &live.EndMeetingRes{}, f.err
}

func (f *fakeLiveRpcCli) KickParticipant(ctx context.Context, in *live.KickParticipantReq, opts ...grpc.CallOption) (*live.KickParticipantRes, error) {
	f.kickReq = in
	return &live.KickParticipantRes{}, f.err
}

func (f *fakeLiveRpcCli) MuteParticipant(ctx context.Context, in *live.MuteParticipantReq, opts ...grpc.CallOption) (*live.MuteParticipantRes, error) {
	f.muteReq = in
	return &live.MuteParticipantRes{}, f.err
}

func (f *fakeLiveRpcCli) ListParticipants(ctx context.Context, in *live.ListParticipantsReq, opts ...grpc.CallOption) (*live.ListParticipantsRes, error) {
	f.listPartReq = in
	return &live.ListParticipantsRes{}, f.err
}

func (f *fakeLiveRpcCli) SendMeetingData(ctx context.Context, in *live.SendMeetingDataReq, opts ...grpc.CallOption) (*live.SendMeetingDataRes, error) {
	f.sendReq = in
	return &live.SendMeetingDataRes{}, f.err
}

func (f *fakeLiveRpcCli) PerformMeetingRpc(ctx context.Context, in *live.PerformMeetingRpcReq, opts ...grpc.CallOption) (*live.PerformMeetingRpcRes, error) {
	f.rpcReq = in
	if f.rpcRes != nil {
		return f.rpcRes, f.err
	}
	return &live.PerformMeetingRpcRes{Response: "pong"}, f.err
}

func (f *fakeLiveRpcCli) WebhookNotify(ctx context.Context, in *live.WebhookNotifyReq, opts ...grpc.CallOption) (*live.WebhookNotifyRes, error) {
	return &live.WebhookNotifyRes{}, f.err
}

func newSvcCtx(fake *fakeLiveRpcCli) *svc.ServiceContext {
	return &svc.ServiceContext{Config: config.Config{}, LiveRpcCli: fake}
}

func TestCreateMeetingMapsRequest(t *testing.T) {
	fake := &fakeLiveRpcCli{}
	svcCtx := newSvcCtx(fake)
	l := NewCreateMeetingLogic(context.Background(), svcCtx)

	resp, err := l.CreateMeeting(&types.CreateMeetingReq{Title: "晨会"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if fake.createReq.GetTitle() != "晨会" {
		t.Fatalf("title not mapped: %+v", fake.createReq)
	}
	if resp.Meeting.MeetingNo != "M1" || resp.Meeting.Title != "晨会" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestJoinMeetingMapsRequest(t *testing.T) {
	fake := &fakeLiveRpcCli{joinRes: &live.JoinMeetingRes{Token: "T", WsUrl: "W"}}
	svcCtx := newSvcCtx(fake)
	l := NewJoinMeetingLogic(context.Background(), svcCtx)

	resp, err := l.JoinMeeting(&types.JoinMeetingReq{MeetingNo: "M1", Identity: "bob", Name: "Bob"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if fake.joinReq.GetMeetingNo() != "M1" || fake.joinReq.GetIdentity() != "bob" || fake.joinReq.GetName() != "Bob" {
		t.Fatalf("join req not mapped: %+v", fake.joinReq)
	}
	if resp.Token != "T" || resp.WsUrl != "W" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestSendMeetingDataDecodesPayload(t *testing.T) {
	fake := &fakeLiveRpcCli{}
	svcCtx := newSvcCtx(fake)
	l := NewSendMeetingDataLogic(context.Background(), svcCtx)

	raw := []byte("hello")
	err := l.SendMeetingData(&types.SendMeetingDataReq{
		MeetingNo: "M1", Topic: "chat", Payload: base64.StdEncoding.EncodeToString(raw),
		Destinations: []string{"bob"},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if fake.sendReq.GetMeetingNo() != "M1" || fake.sendReq.GetTopic() != "chat" || string(fake.sendReq.GetPayload()) != "hello" {
		t.Fatalf("send req not mapped: %+v", fake.sendReq)
	}
	if len(fake.sendReq.GetDestinations()) != 1 || fake.sendReq.GetDestinations()[0] != "bob" {
		t.Fatalf("destinations not mapped: %+v", fake.sendReq.GetDestinations())
	}
}

func TestSendMeetingDataInvalidBase64(t *testing.T) {
	svcCtx := newSvcCtx(&fakeLiveRpcCli{})
	l := NewSendMeetingDataLogic(context.Background(), svcCtx)
	if err := l.SendMeetingData(&types.SendMeetingDataReq{MeetingNo: "M1", Topic: "t", Payload: "!!not-base64!!"}); err == nil {
		t.Fatal("want error for invalid base64")
	}
}

func TestPerformMeetingRpcMapsRequest(t *testing.T) {
	fake := &fakeLiveRpcCli{rpcRes: &live.PerformMeetingRpcRes{Response: "resp"}}
	svcCtx := newSvcCtx(fake)
	l := NewPerformMeetingRpcLogic(context.Background(), svcCtx)

	resp, err := l.PerformMeetingRpc(&types.PerformMeetingRpcReq{
		MeetingNo: "M1", Identity: "bob", Method: "echo", Payload: "p", ResponseTimeoutMs: 8000,
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if fake.rpcReq.GetMeetingNo() != "M1" || fake.rpcReq.GetIdentity() != "bob" || fake.rpcReq.GetMethod() != "echo" || fake.rpcReq.GetResponseTimeoutMs() != 8000 {
		t.Fatalf("rpc req not mapped: %+v", fake.rpcReq)
	}
	if resp.Response != "resp" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}
