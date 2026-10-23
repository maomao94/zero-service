package app

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
	"zero-service/openapp/opengtw/internal/security"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
	"gorm.io/gorm"
)

const maxCreateRetries = 5

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

	for attempt := 0; attempt < maxCreateRetries; attempt++ {
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

		// 极小概率命中已存在的 app_id，重新生成。
		if _, err := l.svcCtx.AppStore.FindByAppId(l.ctx, appID); err == nil {
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			l.Errorf("check app_id failed: %v", err)
			return nil, xerrors.New(int(extproto.Code__1_00_DB), "数据库错误")
		}

		app := &gormmodel.OpenApp{
			AppId:  appID,
			Name:   name,
			Remark: remark,
			Status: 1,
		}
		secretModel := &gormmodel.OpenAppSecret{
			AppId:      appID,
			SecretHash: security.HashSecret(secret),
			Status:     1,
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
			AppId:     app.AppId,
			Name:      app.Name,
			Secret:    secret,
			Status:    app.Status,
			CreatedAt: formatTime(app.CreatedAt),
		}, nil
	}

	return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "生成唯一应用 ID 失败")
}
