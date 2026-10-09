package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type StartMeetingRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartMeetingRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartMeetingRecordLogic {
	return &StartMeetingRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 开始会议录制（房间合成录制，落库并返回录制任务）。
// 幂等：会议已有"录制中"记录时，与 LiveKit 对账后返回该记录，不重复发起 Egress。
func (l *StartMeetingRecordLogic) StartMeetingRecord(in *live.StartMeetingRecordReq) (*live.StartMeetingRecordRes, error) {
	if !l.svcCtx.Config.LiveKit.Record.Enabled {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "录制功能未启用")
	}
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束，无法录制")
	}
	// 仅会议创建者（主持人）可操作录制；系统创建（create_user 为空）放行
	if err := requireMeetingOperator(meeting, authctx.GetUserId(l.ctx)); err != nil {
		return nil, err
	}

	// 已有录制中记录：与 LiveKit 对账后幂等返回（对账可清理"孤儿"录制）
	if rec, err := l.activeRecording(in.MeetingNo); err == nil {
		return &live.StartMeetingRecordRes{Recording: toRecordingInfo(rec, l.svcCtx.Config.LiveKit.Record.PlayURLBase)}, nil
	} else if !errors.Is(err, svc.ErrRecordingNotFound) {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询录制记录失败")
	}

	// 分布式锁防并发启录（go-zero RedisLock，Lua 原子 + TTL 自动释放）
	lock := redis.NewRedisLock(l.svcCtx.Redis, redisMeetingLockPrefix+in.MeetingNo+":record")
	lock.SetExpire(recordingLockTTL)
	ok, err := lock.AcquireCtx(l.ctx)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "获取录制锁失败")
	}
	if !ok {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_REPEAT, "录制正在启动，请稍后重试")
	}
	defer lock.Release()

	// 二次检查：锁等待期间可能已启录；与首次检查一致，DB 错误必须阻断，避免起第二个
	if rec, err := l.activeRecording(in.MeetingNo); err == nil {
		return &live.StartMeetingRecordRes{Recording: toRecordingInfo(rec, l.svcCtx.Config.LiveKit.Record.PlayURLBase)}, nil
	} else if !errors.Is(err, svc.ErrRecordingNotFound) {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询录制记录失败")
	}

	outputDir := strings.TrimRight(l.svcCtx.Config.LiveKit.Record.OutputDir, "/")
	if outputDir == "" {
		outputDir = "/out"
	}
	req := &livekit.RoomCompositeEgressRequest{
		RoomName:  in.MeetingNo,
		AudioOnly: in.AudioOnly,
		FileOutputs: []*livekit.EncodedFileOutput{
			{
				FileType: livekit.EncodedFileType_MP4,
				Filepath: fmt.Sprintf("%s/{room_name}/{time}", outputDir),
			},
		},
	}
	// audio_only 不设置 layout/preset，避免强制走视频管线（官方约定）
	if !in.AudioOnly {
		layout := strings.TrimSpace(in.Layout)
		if layout == "" {
			layout = strings.TrimSpace(l.svcCtx.Config.LiveKit.Record.Layout)
		}
		req.Layout = layout
		req.Options = &livekit.RoomCompositeEgressRequest_Preset{Preset: livekit.EncodingOptionsPreset_H264_720P_30}
	}

	info, err := l.svcCtx.LiveKit.API().Egress().StartRoomCompositeEgress(l.ctx, req)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "发起录制失败")
	}

	operator := authctx.GetUserId(l.ctx)
	deptCode := authctx.GetDeptCode(l.ctx)
	rec := &gormmodel.LiveMeetingRecording{
		CreateUser: sql.NullString{String: operator, Valid: operator != ""},
		UpdateUser: sql.NullString{String: operator, Valid: operator != ""},
		DeptCode:   sql.NullString{String: deptCode, Valid: deptCode != ""},
		MeetingNo:  in.MeetingNo,
		EgressId:   info.GetEgressId(),
		RoomName:   info.GetRoomName(),
		Status:     int(info.GetStatus()),
		StartTime:  carbonx.NowStartOfSecond().StdTime(),
	}
	if rec.RoomName == "" {
		rec.RoomName = in.MeetingNo
	}
	// 保存开始记录：按 egress_id 幂等 upsert，绝不回退终态，并处理 webhook 抢先建行的竞态
	saved, err := l.svcCtx.MeetingRepo.SaveRecordingStarted(l.ctx, rec)
	if err != nil {
		// 落库失败：停止已发起的 Egress，避免产生无主录制
		if _, serr := l.svcCtx.LiveKit.API().Egress().StopEgress(l.ctx, &livekit.StopEgressRequest{EgressId: info.GetEgressId()}); serr != nil {
			l.Logger.Errorf("orphan egress stop failed: meeting=%s egress=%s err=%v", in.MeetingNo, info.GetEgressId(), serr)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "录制落库失败")
	}

	l.Logger.Infof("meeting record started: meeting=%s egress=%s", in.MeetingNo, info.GetEgressId())
	return &live.StartMeetingRecordRes{Recording: toRecordingInfo(saved, l.svcCtx.Config.LiveKit.Record.PlayURLBase)}, nil
}

// activeRecording 查询会议进行中的录制，并与 LiveKit Egress 对账。
// 返回记录表示该会议已有进行中录制（幂等返回，且保证"每会议同时只有一个"）；
// 返回 ErrRecordingNotFound 表示可新建；对账失败时保守返回现有记录，避免重复录制。
func (l *StartMeetingRecordLogic) activeRecording(meetingNo string) (*gormmodel.LiveMeetingRecording, error) {
	rec, err := l.svcCtx.MeetingRepo.GetActiveRecordingByMeeting(l.ctx, meetingNo)
	if err != nil {
		return nil, err
	}
	active, rerr := reconcileRecordingState(l.ctx, l.svcCtx, rec)
	if rerr != nil {
		l.Logger.Errorf("reconcile recording failed: egress=%s err=%v", rec.EgressId, rerr)
		return rec, nil // 对账失败：保守阻断，避免起第二个
	}
	if active {
		// 仍在录制/收尾：视为进行中，阻塞新录制（保证同时只有一个）
		return rec, nil
	}
	// 已落终态（孤儿已对账）：允许新建
	return nil, svc.ErrRecordingNotFound
}
