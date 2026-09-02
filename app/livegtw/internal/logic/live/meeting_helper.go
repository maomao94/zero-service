package live

import (
	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/types"
)

func toMeetingInfo(m *live.MeetingInfo) types.MeetingInfo {
	if m == nil {
		return types.MeetingInfo{}
	}
	return types.MeetingInfo{
		MeetingNo:  m.GetMeetingNo(),
		Title:      m.GetTitle(),
		Status:     m.GetStatus(),
		CreateUser: m.GetCreateUser(),
		UpdateUser: m.GetUpdateUser(),
		DeptCode:   m.GetDeptCode(),
		StartTime:  m.GetStartTime(),
		EndTime:    m.GetEndTime(),
		CreateTime: m.GetCreateTime(),
	}
}

func toParticipantInfo(p *live.ParticipantInfo) types.ParticipantInfo {
	if p == nil {
		return types.ParticipantInfo{}
	}
	return types.ParticipantInfo{
		Identity: p.GetIdentity(),
		Name:     p.GetName(),
		Status:   p.GetStatus(),
		JoinTime: p.GetJoinTime(),
		LeftTime: p.GetLeftTime(),
	}
}
