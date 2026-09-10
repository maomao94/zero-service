package logic

import (
	"strings"

	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
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