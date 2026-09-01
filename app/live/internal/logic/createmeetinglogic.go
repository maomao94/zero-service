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
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMeetingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMeetingLogic {
	return &CreateMeetingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建会议（落库 + 创建 LiveKit 房间）
func (l *CreateMeetingLogic) CreateMeeting(in *live.CreateMeetingReq) (*live.CreateMeetingRes, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "会议标题不能为空")
	}
	// 创建人/机构取自 gRPC metadata（网关注入 user-id/dept-code），不通过 proto 传输；
	// 创建时创建人与更新人同时赋值（项目惯例，对齐 trigger）
	creator := authctx.GetUserId(l.ctx)
	if creator == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_03_UNAUTHORIZED, "缺少用户身份")
	}
	deptCode := authctx.GetDeptCode(l.ctx)
	now := time.Now()
	// 会议号用 IdUtil（Redis 序号 + 日期；category=live 与其他业务隔离，
	// outDescType=M 标识会议单据）。Redis 是 live 服务必需依赖。
	meetingNo, err := l.svcCtx.IdUtil.NextId("M", "live")
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_CACHE, err, "生成会议号失败")
	}

	emptyTimeout := in.GetEmptyTimeout()
	if emptyTimeout == 0 {
		emptyTimeout = 600
	}
	departureTimeout := in.GetDepartureTimeout()
	if departureTimeout == 0 {
		departureTimeout = 120
	}
	maxParticipants := in.GetMaxParticipants()
	if maxParticipants == 0 {
		maxParticipants = 50
	}
	// 先创建 LiveKit 房间（房间名 = 会议号），成功后再落库；
	// 落库失败时清理房间，保证单据与房间一致。
	if _, err := l.svcCtx.LiveKit.Room().CreateRoom(l.ctx, &livekit.CreateRoomRequest{
		Name:             meetingNo,
		EmptyTimeout:     emptyTimeout,
		DepartureTimeout: departureTimeout,
		MaxParticipants:  maxParticipants,
		Metadata:         in.GetMetadata(),
	}); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "创建房间失败")
	}
	meeting := &gormmodel.LiveMeeting{
		CreateUser: sql.NullString{String: creator, Valid: creator != ""},
		UpdateUser: sql.NullString{String: creator, Valid: creator != ""},
		DeptCode:   sql.NullString{String: deptCode, Valid: deptCode != ""},
		MeetingNo:  meetingNo,
		Title:      strings.TrimSpace(in.Title),
		Status:     gormmodel.MeetingStatusActive,
		StartTime:  now,
	}
	if err := l.svcCtx.MeetingRepo.CreateMeeting(l.ctx, meeting); err != nil {
		_, _ = l.svcCtx.LiveKit.Room().DeleteRoom(l.ctx, &livekit.DeleteRoomRequest{Room: meetingNo})
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "会议落库失败")
	}
	l.Logger.Infof("meeting created: %s title=%q creator=%s", meetingNo, meeting.Title, creator)
	return &live.CreateMeetingRes{Meeting: toMeetingInfo(meeting)}, nil
}