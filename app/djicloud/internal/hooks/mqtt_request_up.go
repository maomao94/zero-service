package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

// NewDeviceRequestHandler 构造 thing/product/{gateway_sn}/requests 上行处理器。
func NewDeviceRequestHandler() djisdk.RequestHandler {
	return func(ctx context.Context, gatewaySn string, req *djisdk.RequestMessage) (int, any, error) {
		if req == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] request: nil request payload, sn=%s", gatewaySn)
			return djisdk.PlatformResultHandlerError, nil, nil
		}
		logx.WithContext(ctx).Infof("[dji-cloud] request: method=%s sn=%s tid=%s", req.Method, gatewaySn, req.Tid)
		return djisdk.PlatformResultOK, nil, nil
	}
}
