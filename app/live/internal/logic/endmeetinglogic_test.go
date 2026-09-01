package logic

import (
	"context"
	"errors"
	"testing"

	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

var errRoom = errors.New("room error")

func TestEndMeetingFlow(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	// 有参会者
	p := &gormmodel.LiveMeetingParticipant{
		MeetingNo: meeting.Meeting.MeetingNo, Identity: "bob", Name: "bob",
		Status: gormmodel.ParticipantStatusJoined,
	}
	if err := svcCtx.MeetingRepo.UpsertParticipant(ctx, p); err != nil {
		t.Fatal(err)
	}

	end := NewEndMeetingLogic(ctx, svcCtx)
	if _, err := end.EndMeeting(&live.EndMeetingReq{MeetingNo: meeting.Meeting.MeetingNo}); err != nil {
		t.Fatalf("end error = %v", err)
	}
	// 房间删除、会议 ended、参与者 left
	if len(mock.deletedRooms) != 1 || mock.deletedRooms[0] != meeting.Meeting.MeetingNo {
		t.Fatalf("deleted rooms = %v", mock.deletedRooms)
	}
	got, _ := svcCtx.MeetingRepo.GetMeeting(ctx, meeting.Meeting.MeetingNo)
	if got.Status != gormmodel.MeetingStatusEnded || !got.EndTime.Valid {
		t.Fatalf("meeting not ended: %+v", got)
	}
	// 结束操作人/机构已记录（来自 gRPC metadata）
	if !got.UpdateUser.Valid || got.UpdateUser.String != "alice" {
		t.Fatalf("update user not recorded: %+v", got.UpdateUser)
	}
	ps, _ := svcCtx.MeetingRepo.ListParticipants(ctx, meeting.Meeting.MeetingNo)
	if ps[0].Status != gormmodel.ParticipantStatusLeft || !ps[0].LeftTime.Valid {
		t.Fatalf("participant not left: %+v", ps[0])
	}
}

func TestEndMeetingIdempotent(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	end := NewEndMeetingLogic(ctx, svcCtx)
	if _, err := end.EndMeeting(&live.EndMeetingReq{MeetingNo: meeting.Meeting.MeetingNo}); err != nil {
		t.Fatal(err)
	}
	// 第二次结束：不再调用 DeleteRoom
	if _, err := end.EndMeeting(&live.EndMeetingReq{MeetingNo: meeting.Meeting.MeetingNo}); err != nil {
		t.Fatalf("second end error = %v", err)
	}
	if len(mock.deletedRooms) != 1 {
		t.Fatalf("DeleteRoom should be called once, got %d", len(mock.deletedRooms))
	}
}

func TestEndMeetingLockContention(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	// 用真实 RedisLock 预占用锁（模拟并发结束正在进行）
	pre := redis.NewRedisLock(svcCtx.Redis, redisEndLockPrefix+meeting.Meeting.MeetingNo+":end")
	pre.SetExpire(endLockTTL)
	ok, err := pre.AcquireCtx(ctx)
	if err != nil || !ok {
		t.Fatalf("pre-acquire lock: ok=%v err=%v", ok, err)
	}

	end := NewEndMeetingLogic(ctx, svcCtx)
	_, err = end.EndMeeting(&live.EndMeetingReq{MeetingNo: meeting.Meeting.MeetingNo})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_05_BIZ_REPEAT) {
		t.Fatalf("want BIZ_REPEAT, got %v", err)
	}
	// 锁被占用时不应删除房间
	if len(mock.deletedRooms) != 0 {
		t.Fatalf("room should not be deleted under lock contention: %v", mock.deletedRooms)
	}
}

func TestEndMeetingMissing(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	end := NewEndMeetingLogic(context.Background(), svcCtx)
	_, err := end.EndMeeting(&live.EndMeetingReq{MeetingNo: "NOPE"})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("want RECORD_NOT_EXIST, got %v", err)
	}
}
