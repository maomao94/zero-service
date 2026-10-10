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
)

func TestStartMeetingRecordIdempotent(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	meeting, err := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	meetingNo := meeting.GetMeeting().GetMeetingNo()

	resp, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatalf("start record error = %v", err)
	}
	if resp.GetRecording().GetStatus() != int32(gormmodel.RecordingStatusActive) {
		t.Fatalf("status = %d, want active", resp.GetRecording().GetStatus())
	}
	if len(mock.egressStarted) != 1 {
		t.Fatalf("egress started = %v, want 1", mock.egressStarted)
	}

	// 幂等：再次启录返回同一记录，不重复发起 Egress
	resp2, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	if resp2.GetRecording().GetRecordId() != resp.GetRecording().GetRecordId() {
		t.Fatalf("idempotent record id mismatch: %s vs %s", resp2.GetRecording().GetRecordId(), resp.GetRecording().GetRecordId())
	}
	if len(mock.egressStarted) != 1 {
		t.Fatalf("egress should not start twice: %v", mock.egressStarted)
	}
}

func TestStartMeetingRecordMeetingNotFound(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	_, err := NewStartMeetingRecordLogic(context.Background(), svcCtx).
		StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: "M404"})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("want RECORD_NOT_EXIST, got %v", err)
	}
}

func TestStopMeetingRecord(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	if _, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStopMeetingRecordLogic(ctx, svcCtx).StopMeetingRecord(&live.StopMeetingRecordReq{MeetingNo: meetingNo}); err != nil {
		t.Fatalf("stop record error = %v", err)
	}
	if len(mock.egressStopped) != 1 {
		t.Fatalf("egress stopped = %v, want 1", mock.egressStopped)
	}
}

func TestStartMeetingRecordForbiddenForNonOwner(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	alice := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(alice, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})

	bob := authctx.WithUserID(context.Background(), "bob")
	_, err := NewStartMeetingRecordLogic(bob, svcCtx).
		StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meeting.GetMeeting().GetMeetingNo()})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_03_UNAUTHORIZED) {
		t.Fatalf("want UNAUTHORIZED for non-owner, got %v", err)
	}
}

func TestStopMeetingRecordIdleIsIdempotent(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if _, err := NewStopMeetingRecordLogic(ctx, svcCtx).
		StopMeetingRecord(&live.StopMeetingRecordReq{MeetingNo: meeting.GetMeeting().GetMeetingNo()}); err != nil {
		t.Fatalf("idle stop should succeed, got %v", err)
	}
}

func TestStartMeetingRecordReconcilesStaleRecording(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()

	first, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	firstEgress := first.GetRecording().GetEgressId()

	// 模拟 Egress 已消失但未收到结束事件（孤儿记录）
	mock.egressInactive = true
	second, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatalf("reconcile start error = %v", err)
	}
	if second.GetRecording().GetEgressId() == firstEgress {
		t.Fatalf("expected a new egress after reconcile, still %s", firstEgress)
	}
	old, _ := svcCtx.MeetingRepo.GetRecordingByEgressID(ctx, firstEgress)
	if old.Status != gormmodel.RecordingStatusFailed {
		t.Fatalf("stale recording should be failed, got %+v", old)
	}
}

// TestStartMeetingRecordBlockedWhileEnding 验证"每会议同时只有一个录制"：
// 停止后 egress 处于 ENDING（收尾中）时不允许再起第二个，幂等返回现有记录。
func TestStartMeetingRecordBlockedWhileEnding(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()

	first, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewStopMeetingRecordLogic(ctx, svcCtx).StopMeetingRecord(&live.StopMeetingRecordReq{MeetingNo: meetingNo}); err != nil {
		t.Fatal(err)
	}
	// 收尾窗口内再次启录：应被阻塞，返回既有记录，不新建 egress
	mock.egressEnding = true
	second, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatalf("start during ending error = %v", err)
	}
	if second.GetRecording().GetEgressId() != first.GetRecording().GetEgressId() {
		t.Fatalf("should block a second egress during ending, got new %s", second.GetRecording().GetEgressId())
	}
	if len(mock.egressStarted) != 1 {
		t.Fatalf("egress started = %v, want 1", mock.egressStarted)
	}
}

// TestStartMeetingRecordAfterEndedAllowsNewRecording 验证上一次录制已完成（ended）后可重新录制。
func TestStartMeetingRecordAfterEndedAllowsNewRecording(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()

	first, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	wh := NewWebhookNotifyLogic(ctx, svcCtx)
	if _, err := wh.WebhookNotify(&live.WebhookNotifyReq{Data: mustWebhookData(t, &livekit.WebhookEvent{
		Id: "EVT-RST", Event: "egress_ended", EgressInfo: &livekit.EgressInfo{
			EgressId: first.GetRecording().GetEgressId(), RoomName: meetingNo, Status: livekit.EgressStatus_EGRESS_COMPLETE,
		},
	})}); err != nil {
		t.Fatal(err)
	}
	second, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatalf("restart after ended error = %v", err)
	}
	if second.GetRecording().GetEgressId() == first.GetRecording().GetEgressId() {
		t.Fatalf("expected a new recording after ended, still %s", first.GetRecording().GetEgressId())
	}
}

func TestGetMeetingRecordState(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")
	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()

	st, err := NewGetMeetingRecordStateLogic(ctx, svcCtx).GetMeetingRecordState(&live.GetMeetingRecordStateReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	if st.GetActive() || st.GetRecording() != nil {
		t.Fatalf("want inactive before start, got %+v", st)
	}

	if _, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo}); err != nil {
		t.Fatal(err)
	}
	st, err = NewGetMeetingRecordStateLogic(ctx, svcCtx).GetMeetingRecordState(&live.GetMeetingRecordStateReq{MeetingNo: meetingNo})
	if err != nil {
		t.Fatal(err)
	}
	if !st.GetActive() || st.GetRecording() == nil {
		t.Fatalf("want active after start, got %+v", st)
	}
}

func TestListMeetingRecordings(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	meeting, _ := NewCreateMeetingLogic(ctx, svcCtx).CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	meetingNo := meeting.GetMeeting().GetMeetingNo()
	if _, err := NewStartMeetingRecordLogic(ctx, svcCtx).StartMeetingRecord(&live.StartMeetingRecordReq{MeetingNo: meetingNo}); err != nil {
		t.Fatal(err)
	}
	resp, err := NewListMeetingRecordingsLogic(ctx, svcCtx).ListMeetingRecordings(&live.ListMeetingRecordingsReq{MeetingNo: meetingNo, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 1 || len(resp.GetRecordings()) != 1 {
		t.Fatalf("list recordings = %+v total=%d", resp.GetRecordings(), resp.GetTotal())
	}
}
