package logic

import (
	"context"
	"testing"

	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"google.golang.org/protobuf/proto"
)

// mustWebhookData 把 WebhookEvent 序列化为请求字节。
func mustWebhookData(t *testing.T, event *livekit.WebhookEvent) []byte {
	t.Helper()
	data, err := proto.Marshal(event)
	if err != nil {
		t.Fatalf("marshal webhook event error = %v", err)
	}
	return data
}

// TestWebhookReplayConsistent 验证事件重放（重复投递/补发）结果一致：
// 重复处理不报错，且会议状态保持 ended（闭环对账可安全重放）。
func TestWebhookReplayConsistent(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	data := mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-1", Event: "room_finished", Room: &livekit.Room{Name: meeting.Meeting.MeetingNo},
	})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: data}); err != nil {
		t.Fatalf("first notify error = %v", err)
	}
	// 重复/补发投递：不报错，状态不变
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: data}); err != nil {
		t.Fatalf("replay notify error = %v", err)
	}
	got, _ := svcCtx.MeetingRepo.GetMeeting(ctx, meeting.Meeting.MeetingNo)
	if got.Status != gormmodel.MeetingStatusEnded {
		t.Fatalf("meeting should be ended: %+v", got)
	}
}

func TestWebhookRoomFinished(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-RF", Event: "room_finished", Room: &livekit.Room{Name: meeting.Meeting.MeetingNo},
	})}); err != nil {
		t.Fatal(err)
	}
	got, _ := svcCtx.MeetingRepo.GetMeeting(ctx, meeting.Meeting.MeetingNo)
	if got.Status != gormmodel.MeetingStatusEnded {
		t.Fatalf("meeting should be ended: %+v", got)
	}
}

func TestWebhookParticipantJoinedAndLeft(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	room := &livekit.Room{Name: meeting.Meeting.MeetingNo}

	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-J1", Event: "participant_joined", Room: room,
		Participant: &livekit.ParticipantInfo{Identity: "bob", Name: "Bob"},
	})}); err != nil {
		t.Fatal(err)
	}
	ps, err := svcCtx.MeetingRepo.ListParticipants(ctx, meeting.Meeting.MeetingNo)
	if err != nil || len(ps) != 1 || ps[0].Status != gormmodel.ParticipantStatusJoined {
		t.Fatalf("participants = %v err=%v", ps, err)
	}

	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-L1", Event: "participant_left", Room: room,
		Participant: &livekit.ParticipantInfo{Identity: "bob"},
	})}); err != nil {
		t.Fatal(err)
	}
	ps, _ = svcCtx.MeetingRepo.ListParticipants(ctx, meeting.Meeting.MeetingNo)
	if ps[0].Status != gormmodel.ParticipantStatusLeft || !ps[0].LeftTime.Valid {
		t.Fatalf("participant should be left: %+v", ps[0])
	}
}

func TestWebhookUnknownEventIgnored(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	wh := NewWebhookNotifyLogic(context.Background(), svcCtx)
	_, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-U", Event: "track_published", Room: &livekit.Room{Name: "whatever"},
	})})
	if err != nil {
		t.Fatalf("unknown event should be ignored, got %v", err)
	}
}

func TestWebhookInvalidDataRejected(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	wh := NewWebhookNotifyLogic(context.Background(), svcCtx)
	// 空数据
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{}); !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("want PARAM_INVALID for empty data, got %v", err)
	}
	// 非法字节
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: []byte("garbage")}); !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("want PARAM_INVALID for garbage data, got %v", err)
	}
}
