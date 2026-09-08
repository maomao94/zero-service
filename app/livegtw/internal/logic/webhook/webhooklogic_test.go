package webhook

import (
	"context"
	"errors"
	"testing"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/config"
	"zero-service/app/livegtw/internal/svc"

	"github.com/livekit/protocol/livekit"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// errForward 模拟 WebhookNotify RPC 转发失败。
var errForward = errors.New("rpc unavailable")

type fakeLiveRpcCli struct {
	live.LiveRpcClient
	notifiedData []byte
	err          error
}

func (f *fakeLiveRpcCli) CreateMeeting(ctx context.Context, in *live.CreateMeetingReq, opts ...grpc.CallOption) (*live.CreateMeetingRes, error) {
	return &live.CreateMeetingRes{}, nil
}
func (f *fakeLiveRpcCli) JoinMeeting(ctx context.Context, in *live.JoinMeetingReq, opts ...grpc.CallOption) (*live.JoinMeetingRes, error) {
	return &live.JoinMeetingRes{}, nil
}
func (f *fakeLiveRpcCli) GetMeeting(ctx context.Context, in *live.GetMeetingReq, opts ...grpc.CallOption) (*live.GetMeetingRes, error) {
	return &live.GetMeetingRes{}, nil
}
func (f *fakeLiveRpcCli) ListMeetings(ctx context.Context, in *live.ListMeetingsReq, opts ...grpc.CallOption) (*live.ListMeetingsRes, error) {
	return &live.ListMeetingsRes{}, nil
}
func (f *fakeLiveRpcCli) EndMeeting(ctx context.Context, in *live.EndMeetingReq, opts ...grpc.CallOption) (*live.EndMeetingRes, error) {
	return &live.EndMeetingRes{}, nil
}
func (f *fakeLiveRpcCli) KickParticipant(ctx context.Context, in *live.KickParticipantReq, opts ...grpc.CallOption) (*live.KickParticipantRes, error) {
	return &live.KickParticipantRes{}, nil
}
func (f *fakeLiveRpcCli) MuteParticipant(ctx context.Context, in *live.MuteParticipantReq, opts ...grpc.CallOption) (*live.MuteParticipantRes, error) {
	return &live.MuteParticipantRes{}, nil
}
func (f *fakeLiveRpcCli) ListParticipants(ctx context.Context, in *live.ListParticipantsReq, opts ...grpc.CallOption) (*live.ListParticipantsRes, error) {
	return &live.ListParticipantsRes{}, nil
}
func (f *fakeLiveRpcCli) SendMeetingData(ctx context.Context, in *live.SendMeetingDataReq, opts ...grpc.CallOption) (*live.SendMeetingDataRes, error) {
	return &live.SendMeetingDataRes{}, nil
}
func (f *fakeLiveRpcCli) PerformMeetingRpc(ctx context.Context, in *live.PerformMeetingRpcReq, opts ...grpc.CallOption) (*live.PerformMeetingRpcRes, error) {
	return &live.PerformMeetingRpcRes{}, nil
}
func (f *fakeLiveRpcCli) WebhookNotify(ctx context.Context, in *live.WebhookNotifyReq, opts ...grpc.CallOption) (*live.WebhookNotifyRes, error) {
	f.notifiedData = in.GetData()
	return &live.WebhookNotifyRes{}, f.err
}
func (f *fakeLiveRpcCli) GenerateMeetingTicket(ctx context.Context, in *live.GenerateMeetingTicketReq, opts ...grpc.CallOption) (*live.GenerateMeetingTicketRes, error) {
	return &live.GenerateMeetingTicketRes{}, nil
}
func (f *fakeLiveRpcCli) JoinMeetingByTicket(ctx context.Context, in *live.JoinMeetingByTicketReq, opts ...grpc.CallOption) (*live.JoinMeetingByTicketRes, error) {
	return &live.JoinMeetingByTicketRes{}, nil
}
func (f *fakeLiveRpcCli) ReportMeetingMessage(ctx context.Context, in *live.ReportMeetingMessageReq, opts ...grpc.CallOption) (*live.ReportMeetingMessageRes, error) {
	return &live.ReportMeetingMessageRes{}, nil
}
func (f *fakeLiveRpcCli) ListMeetingMessages(ctx context.Context, in *live.ListMeetingMessagesReq, opts ...grpc.CallOption) (*live.ListMeetingMessagesRes, error) {
	return &live.ListMeetingMessagesRes{}, nil
}

func TestWebhookNotifyMarshalsAndForwards(t *testing.T) {
	fake := &fakeLiveRpcCli{}
	svcCtx := &svc.ServiceContext{Config: config.Config{}, LiveRpcCli: fake}
	l := NewWebhookNotifyLogic(context.Background(), svcCtx)

	event := &livekit.WebhookEvent{
		Id:    "EVT-1",
		Event: "participant_joined",
		Room:  &livekit.Room{Name: "M001"},
		Participant: &livekit.ParticipantInfo{
			Identity: "bob",
			Name:     "Bob",
		},
	}
	if err := l.WebhookNotify(event); err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(fake.notifiedData) == 0 {
		t.Fatal("notified data should not be empty")
	}
	// 数据应能反序列化回相同的 WebhookEvent
	var got livekit.WebhookEvent
	if err := proto.Unmarshal(fake.notifiedData, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.GetId() != "EVT-1" || got.GetEvent() != "participant_joined" || got.GetRoom().GetName() != "M001" || got.GetParticipant().GetIdentity() != "bob" {
		t.Fatalf("round-trip mismatch: %+v", &got)
	}
}

func TestWebhookNotifyForwardsError(t *testing.T) {
	fake := &fakeLiveRpcCli{err: errForward}
	svcCtx := &svc.ServiceContext{Config: config.Config{}, LiveRpcCli: fake}
	l := NewWebhookNotifyLogic(context.Background(), svcCtx)

	if err := l.WebhookNotify(&livekit.WebhookEvent{Id: "EVT-2", Event: "room_finished"}); err == nil {
		t.Fatal("want error, got nil")
	}
}
