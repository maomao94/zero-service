package webhook

import (
	"bytes"
	"context"
	"net/http"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/config"
	"zero-service/app/livegtw/internal/svc"

	"google.golang.org/grpc"
)

// fakeLiveRpcCli 实现 live.LiveRpcClient，用于 webhook handler 测试。
type fakeLiveRpcCli struct {
	live.LiveRpcClient
	webhookData   []byte
	webhookCalled bool
	webhookErr    error
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
	f.webhookCalled = true
	f.webhookData = in.GetData()
	return &live.WebhookNotifyRes{}, f.webhookErr
}
func (f *fakeLiveRpcCli) GenerateMeetingTicket(ctx context.Context, in *live.GenerateMeetingTicketReq, opts ...grpc.CallOption) (*live.GenerateMeetingTicketRes, error) {
	return &live.GenerateMeetingTicketRes{}, nil
}
func (f *fakeLiveRpcCli) JoinMeetingByTicket(ctx context.Context, in *live.JoinMeetingByTicketReq, opts ...grpc.CallOption) (*live.JoinMeetingByTicketRes, error) {
	return &live.JoinMeetingByTicketRes{}, nil
}
func (f *fakeLiveRpcCli) NotifyMeetingParticipant(ctx context.Context, in *live.NotifyMeetingParticipantReq, opts ...grpc.CallOption) (*live.NotifyMeetingParticipantRes, error) {
	return &live.NotifyMeetingParticipantRes{}, nil
}
func (f *fakeLiveRpcCli) ReportMeetingMessage(ctx context.Context, in *live.ReportMeetingMessageReq, opts ...grpc.CallOption) (*live.ReportMeetingMessageRes, error) {
	return &live.ReportMeetingMessageRes{}, nil
}
func (f *fakeLiveRpcCli) ListMeetingMessages(ctx context.Context, in *live.ListMeetingMessagesReq, opts ...grpc.CallOption) (*live.ListMeetingMessagesRes, error) {
	return &live.ListMeetingMessagesRes{}, nil
}

func newTestSvcCtx(fake *fakeLiveRpcCli) *svc.ServiceContext {
	return &svc.ServiceContext{
		Config: config.Config{
			LiveKit: struct {
				WebhookKey string
			}{WebhookKey: "secret"},
		},
		LiveRpcCli: fake,
	}
}

// newWebhookRequest 构造带指定 Authorization 的 POST 请求（body 为需校验原文）。
func newWebhookRequest(body string, token string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/webhook/livekit", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/webhook+json")
	req.Header.Set("Authorization", token)
	return req
}
