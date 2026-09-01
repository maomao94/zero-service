package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

type ListParticipantsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListParticipantsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListParticipantsLogic {
	return &ListParticipantsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询会议参与者
func (l *ListParticipantsLogic) ListParticipants(in *live.ListParticipantsReq) (*live.ListParticipantsRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	participants, err := l.svcCtx.MeetingRepo.ListParticipants(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询参与者失败")
	}
	resp := &live.ListParticipantsRes{}
	for i := range participants {
		resp.Participants = append(resp.Participants, &live.ParticipantInfo{
			Identity: participants[i].Identity,
			Name:     participants[i].Name,
			Status:   int32(participants[i].Status),
			JoinTime: carbonx.FormatDateTimeOrEmpty(participants[i].JoinTime),
			LeftTime: carbonx.FormatNullDateTime(participants[i].LeftTime),
		})
	}
	return resp, nil
}
