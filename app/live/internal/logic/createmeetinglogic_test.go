package logic

import (
	"context"
	"regexp"
	"testing"
	"time"

	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

func TestCreateMeetingSuccess(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	// 创建人 + 机构注入（模拟网关注入 gRPC metadata）
	baseCtx := authctx.WithUserID(context.Background(), "alice")
	ctx := authctx.WithDeptCode(baseCtx, "d01")
	l := NewCreateMeetingLogic(ctx, svcCtx)

	resp, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "晨会"})
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	// IdUtil 会议号格式：M + 4 位年 + MMddHHmmss + 4 位序号
	if !regexp.MustCompile(`^M\d{18}$`).MatchString(resp.Meeting.MeetingNo) {
		t.Fatalf("unexpected meeting no: %s", resp.Meeting.MeetingNo)
	}
	if resp.Meeting.Status != int32(gormmodel.MeetingStatusActive) || resp.Meeting.Title != "晨会" {
		t.Fatalf("unexpected meeting: %+v", resp)
	}
	if resp.Meeting.CreateUser != "alice" || resp.Meeting.DeptCode != "d01" {
		t.Fatalf("create user/dept missing: %+v", resp.Meeting)
	}
	// 房间已创建
	if len(mock.createdRooms) != 1 || mock.createdRooms[0] != resp.Meeting.MeetingNo {
		t.Fatalf("rooms = %v", mock.createdRooms)
	}
	// 落库
	got, err := svcCtx.MeetingRepo.GetMeeting(context.Background(), resp.Meeting.MeetingNo)
	if err != nil {
		t.Fatalf("meeting not persisted: %v", err)
	}
	if got.Title != "晨会" || !got.CreateUser.Valid || got.CreateUser.String != "alice" ||
		!got.UpdateUser.Valid || got.UpdateUser.String != "alice" ||
		!got.DeptCode.Valid || got.DeptCode.String != "d01" {
		t.Fatalf("unexpected persisted meeting: %+v", got)
	}
}

func TestCreateMeetingInvalidParams(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	l := NewCreateMeetingLogic(authctx.WithUserID(context.Background(), "alice"), svcCtx)
	_, err := l.CreateMeeting(&live.CreateMeetingReq{Title: ""})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_01_PARAM_INVALID) {
		t.Fatalf("want PARAM_INVALID error, got %v", err)
	}
}

func TestCreateMeetingRoomFailedRollback(t *testing.T) {
	mock := &liveKitMock{createErr: errRoom}
	svcCtx := newTestSvcCtx(t, mock)
	l := NewCreateMeetingLogic(authctx.WithUserID(context.Background(), "alice"), svcCtx)

	_, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err == nil {
		t.Fatal("want error")
	}
	if len(mock.createdRooms) != 0 {
		t.Fatalf("no room should be created: %v", mock.createdRooms)
	}
}

func TestJoinMeetingIssuesTokenAndUpsertsParticipant(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	// 先创建会议
	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}

	join := NewJoinMeetingLogic(ctx, svcCtx)
	resp, err := join.JoinMeeting(&live.JoinMeetingReq{MeetingNo: meeting.Meeting.MeetingNo, Identity: "bob", Name: "Bob"})
	if err != nil {
		t.Fatalf("join error = %v", err)
	}
	if resp.Token == "" || resp.WsUrl != "ws://127.0.0.1:7880" {
		t.Fatalf("unexpected join resp: %+v", resp)
	}
	// 参会记录已落库（create_user 记录加入者）
	ps, err := svcCtx.MeetingRepo.ListParticipants(ctx, meeting.Meeting.MeetingNo)
	if err != nil || len(ps) != 1 || ps[0].Identity != "bob" {
		t.Fatalf("participants = %v err=%v", ps, err)
	}
	if !ps[0].CreateUser.Valid || ps[0].CreateUser.String != "alice" ||
		!ps[0].UpdateUser.Valid || ps[0].UpdateUser.String != "alice" {
		t.Fatalf("participant create/update user not recorded: %+v", ps[0].CreateUser)
	}
}

func TestJoinMeetingEndedRejected(t *testing.T) {
	mock := &liveKitMock{}
	svcCtx := newTestSvcCtx(t, mock)
	ctx := authctx.WithUserID(context.Background(), "alice")

	l := NewCreateMeetingLogic(ctx, svcCtx)
	meeting, err := l.CreateMeeting(&live.CreateMeetingReq{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	// 直接置为已结束
	if _, err := svcCtx.MeetingRepo.UpdateMeetingEnded(ctx, meeting.Meeting.MeetingNo, time.Now(), "alice", "d01"); err != nil {
		t.Fatal(err)
	}
	join := NewJoinMeetingLogic(ctx, svcCtx)
	_, err = join.JoinMeeting(&live.JoinMeetingReq{MeetingNo: meeting.Meeting.MeetingNo, Identity: "bob"})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_05_BIZ_STATE) {
		t.Fatalf("want BIZ_STATE error, got %v", err)
	}
}

func TestJoinMeetingNotFound(t *testing.T) {
	svcCtx := newTestSvcCtx(t, &liveKitMock{})
	join := NewJoinMeetingLogic(context.Background(), svcCtx)
	_, err := join.JoinMeeting(&live.JoinMeetingReq{MeetingNo: "NOPE", Identity: "bob"})
	if !tool.IsErrorByPbCode(err, extproto.Code__1_02_RECORD_NOT_EXIST) {
		t.Fatalf("want RECORD_NOT_EXIST error, got %v", err)
	}
}
