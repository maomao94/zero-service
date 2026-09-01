package logic

import (
	"context"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

type MuteParticipantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMuteParticipantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MuteParticipantLogic {
	return &MuteParticipantLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 静音/取消静音参与者音频或视频
func (l *MuteParticipantLogic) MuteParticipant(in *live.MuteParticipantReq) (*live.MuteParticipantRes, error) {
	if err := requireMeetingIdentity(in.MeetingNo, in.Identity); err != nil {
		return nil, err
	}
	var trackType livekit.TrackType
	switch in.Kind {
	case "video":
		trackType = livekit.TrackType_VIDEO
	case "audio", "":
		trackType = livekit.TrackType_AUDIO
	default:
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "kind 仅支持 audio/video")
	}
	if err := muteTracksByType(l.ctx, l.svcCtx.LiveKit.Room(), in.MeetingNo, in.Identity, trackType, in.Muted); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "静音操作失败")
	}
	return &live.MuteParticipantRes{}, nil
}

// muteTracksByType 列出参与者的所有轨道，按类型逐个静音/取消静音。
func muteTracksByType(ctx context.Context, roomSvc livekit.RoomService, roomName, identity string, trackType livekit.TrackType, muted bool) error {
	participants, err := roomSvc.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: roomName})
	if err != nil {
		return err
	}
	for _, p := range participants.GetParticipants() {
		if p.Identity != identity {
			continue
		}
		for _, track := range p.Tracks {
			if track.Type != trackType {
				continue
			}
			if _, err := roomSvc.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
				Room:     roomName,
				Identity: identity,
				TrackSid: track.Sid,
				Muted:    muted,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
