package logic

import (
	"context"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

type SendMeetingDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendMeetingDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMeetingDataLogic {
	return &SendMeetingDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向会议发送 Data（destinations 为空则广播）
func (l *SendMeetingDataLogic) SendMeetingData(in *live.SendMeetingDataReq) (*live.SendMeetingDataRes, error) {
	if err := requireMeetingNo(in.MeetingNo); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Topic) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "topic 不能为空")
	}
	if _, err := l.svcCtx.LiveKit.Room().SendData(l.ctx, &livekit.SendDataRequest{
		Room:                  in.MeetingNo,
		Topic:                 &in.Topic,
		Data:                  in.Payload,
		Kind:                  livekit.DataPacket_RELIABLE,
		DestinationIdentities: in.Destinations,
	}); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "发送数据失败")
	}
	return &live.SendMeetingDataRes{}, nil
}
