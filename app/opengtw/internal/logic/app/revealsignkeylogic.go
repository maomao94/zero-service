package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"zero-service/app/opengtw/internal/signkey"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/common/authctx"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type RevealSignKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查看当前签名密钥（审计）
func NewRevealSignKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevealSignKeyLogic {
	return &RevealSignKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// RevealSignKey 返回应用当前生效的 signKey 明文。
//
// 这是管理面的敏感读操作：响应不落盘（NoStore 中间件），成功时记录操作人审计日志；
// 失败时只返回稳定的业务文案，不向客户端暴露主密钥、密文格式或解密细节。
func (l *RevealSignKeyLogic) RevealSignKey(req *types.RevealSignKeyRequest) (*types.RevealSignKeyReply, error) {
	if req == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "请求不能为空")
	}
	if !l.svcCtx.Config.SignKeyRevealEnabled {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "查看签名密钥功能未启用")
	}

	appID := strings.TrimSpace(req.AppId)
	plaintext, err := signkey.Reveal(
		l.ctx,
		l.svcCtx.AppStore,
		l.svcCtx.SecretStore,
		l.svcCtx.Config.SignKeyEncryptionKey,
		appID,
	)
	if err != nil {
		return nil, l.mapRevealError(appID, err)
	}

	l.Infow("reveal sign key",
		logx.Field("app_id", appID),
		logx.Field("operator_id", authctx.GetUserId(l.ctx)),
		logx.Field("operator_name", authctx.GetUserName(l.ctx)),
	)
	return &types.RevealSignKeyReply{
		AppId:        appID,
		SignKey:      plaintext,
		SignRequired: true,
		RevealedAt:   formatTime(time.Now()),
	}, nil
}

func (l *RevealSignKeyLogic) mapRevealError(appID string, err error) error {
	switch {
	case errors.Is(err, signkey.ErrAppIDRequired):
		return xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), err.Error())
	case errors.Is(err, signkey.ErrAppNotFound):
		return xerrors.New(int(extproto.Code__1_00_RECORD_NOT_EXIST), err.Error())
	case errors.Is(err, signkey.ErrNotEnabled):
		return xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), err.Error())
	case errors.Is(err, signkey.ErrEncryptionKeyMissing):
		return xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "签名配置未就绪")
	case errors.Is(err, signkey.ErrKeyMissing):
		return xerrors.New(int(extproto.Code__1_00_RECORD_NOT_EXIST), err.Error())
	case errors.Is(err, signkey.ErrDecrypt):
		l.Errorf("decrypt sign key failed, app_id=%s: %v", appID, err)
		return xerrors.New(int(extproto.Code__1_00_INTERNAL), "解密签名密钥失败")
	default:
		l.Errorf("reveal sign key failed, app_id=%s: %v", appID, err)
		return xerrors.New(int(extproto.Code__1_00_DB), "查询签名密钥失败")
	}
}
