package logic

import (
	"context"

	"zero-service/app/openapi/internal/dispatch"
	"zero-service/app/openapi/internal/svc"
	"zero-service/app/openapi/openapi"
	"zero-service/common/tool"
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
	if in == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_MISSING, "invoke 请求不能为空")
	}
	if in.Method == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_MISSING, "method 不能为空")
	}

	msg, err := l.svcCtx.Registry.Invoke(l.ctx, in.Method, in.Data)
	if err != nil {
		l.Errorf("invoke failed app_id=%s method=%s err=%v", in.AppId, in.Method, err)
		return nil, err
	}

	data, err := dispatch.Encode(msg)
	if err != nil {
		l.Errorf("encode response failed app_id=%s method=%s err=%v", in.AppId, in.Method, err)
		return nil, tool.NewErrorByPbCode(extproto.Code__1_00_INTERNAL, "结果序列化失败")
	}
	return &openapi.InvokeRes{Data: string(data)}, nil
}
