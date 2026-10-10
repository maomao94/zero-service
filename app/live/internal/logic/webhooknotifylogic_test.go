package logic

import (
	"context"
	"testing"
	"time"

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

func TestWebhookEgressEndedCompleted(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	start, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	egressID := start.GetRecording().GetEgressId()

	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	location := "/out/" + meetingNo + "/rec.mp4"
	data := mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-E1", Event: "egress_ended",
		EgressInfo: &livekit.EgressInfo{
			EgressId: egressID,
			RoomName: meetingNo,
			Status:   livekit.EgressStatus_EGRESS_COMPLETE,
			EndedAt:  time.Now().UnixNano(),
			FileResults: []*livekit.FileInfo{
				{Filename: location, Location: location, Size: 1234, Duration: 60},
			},
		},
	})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: data}); err != nil {
		t.Fatalf("webhook error = %v", err)
	}
	rec, err := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != gormmodel.RecordingStatusComplete {
		t.Fatalf("status = %d, want completed", rec.Status)
	}
	if rec.FileName != meetingNo+"/rec.mp4" || rec.FileSize != 1234 || rec.Duration != 60 {
		t.Fatalf("recording file info = %+v", rec)
	}

	// 重放幂等：状态不变
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: data}); err != nil {
		t.Fatalf("replay error = %v", err)
	}
	rec2, _ := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if rec2.Status != gormmodel.RecordingStatusComplete {
		t.Fatalf("replay changed status: %+v", rec2)
	}
}

func TestWebhookEgressEndedFailed(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	start, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	egressID := start.GetRecording().GetEgressId()

	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	data := mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-E2", Event: "egress_ended",
		EgressInfo: &livekit.EgressInfo{
			EgressId: egressID,
			RoomName: meetingNo,
			Status:   livekit.EgressStatus_EGRESS_FAILED,
			Error:    "boom",
			EndedAt:  time.Now().UnixNano(),
		},
	})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: data}); err != nil {
		t.Fatalf("webhook error = %v", err)
	}
	rec, err := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != gormmodel.RecordingStatusFailed || rec.Error != "boom" {
		t.Fatalf("recording = %+v", rec)
	}
}

// TestWebhookEgressStartedAfterEndedDoesNotRegress 验证迟到的 egress_started/非终态事件
// 不会把已完成记录回退为"录制中"（Blocker：状态只允许 录制中→终态 单向流转）。
func TestWebhookEgressStartedAfterEndedDoesNotRegress(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	start, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	egressID := start.GetRecording().GetEgressId()

	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	ended := mustWebhookData(t, &livekit.WebhookEvent{Id: "EVT-END", Event: "egress_ended", EgressInfo: &livekit.EgressInfo{
		EgressId: egressID, RoomName: meetingNo, Status: livekit.EgressStatus_EGRESS_COMPLETE,
	}})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: ended}); err != nil {
		t.Fatal(err)
	}

	// 迟到/重放的 egress_started 不得回退终态
	started := mustWebhookData(t, &livekit.WebhookEvent{Id: "EVT-START", Event: "egress_started", EgressInfo: &livekit.EgressInfo{
		EgressId: egressID, RoomName: meetingNo, Status: livekit.EgressStatus_EGRESS_ACTIVE,
	}})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: started}); err != nil {
		t.Fatal(err)
	}
	rec, _ := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if rec.Status != gormmodel.RecordingStatusComplete {
		t.Fatalf("terminal status regressed: %+v", rec)
	}
}

// TestWebhookEgressUpdatedThenEndedBackfillsFile 验证 egress_updated 先标记完成（无文件信息）后，
// egress_ended 能补全文件字段。
func TestWebhookEgressUpdatedThenEndedBackfillsFile(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	start, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	egressID := start.GetRecording().GetEgressId()
	wh := NewWebhookNotifyLogic(ctx, svcCtx)

	updated := mustWebhookData(t, &livekit.WebhookEvent{Id: "EVT-UPD", Event: "egress_updated", EgressInfo: &livekit.EgressInfo{
		EgressId: egressID, RoomName: meetingNo, Status: livekit.EgressStatus_EGRESS_COMPLETE,
	}})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: updated}); err != nil {
		t.Fatal(err)
	}
	rec, _ := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if rec.Status != gormmodel.RecordingStatusComplete || rec.FileName != "" {
		t.Fatalf("expected completed without file, got %+v", rec)
	}

	location := "/out/" + meetingNo + "/rec.mp4"
	ended := mustWebhookData(t, &livekit.WebhookEvent{Id: "EVT-END2", Event: "egress_ended", EgressInfo: &livekit.EgressInfo{
		EgressId: egressID, RoomName: meetingNo, Status: livekit.EgressStatus_EGRESS_COMPLETE,
		EndedAt:     time.Now().UnixNano(),
		FileResults: []*livekit.FileInfo{{Filename: location, Location: location, Size: 999, Duration: 30}},
	}})
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: ended}); err != nil {
		t.Fatal(err)
	}
	rec, _ = svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, egressID)
	if rec.FileName != meetingNo+"/rec.mp4" || rec.FileSize != 999 || rec.Duration != 30 {
		t.Fatalf("file info not backfilled: %+v", rec)
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
