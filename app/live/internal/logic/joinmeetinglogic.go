package logic

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/livekitx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type JoinMeetingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewJoinMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JoinMeetingLogic {
	return &JoinMeetingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 加入会议（校验会议后发放 join token，浏览器直连 LiveKit）
func (l *JoinMeetingLogic) JoinMeeting(in *live.JoinMeetingReq) (*live.JoinMeetingRes, error) {
	if err := requireMeetingIdentity(in.MeetingNo, in.Identity); err != nil {
		return nil, err
	}
	meeting, err := l.svcCtx.MeetingRepo.GetMeeting(l.ctx, in.MeetingNo)
	if err != nil {
		return nil, meetingErr(err)
	}
	if meeting.Status == gormmodel.MeetingStatusEnded {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "会议已结束")
	}

	token, err := livekitx.NewJoinToken(livekitx.JoinTokenOptions{
		APIKey:         l.svcCtx.Config.LiveKit.ApiKey,
		APISecret:      l.svcCtx.Config.LiveKit.ApiSecret,
		Room:           in.MeetingNo,
		Identity:       in.Identity,
		ValidFor:       l.svcCtx.Config.LiveKit.TokenValidFor,
		CanPublish:     true,
		CanSubscribe:   true,
		CanPublishData: true,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "生成入会 token 失败")
	}

	// 参会记录（webhook participant_joined 到达前先落库，保证查询即时准确）；
	// 创建时创建人与更新人同时赋值
	now := time.Now()
	operator := authctx.GetUserId(l.ctx)
	participant := &gormmodel.LiveMeetingParticipant{
		CreateUser: sql.NullString{String: operator, Valid: operator != ""},
		UpdateUser: sql.NullString{String: operator, Valid: operator != ""},
		MeetingNo:  in.MeetingNo,
		Identity:   in.Identity,
		Name:       strings.TrimSpace(in.Name),
		Status:     gormmodel.ParticipantStatusJoined,
		JoinTime:   now,
	}
	if err := l.svcCtx.MeetingRepo.UpsertParticipant(l.ctx, participant); err != nil {
		l.Logger.Errorf("upsert participant failed: %v", err)
	}

	l.Logger.Infof("join token issued: meeting=%s identity=%s", in.MeetingNo, in.Identity)
	return &live.JoinMeetingRes{
		Token:   token,
		WsUrl:   wsURL(l.svcCtx.Config.LiveKit.Url),
		Meeting: toMeetingInfo(meeting),
	}, nil
}

// meetingErr 把 repo 错误映射为 extproto 业务错误码。
func meetingErr(err error) error {
	if err == svc.ErrMeetingNotFound {
		return tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "会议不存在")
	}
	return tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询会议失败")
}