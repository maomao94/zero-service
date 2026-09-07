package logic

import (
	"context"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSipProviderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSipProviderLogic {
	return &DeleteSipProviderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteSipProviderLogic) DeleteSipProvider(in *live.DeleteSipProviderReq) (*live.DeleteSipProviderRes, error) {
	id := strings.TrimSpace(in.GetId())
	if id == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商 ID 不能为空")
	}

	// 查询供应商，获取 SipTrunkId
	existing, err := l.svcCtx.MeetingRepo.GetSipProviderByID(l.ctx, id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询供应商失败")
	}
	if existing == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "供应商不存在")
	}

	// 删除 LiveKit SIP Outbound Trunk
	if existing.SipTrunkId != "" {
		_, _ = l.svcCtx.LiveKit.SIP().DeleteSIPTrunk(l.ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: existing.SipTrunkId})
	}

	if err := l.svcCtx.MeetingRepo.DeleteSipProvider(l.ctx, id); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "删除供应商失败")
	}
	l.Logger.Infof("SIP provider deleted: %s trunk=%s", id, existing.SipTrunkId)
	return &live.DeleteSipProviderRes{}, nil
}
