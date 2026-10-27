package app

import (
	"context"
	"strings"
	"unicode/utf8"

	"zero-service/app/opengtw/internal/security"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type CreateAppLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建应用（secret 仅返回一次）
func NewCreateAppLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAppLogic {
	return &CreateAppLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAppLogic) CreateApp(req *types.CreateAppRequest) (*types.CreateAppReply, error) {
	if req == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "请求不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "name 不能为空")
	}
	if utf8.RuneCountInString(name) > 64 {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "name 不能超过 64 个字符")
	}
	remark := strings.TrimSpace(req.Remark)
	if utf8.RuneCountInString(remark) > 255 {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "remark 不能超过 255 个字符")
	}
	if req.SignRequired && !l.svcCtx.SigningReady() {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "签名配置未就绪")
	}

	// appId 由 UUID v4 生成（122 位随机），碰撞概率可忽略，不再做重试；
	// open_app.app_id 的主键约束仍是最终兜底。
	appID, err := security.NewAppID()
	if err != nil {
		l.Errorf("generate app_id failed: %v", err)
		return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成应用 ID 失败")
	}
	secret, err := security.NewSecret()
	if err != nil {
		l.Errorf("generate app secret failed: %v", err)
		return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成应用密钥失败")
	}
	signKey, signKeyCipher := "", ""
	if req.SignRequired {
		signKey, err = security.NewSignKey()
		if err != nil {
			l.Errorf("generate sign key failed: %v", err)
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成签名密钥失败")
		}
		signKeyCipher, err = security.EncryptSignKey(l.svcCtx.Config.SignKeyEncryptionKey, signKey)
		if err != nil {
			l.Errorf("encrypt sign key failed: %v", err)
			return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "保护签名密钥失败")
		}
	}

	app := &gormmodel.OpenApp{
		AppId:        appID,
		Name:         name,
		Remark:       remark,
		Status:       1,
		SignRequired: req.SignRequired,
	}
	secretModel := &gormmodel.OpenAppSecret{
		AppId:         appID,
		SecretHash:    security.HashSecret(secret),
		SignKeyCipher: signKeyCipher,
		Status:        1,
	}
	err = l.svcCtx.DB.Transact(func(tx *gormx.DB) error {
		if err := tx.Create(app).Error; err != nil {
			return err
		}
		return tx.Create(secretModel).Error
	})
	if err != nil {
		l.Errorf("create open app failed: %v", err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "创建应用失败")
	}

	l.svcCtx.RefreshCredentials(l.ctx)
	return &types.CreateAppReply{
		AppId:        app.AppId,
		Name:         app.Name,
		Secret:       secret,
		Status:       app.Status,
		SignRequired: app.SignRequired,
		SignKey:      signKey,
		CreatedAt:    formatTime(app.CreatedAt),
	}, nil
}
