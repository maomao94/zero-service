package logic

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
)

// Redis key 前缀（统一 live: 区分业务域）。
const (
	// redisMeetingLockPrefix 会议相关分布式锁 key 前缀（配合 go-zero RedisLock）
	// 使用方式：redisMeetingLockPrefix + meetingNo
	redisMeetingLockPrefix = "live:lock:meeting:"
	// redisMeetingCodeLockPrefix 会议号生成分布式锁 key（防并发生成重复 meeting_code）
	redisMeetingCodeLockPrefix = "live:lock:meeting_code_gen"
)

// meetingLockTTL 会议相关分布式锁持有时间（超过视为持锁方崩溃，自动释放）。
const meetingLockTTL = 10

// meetingCodeLockTTL 会议号生成锁持有时间（5秒足够生成+校验唯一性）。
const meetingCodeLockTTL = 5

// recordingLockTTL 启录锁持有时间（覆盖 StartRoomCompositeEgress + 落库，留足余量）。
const recordingLockTTL = 30

// requireMeetingNo 校验会议号非空。
func requireMeetingNo(meetingNo string) error {
	if strings.TrimSpace(meetingNo) == "" {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号不能为空")
	}
	return nil
}

// requireMeetingIdentity 校验会议号与身份非空。
func requireMeetingIdentity(meetingNo, identity string) error {
	if strings.TrimSpace(meetingNo) == "" || strings.TrimSpace(identity) == "" {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号与身份不能为空")
	}
	return nil
}

// requireMeetingOperator 校验调用者为会议创建者（主持人）。
// 系统创建（create_user 为空）或未鉴权身份（operator 为空）时放行，避免误伤系统会议。
func requireMeetingOperator(meeting *gormmodel.LiveMeeting, operator string) error {
	if meeting == nil {
		return nil
	}
	owner := strings.TrimSpace(meeting.CreateUser.String)
	if owner != "" && operator != "" && operator != owner {
		return tool.NewErrorByPbCode(extproto.Code__1_03_UNAUTHORIZED, "仅会议创建者可操作录制")
	}
	return nil
}

// toMeetingInfo 转换会议单据为 RPC 视图（时间用 carbon 格式化输出）。
func toMeetingInfo(m *gormmodel.LiveMeeting) *live.MeetingInfo {
	return &live.MeetingInfo{
		MeetingNo:        m.MeetingNo,
		MeetingCode:      m.MeetingCode,
		Title:            m.Title,
		Status:           int32(m.Status),
		CreateUser:       m.CreateUser.String,
		UpdateUser:       m.UpdateUser.String,
		DeptCode:         m.DeptCode.String,
		CreateTime:       carbonx.FormatDateTimeOrEmpty(m.CreateTime),
		StartTime:        carbonx.FormatDateTimeOrEmpty(m.StartTime),
		EndTime:          carbonx.FormatNullDateTime(m.EndTime),
		EmptyTimeout:     uint32(m.EmptyTimeout),
		DepartureTimeout: uint32(m.DepartureTimeout),
		MaxParticipants:  uint32(m.MaxParticipants),
		RoomSid:          m.RoomSid,
		Metadata:         m.Metadata,
	}
}

// reconcileRecordingState 与 LiveKit 对账单条录制记录，把 DB 状态刷新为 Egress 实际状态。
// 返回 active 表示仍进行中（STARTING/ACTIVE/ENDING）；false 表示已落终态。
// ListEgress 失败时返回 (true, err)，调用方按"进行中"保守处理，避免误起第二个。
func reconcileRecordingState(ctx context.Context, svcCtx *svc.ServiceContext, rec *gormmodel.LiveMeetingRecording) (bool, error) {
	resp, err := svcCtx.LiveKit.API().Egress().ListEgress(ctx, &livekit.ListEgressRequest{EgressId: rec.EgressId})
	if err != nil {
		return true, err
	}
	var found *livekit.EgressInfo
	for _, info := range resp.GetItems() {
		if info.GetEgressId() == rec.EgressId {
			found = info
			break
		}
	}
	// Egress 已不在（结束事件丢失/被清理）：落失败
	if found == nil {
		if _, err := svcCtx.MeetingRepo.UpdateRecordingStatus(ctx, rec.EgressId, gormmodel.RecordingStatusFailed,
			svc.RecordingResult{EndedAt: carbonx.NowStartOfSecond().StdTime(), Error: "Egress 已结束但未收到结束事件，自动标记失败"}); err != nil {
			return false, err
		}
		return false, nil
	}
	status := int(found.GetStatus())
	if gormmodel.RecordingStatusIsActive(status) {
		// 进行中：同步状态（如 STARTING→ACTIVE/ENDING），供 UI/排查看到实时值
		if status != rec.Status {
			if _, err := svcCtx.MeetingRepo.UpdateRecordingStatus(ctx, rec.EgressId, status, svc.RecordingResult{}); err != nil {
				return true, err
			}
			rec.Status = status
		}
		return true, nil
	}
	// 终态：落库（COMPLETE 回填文件信息；其余记录错误原因）
	result := egressFileResult(found, svcCtx.Config.LiveKit.Record.OutputDir)
	if status != gormmodel.RecordingStatusComplete {
		result.Error = found.GetError()
		if result.Error == "" {
			result.Error = found.GetDetails()
		}
		if result.Error == "" {
			result.Error = "egress ended: " + found.GetStatus().String()
		}
	}
	if _, err := svcCtx.MeetingRepo.UpdateRecordingStatus(ctx, rec.EgressId, status, result); err != nil {
		return false, err
	}
	return false, nil
}

// egressFileResult 从 EgressInfo 提取文件信息与结束时间（相对文件名用于拼接播放地址）。
func egressFileResult(info *livekit.EgressInfo, outputDir string) svc.RecordingResult {
	result := svc.RecordingResult{EndedAt: carbonx.NowStartOfSecond().StdTime()}
	if info.GetEndedAt() > 0 {
		result.EndedAt = time.Unix(0, info.GetEndedAt())
	}
	if results := info.GetFileResults(); len(results) > 0 {
		f := results[0]
		location := f.GetLocation()
		if location == "" {
			location = f.GetFilename()
		}
		result.FilePath = location
		result.FileName = relativeRecordingFile(location, outputDir)
		result.FileSize = f.GetSize()
		result.Duration = f.GetDuration()
	}
	return result
}

// relativeRecordingFile 把 Egress 返回的位置转为相对输出根目录的文件名（用于拼接播放地址）；
// 输出目录不匹配时只取 basename，避免把内部目录结构透出给客户端。
func relativeRecordingFile(location, outputDir string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	dir := strings.TrimRight(outputDir, "/")
	if dir != "" {
		if rel := strings.TrimPrefix(location, dir+"/"); rel != location {
			return rel
		}
	}
	return path.Base(location)
}

// recordingErr 把录制 repo 错误映射为 extproto 业务错误码。
func recordingErr(err error) error {
	if errors.Is(err, svc.ErrRecordingNotFound) {
		return tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "录制记录不存在")
	}
	return tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询录制记录失败")
}

// toRecordingInfo 转换录制记录为 RPC 视图。
// fileUrl 仅在录制已产出文件（COMPLETE / LIMIT_REACHED 且有文件名）且配置了播放基址时返回，
// 使用播放基址拼接相对路径，不对外返回本地绝对路径（对齐接口设计规范）。
func toRecordingInfo(rec *gormmodel.LiveMeetingRecording, playURLBase string) *live.MeetingRecordingInfo {
	info := &live.MeetingRecordingInfo{
		RecordId:  rec.Id,
		MeetingNo: rec.MeetingNo,
		EgressId:  rec.EgressId,
		Status:    int32(rec.Status),
		FileName:  rec.FileName,
		FileSize:  rec.FileSize,
		Duration:  rec.Duration,
		StartTime: carbonx.FormatDateTimeOrEmpty(rec.StartTime),
		EndTime:   carbonx.FormatNullDateTime(rec.EndTime),
		Error:     rec.Error,
	}
	if (rec.Status == gormmodel.RecordingStatusComplete || rec.Status == gormmodel.RecordingStatusLimitReached) &&
		rec.FileName != "" && strings.TrimSpace(playURLBase) != "" {
		info.FileUrl = strings.TrimRight(playURLBase, "/") + "/" + strings.TrimLeft(rec.FileName, "/")
	}
	return info
}
