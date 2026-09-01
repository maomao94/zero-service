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

type PerformMeetingRpcLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPerformMeetingRpcLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PerformMeetingRpcLogic {
	return &PerformMeetingRpcLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 服务端对参与者执行 RPC
func (l *PerformMeetingRpcLogic) PerformMeetingRpc(in *live.PerformMeetingRpcReq) (*live.PerformMeetingRpcRes, error) {
	if err := requireMeetingIdentity(in.MeetingNo, in.Identity); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Method) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "method 不能为空")
	}
	resp, err := l.svcCtx.LiveKit.Room().PerformRpc(l.ctx, &livekit.PerformRpcRequest{
		Room:                in.MeetingNo,
		DestinationIdentity: in.Identity,
		Method:              in.Method,
		Payload:             in.Payload,
		ResponseTimeoutMs:   in.ResponseTimeoutMs,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "RPC 调用失败")
	}
	return &live.PerformMeetingRpcRes{Response: resp.GetPayload()}, nil
}
