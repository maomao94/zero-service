package logic

import (
	"context"
	"encoding/json"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/socketpush"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

const meetingInviteEvent = "live:meeting-invite"

type meetingInvitePayload struct {
	MeetingNo    string `json:"meetingNo"`
	MeetingCode  string `json:"meetingCode"`
	MeetingTitle string `json:"meetingTitle"`
	Identity     string `json:"identity"`
	InvitedAt    string `json:"invitedAt"`
}

type NotifyMeetingParticipantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNotifyMeetingParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyMeetingParticipantLogic {
	return &NotifyMeetingParticipantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// NotifyMeetingParticipant submits an invitation notification to the target user's socket room.
func (l *NotifyMeetingParticipantLogic) NotifyMeetingParticipant(in *live.NotifyMeetingParticipantReq) (*live.NotifyMeetingParticipantRes, error) {
	identity := strings.TrimSpace(in.GetIdentity())
	if identity == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "参会人身份不能为空")
	}
	meetingNo := strings.TrimSpace(in.GetMeetingNo())
	meetingCode := strings.TrimSpace(in.GetMeetingCode())
	if meetingNo == "" && meetingCode == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议号或会议码不能同时为空")
	}
	var meeting *gormmodel.LiveMeeting
	var err error
	if meetingNo != "" {
		meeting, err = l.svcCtx.MeetingRepo.GetMeeting(l.ctx, meetingNo)
	} else {
		meeting, err = l.svcCtx.MeetingRepo.GetMeetingByCode(l.ctx, meetingCode)
	}
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
	}
	if l.svcCtx.SocketPushCli == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "通知服务未配置")
	}

	requestID, err := tool.SimpleUUID()
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_INTERNAL, err, "生成通知请求 ID 失败")
	}
	payload, err := json.Marshal(meetingInvitePayload{
		MeetingNo: meeting.MeetingNo, MeetingCode: meeting.MeetingCode, MeetingTitle: meeting.Title,
		Identity:  identity,
		InvitedAt: carbonx.NowDateTime(),
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_INTERNAL, err, "编码通知内容失败")
	}
	if _, err = l.svcCtx.SocketPushCli.BroadcastRoom(l.ctx, &socketpush.BroadcastRoomReq{
		ReqId: requestID, Room: identity, Event: meetingInviteEvent, Payload: string(payload),
	}); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "提交入会通知失败")
	}
	l.Logger.Infof("meeting invitation submitted: meeting=%s identity=%s request=%s", meeting.MeetingNo, identity, requestID)
	return &live.NotifyMeetingParticipantRes{RequestId: requestID}, nil
}
