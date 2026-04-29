package hooks

import (
	"context"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

// NewDrcUpHandler 构造 thing/product/{device_sn}/drc/up 处理器（DRC 设备→云：回执、遥测推送、心跳等）。
func NewDrcUpHandler(_ *collection.Cache) djisdk.DrcUpHandler {
	return func(ctx context.Context, gatewaySn string, msg *djisdk.DrcUpMessage, parsed any) error {
		sum := djisdk.DrcUpPayloadSummary(msg.Method, parsed)
		logx.WithContext(ctx).Debugf("[dji-cloud] drc/up: sn=%s method=%s ts=%d %s", gatewaySn, msg.Method, msg.Timestamp, sum)
		return nil
	}
}
