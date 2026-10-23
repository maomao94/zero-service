package logic

import (
	"context"

	"zero-service/common/errx"
	"zero-service/facade/openapi/internal/dispatch"
	"zero-service/facade/openapi/internal/svc"
	"zero-service/facade/openapi/openapi"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type InvokeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInvokeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InvokeLogic {
	return &InvokeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InvokeLogic) Invoke(in *openapi.InvokeReq) (*openapi.InvokeRes, error) {
	res := &openapi.InvokeRes{}
	if in == nil {
		res.Code = int32(extproto.Code__1_00_PARAM_MISSING)
		res.Msg = "invoke 请求不能为空"
		return res, nil
	}
	res.Tid = in.Tid
	if in.Method == "" {
		res.Code = int32(extproto.Code__1_00_PARAM_MISSING)
		res.Msg = "method 不能为空"
		return res, nil
	}

	msg, err := l.svcCtx.Registry.Invoke(l.ctx, in.Method, in.Data)
	if err != nil {
		code, msgText := errx.CodeFromStatus(err)
		if code == 0 {
			code = int(extproto.Code__1_00_INTERNAL)
		}
		res.Code = int32(code)
		res.Msg = msgText
		l.Errorf("invoke failed app_id=%s method=%s code=%d err=%v", in.AppId, in.Method, code, err)
		return res, nil
	}

	data, err := dispatch.Encode(msg)
	if err != nil {
		res.Code = int32(extproto.Code__1_00_INTERNAL)
		res.Msg = "结果序列化失败"
		l.Errorf("encode response failed app_id=%s method=%s err=%v", in.AppId, in.Method, err)
		return res, nil
	}
	res.Code = 0
	res.Msg = "ok"
	res.Data = data
	return res, nil
}
