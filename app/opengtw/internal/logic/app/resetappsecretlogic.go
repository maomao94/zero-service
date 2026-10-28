package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
	"gorm.io/gorm"
)

const oldSecretRetention = 24 * time.Hour

type ResetAppSecretLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 重置应用密钥（旧密钥保留 24h）
func NewResetAppSecretLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetAppSecretLogic {
	return &ResetAppSecretLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetAppSecretLogic) ResetAppSecret(req *types.ResetAppSecretRequest) (*types.ResetAppSecretReply, error) {
	if req == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "请求不能为空")
	}
	appID := strings.TrimSpace(req.AppId)
	if appID == "" {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "appId 不能为空")
	}

	app, err := l.svcCtx.AppStore.FindByAppId(l.ctx, appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerrors.New(int(extproto.Code__1_00_RECORD_NOT_EXIST), "应用不存在")
		}
		l.Errorf("find open app failed, app_id=%s: %v", appID, err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "查询应用失败")
	}

	secret, err := security.NewSecret()
	if err != nil {
		l.Errorf("generate app secret failed: %v", err)
		return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成应用密钥失败")
	}
	signKey, signKeyCipher := "", ""
	if app.SignRequired {
		if !l.svcCtx.SigningReady() {
			return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "签名配置未就绪")
		}
		signKey, err = security.NewSignKey()
		if err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成签名密钥失败")
		}
		signKeyCipher, err = security.EncryptSignKey(l.svcCtx.Config.SignKeyEncryptionKey, signKey)
		if err != nil {
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "保护签名密钥失败")
		}
	}
	expireAt := time.Now().Add(oldSecretRetention)
	if err := l.svcCtx.SecretStore.Replace(l.ctx, appID, security.HashSecret(secret), signKeyCipher, expireAt); err != nil {
		l.Errorf("replace app secret failed, app_id=%s: %v", appID, err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "重置应用密钥失败")
	}

	l.svcCtx.RefreshCredentials(l.ctx)
	return &types.ResetAppSecretReply{
		AppId:             appID,
		Secret:            secret,
		OldSecretExpireAt: formatTime(expireAt),
		SignKey:           signKey,
		SignKeyExpireAt:   formatTime(expireAt),
	}, nil
}
