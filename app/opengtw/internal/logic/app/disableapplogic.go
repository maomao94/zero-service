package app

import (
	"context"
	"strings"

	"zero-service/app/opengtw/internal/security"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type DisableAppLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 启用或停用应用
func NewDisableAppLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableAppLogic {
	return &DisableAppLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableAppLogic) DisableApp(req *types.DisableAppRequest) (*types.DisableAppReply, error) {
	if req == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "请求不能为空")
	}
	appID := strings.TrimSpace(req.AppId)
	if appID == "" {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "appId 不能为空")
	}
	if req.Status != 1 && req.Status != 2 {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "status 只能是 1（启用）或 2（停用）")
	}
	if req.SignRequired != nil && *req.SignRequired && !l.svcCtx.SigningReady() {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "签名配置未就绪")
	}
	signKey := ""
	if req.SignRequired != nil && *req.SignRequired {
		var err error
		signKey, err = security.NewSignKey()
		if err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成签名密钥失败")
		}
		signKeyCipher, err := security.EncryptSignKey(l.svcCtx.Config.SignKeyEncryptionKey, signKey)
		if err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "保护签名密钥失败")
		}
		if err := l.svcCtx.SecretStore.UpdateSignKey(l.ctx, appID, signKeyCipher); err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_DB), "保存签名密钥失败")
		}
	}

	rows, err := l.svcCtx.AppStore.UpdateStatus(l.ctx, appID, req.Status)
	if err != nil {
		l.Errorf("update open app status failed, app_id=%s: %v", appID, err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "更新应用状态失败")
	}
	if rows == 0 {
		return nil, xerrors.New(int(extproto.Code__1_00_RECORD_NOT_EXIST), "应用不存在")
	}
	if req.SignRequired != nil {
		if _, err := l.svcCtx.AppStore.UpdateSignRequired(l.ctx, appID, *req.SignRequired); err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_DB), "更新签名配置失败")
		}
	}

	l.svcCtx.RefreshCredentials(l.ctx)
	app, err := l.svcCtx.AppStore.FindByAppId(l.ctx, appID)
	if err != nil {
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "查询应用状态失败")
	}
	return &types.DisableAppReply{AppId: appID, Status: req.Status, SignRequired: app.SignRequired, SignKey: signKey}, nil
}
