package svc

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"zero-service/app/live/model/gormmodel"
	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMeetingRepoCreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()
	now := time.Now()

	m := &gormmodel.LiveMeeting{
		MeetingNo:  "M001",
		Title:      "测试会议",
		Status:     gormmodel.MeetingStatusActive,
		CreateUser: sql.NullString{String: "alice", Valid: true},
		StartTime:  now,
	}
	if err := repo.CreateMeeting(ctx, m); err != nil {
		t.Fatalf("create error = %v", err)
	}
	got, err := repo.GetMeeting(ctx, "M001")
	if err != nil {
		t.Fatalf("get error = %v", err)
	}
	if got.Title != "测试会议" || got.Status != gormmodel.MeetingStatusActive {
		t.Fatalf("unexpected meeting: %+v", got)
	}
	if _, err := repo.GetMeeting(ctx, "NOPE"); !errors.Is(err, ErrMeetingNotFound) {
		t.Fatalf("want ErrMeetingNotFound, got %v", err)
	}
}

func TestMeetingRepoUpdateMeetingEnded(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()

	m := &gormmodel.LiveMeeting{
		MeetingNo:  "M002",
		Title:      "t",
		Status:     gormmodel.MeetingStatusActive,
		CreateUser: sql.NullString{String: "alice", Valid: true},
		StartTime:  time.Now(),
	}
	if err := repo.CreateMeeting(ctx, m); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.UpdateMeetingEnded(ctx, "M002", time.Now(), "alice", "d01")
	if err != nil || !updated {
		t.Fatalf("first end: updated=%v err=%v", updated, err)
	}
	// 幂等：再次结束不更新
	updated, err = repo.UpdateMeetingEnded(ctx, "M002", time.Now(), "alice", "d01")
	if err != nil || updated {
		t.Fatalf("second end: updated=%v err=%v", updated, err)
	}
	got, _ := repo.GetMeeting(ctx, "M002")
	if got.Status != gormmodel.MeetingStatusEnded || !got.EndTime.Valid {
		t.Fatalf("unexpected ended meeting: %+v", got)
	}
}

func TestMeetingRepoUpsertParticipantKeepsFirstJoinTime(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()
	first := time.Now().Add(-time.Hour)

	p1 := &gormmodel.LiveMeetingParticipant{
		MeetingNo: "M001",
		Identity:  "bob",
		Name:      "bob-old",
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  first,
	}
	if err := repo.UpsertParticipant(ctx, p1); err != nil {
		t.Fatal(err)
	}
	p2 := &gormmodel.LiveMeetingParticipant{
		MeetingNo: "M001",
		Identity:  "bob",
		Name:      "bob-new",
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  time.Now(),
	}
	if err := repo.UpsertParticipant(ctx, p2); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListParticipants(ctx, "M001", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 participant, got %d", len(list))
	}
	got := list[0]
	if got.Name != "bob-new" {
		t.Fatalf("name not updated: %q", got.Name)
	}
	if !got.JoinTime.Equal(first) {
		t.Fatalf("join_time should keep first value: got %v want %v", got.JoinTime, first)
	}
}

func TestMeetingRepoMarkParticipantLeft(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()

	p := &gormmodel.LiveMeetingParticipant{
		MeetingNo: "M001",
		Identity:  "bob",
		Name:      "bob",
		Status:    gormmodel.ParticipantStatusJoined,
		JoinTime:  time.Now(),
	}
	if err := repo.UpsertParticipant(ctx, p); err != nil {
		t.Fatal(err)
	}
	leftAt := time.Now()
	if err := repo.MarkParticipantLeft(ctx, "M001", "bob", leftAt); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListParticipants(ctx, "M001", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := list[0]
	if got.Status != gormmodel.ParticipantStatusLeft || !got.LeftTime.Valid {
		t.Fatalf("unexpected participant: %+v", got)
	}
}

func TestMeetingRepoListMeetingsPaged(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		m := &gormmodel.LiveMeeting{
			MeetingNo:  "M" + string(rune('A'+i)),
			Title:      "t",
			Status:     gormmodel.MeetingStatusActive,
			CreateUser: sql.NullString{String: "alice", Valid: true},
			StartTime:  time.Now(),
		}
		if err := repo.CreateMeeting(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	meetings, total, err := repo.ListMeetings(ctx, &MeetingListQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(meetings) != 2 {
		t.Fatalf("total=%d len=%d", total, len(meetings))
	}
}

func TestMeetingRepoListParticipantsExcludeKinds(t *testing.T) {
	db := openTestDB(t)
	repo := NewMeetingRepo(db)
	ctx := context.Background()
	now := time.Now()
	for _, p := range []*gormmodel.LiveMeetingParticipant{
		{MeetingNo: "M001", Identity: "bob", Name: "bob", Status: gormmodel.ParticipantStatusJoined, Kind: gormmodel.ParticipantKindStandard, JoinTime: now},
		{MeetingNo: "M001", Identity: "EG_1", Name: "egress", Status: gormmodel.ParticipantStatusJoined, Kind: gormmodel.ParticipantKindEgress, JoinTime: now},
	} {
		if err := repo.UpsertParticipant(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	all, err := repo.ListParticipants(ctx, "M001", nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("want 2 participants, got %d err=%v", len(all), err)
	}
	filtered, err := repo.ListParticipants(ctx, "M001", []int{gormmodel.ParticipantKindEgress})
	if err != nil || len(filtered) != 1 || filtered[0].Identity != "bob" {
		t.Fatalf("exclude egress failed: %+v err=%v", filtered, err)
	}
}

func TestMeetingRepoListMeetingsHasRecordingAndCount(t *testing.T) {
	db := openTestDB(t)
	if err := db.AutoMigrate(&gormmodel.LiveMeetingRecording{}); err != nil {
		t.Fatal(err)
	}
	repo := NewMeetingRepo(db)
	ctx := context.Background()
	for _, no := range []string{"M1", "M2"} {
		m := &gormmodel.LiveMeeting{MeetingNo: no, Title: "t", Status: gormmodel.MeetingStatusActive, StartTime: time.Now()}
		if err := repo.CreateMeeting(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.WithContext(ctx).Create(&gormmodel.LiveMeetingRecording{
		MeetingNo: "M1", EgressId: "EG_1", RoomName: "M1", Status: gormmodel.RecordingStatusComplete, StartTime: time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	withRec, total, err := repo.ListMeetings(ctx, &MeetingListQuery{Page: 1, PageSize: 10, HasRecording: 1})
	if err != nil || total != 1 || len(withRec) != 1 || withRec[0].MeetingNo != "M1" {
		t.Fatalf("has recording filter failed: total=%d meetings=%v err=%v", total, withRec, err)
	}
	noRec, total, err := repo.ListMeetings(ctx, &MeetingListQuery{Page: 1, PageSize: 10, HasRecording: 2})
	if err != nil || total != 1 || len(noRec) != 1 || noRec[0].MeetingNo != "M2" {
		t.Fatalf("no recording filter failed: total=%d meetings=%v err=%v", total, noRec, err)
	}
	counts, err := repo.CountRecordingsByMeetings(ctx, []string{"M1", "M2"})
	if err != nil || counts["M1"] != 1 || counts["M2"] != 0 {
		t.Fatalf("counts=%v err=%v", counts, err)
	}
}

// openTestDB 打开内存 sqlite 并迁移模型。
func openTestDB(t *testing.T) *gormx.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_loc=auto"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}
	if err := db.AutoMigrate(&gormmodel.LiveMeeting{}, &gormmodel.LiveMeetingParticipant{}); err != nil {
		t.Fatalf("auto migrate error = %v", err)
	}
	return &gormx.DB{DB: db}
}
