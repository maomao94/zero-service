package logic

import (
	"context"
	"encoding/json"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSipProviderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSipProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSipProviderLogic {
	return &UpdateSipProviderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateSipProviderLogic) UpdateSipProvider(in *live.UpdateSipProviderReq) (*live.UpdateSipProviderRes, error) {
	id := strings.TrimSpace(in.GetId())
	if id == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "供应商 ID 不能为空")
	}

	// 检查是否存在
	existing, err := l.svcCtx.MeetingRepo.GetSipProviderByID(l.ctx, id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询供应商失败")
	}
	if existing == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "供应商不存在")
	}

	// 构造更新字段（只更新非空字段）
	updates := map[string]any{}
	trunkChanged := false
	if name := strings.TrimSpace(in.GetName()); name != "" {
		updates["name"] = name
		trunkChanged = true
	}
	if address := strings.TrimSpace(in.GetAddress()); address != "" {
		updates["address"] = address
		trunkChanged = true
	}
	if len(in.GetNumbers()) > 0 {
		numbersJSON, _ := json.Marshal(in.GetNumbers())
		updates["numbers"] = string(numbersJSON)
		trunkChanged = true
	}
	if in.GetAuthUsername() != "" {
		updates["auth_username"] = in.GetAuthUsername()
		trunkChanged = true
	}
	if in.GetAuthPassword() != "" {
		updates["auth_password"] = in.GetAuthPassword()
		trunkChanged = true
	}
	if in.GetStatus() > 0 {
		updates["status"] = in.GetStatus()
	}

	if len(updates) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "没有需要更新的字段")
	}

	// trunk 相关配置变更时，删除旧 trunk 并重建
	if trunkChanged && existing.SipTrunkId != "" {
		l.Logger.Infof("SIP trunk %s config changed, recreating (provider=%s)", existing.SipTrunkId, existing.Code)
		_, _ = l.svcCtx.LiveKit.SIP().DeleteSIPTrunk(l.ctx, &livekit.DeleteSIPTrunkRequest{SipTrunkId: existing.SipTrunkId})
	}

	if err := l.svcCtx.MeetingRepo.UpdateSipProvider(l.ctx, id, updates); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新供应商失败")
	}

	// trunk 相关配置变更时，创建新 trunk 并更新 SipTrunkId
	if trunkChanged {
		updated, _ := l.svcCtx.MeetingRepo.GetSipProviderByID(l.ctx, id)
		var numbers []string
		_ = json.Unmarshal([]byte(updated.Numbers), &numbers)
		trunkRes, err := l.svcCtx.LiveKit.SIP().CreateSIPOutboundTrunk(l.ctx, &livekit.CreateSIPOutboundTrunkRequest{
			Trunk: &livekit.SIPOutboundTrunkInfo{
				Name:         updated.Name,
				Address:      updated.Address,
				Numbers:      numbers,
				AuthUsername: updated.AuthUsername,
				AuthPassword: updated.AuthPassword,
			},
		})
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "创建 SIP trunk 失败")
		}
		if err := l.svcCtx.MeetingRepo.UpdateSipProvider(l.ctx, id, map[string]any{"sip_trunk_id": trunkRes.SipTrunkId}); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新 trunk ID 失败")
		}
		l.Logger.Infof("SIP trunk recreated: %s -> %s (provider=%s)", existing.SipTrunkId, trunkRes.SipTrunkId, existing.Code)
	}

	// 重新查询返回
	updated, _ := l.svcCtx.MeetingRepo.GetSipProviderByID(l.ctx, id)
	l.Logger.Infof("SIP provider updated: %s", id)
	return &live.UpdateSipProviderRes{Provider: toSipProviderInfo(updated)}, nil
}
