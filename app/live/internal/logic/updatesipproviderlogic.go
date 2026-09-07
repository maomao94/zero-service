package logic

import (
	"context"
	"encoding/json"
	"strings"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
	if name := strings.TrimSpace(in.GetName()); name != "" {
		updates["name"] = name
	}
	if address := strings.TrimSpace(in.GetAddress()); address != "" {
		updates["address"] = address
	}
	if len(in.GetNumbers()) > 0 {
		numbersJSON, _ := json.Marshal(in.GetNumbers())
		updates["numbers"] = string(numbersJSON)
	}
	if in.GetAuthUsername() != "" {
		updates["auth_username"] = in.GetAuthUsername()
	}
	if in.GetAuthPassword() != "" {
		updates["auth_password"] = in.GetAuthPassword()
	}
	if in.GetStatus() > 0 {
		updates["status"] = in.GetStatus()
	}

	if len(updates) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "没有需要更新的字段")
	}

	if err := l.svcCtx.MeetingRepo.UpdateSipProvider(l.ctx, id, updates); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新供应商失败")
	}

	// 重新查询返回
	updated, _ := l.svcCtx.MeetingRepo.GetSipProviderByID(l.ctx, id)
	l.Logger.Infof("SIP provider updated: %s", id)
	return &live.UpdateSipProviderRes{Provider: toSipProviderInfo(updated)}, nil
}
