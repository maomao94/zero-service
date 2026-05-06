package hooks

import (
	"context"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
)

// NewDrcUpHandler 构造 thing/product/{gateway_sn}/drc/up 处理器（DRC 设备→云：回执、遥测推送、心跳等）。
//
// 处理策略：
//  1. 始终输出短摘要日志，未知 method 也不阻断 SDK 分发。
//  2. 按上行消息逐条写入 DjiDrcUpEvent，保留 RawJSON 与 Summary 便于链路排障。
func NewDrcUpHandler(db *gormx.DB) djisdk.DrcUpHandler {
	return func(ctx context.Context, gatewaySn string, msg *djisdk.DrcUpMessage, parsed any) error {
		if msg == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] drc/up: nil message, sn=%s", gatewaySn)
			return nil
		}
		sum := djisdk.DrcUpPayloadSummary(msg.Method, parsed)
		logx.WithContext(ctx).Debugf("[dji-cloud] drc/up: sn=%s method=%s ts=%d %s", gatewaySn, msg.Method, msg.Timestamp, sum)
		reportedAt := time.Now()
		if msg.Timestamp > 0 {
			reportedAt = reportTime(msg.Timestamp)
		}
		if err := gormx.CreateRecord(ctx, db, &gormmodel.DjiDrcUpEvent{
			GatewaySn:  gatewaySn,
			Method:     msg.Method,
			RawJSON:    toJSONString(parsed),
			Summary:    sum,
			ReportedAt: reportedAt,
		}); err != nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] create drc/up event failed: %v", err)
		}
		return nil
	}
}
