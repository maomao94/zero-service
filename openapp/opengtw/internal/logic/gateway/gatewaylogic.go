package gateway

import (
	"context"
	"encoding/json"

	"zero-service/common/errx"
	"zero-service/facade/openapi/openapi"
	"zero-service/openapp/opengtw/internal/middleware"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type GatewayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 统一转发（Basic 鉴权，method 决定下游调用）
func NewGatewayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GatewayLogic {
	return &GatewayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Gateway 将 data 原样转发给 facade/openapi，只做错误码折叠与成功信封。
func (l *GatewayLogic) Gateway(req *types.GatewayRequest) (json.RawMessage, error) {
	if req == nil || req.Method == "" {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "method 不能为空")
	}

	res, err := l.svcCtx.OpenApiCli.Invoke(l.ctx, &openapi.InvokeReq{
		AppId:     middleware.AppIDFromContext(l.ctx),
		Method:    req.Method,
		Data:      req.Data,
		Tid:       req.Tid,
		Timestamp: req.Timestamp,
	})
	if err != nil {
		code, msg := errx.CodeFromStatus(err)
		if code == 0 {
			code = int(extproto.Code__1_00_INTERNAL)
		}
		l.Errorf("invoke openapi failed app_id=%s method=%s code=%d err=%v",
			middleware.AppIDFromContext(l.ctx), req.Method, code, err)
		return nil, xerrors.New(errx.FoldPublic(code), errx.PublicMessage(code, msg))
	}
	if res == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_INTERNAL), "openapi 返回空响应")
	}
	if res.Code != 0 {
		code := int(res.Code)
		return nil, xerrors.New(errx.FoldPublic(code), errx.PublicMessage(code, res.Msg))
	}
	if len(res.Data) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.RawMessage(res.Data), nil
}
