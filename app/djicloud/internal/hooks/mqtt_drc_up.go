package hooks

import (
	"context"
	"time"
	"zero-service/common/tool"

	"zero-service/app/djicloud/internal/drc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

// NewDrcUpHandler 构造 thing/product/{gateway_sn}/drc/up 处理器（DRC 设备→云：回执、遥测推送、心跳等）。
//
// 处理策略：
//  1. 始终输出短摘要日志，未知 method 也不阻断 SDK 分发。
//  2. DrcUnmarshalUpData 全量解析所有已知 method；高频周期上报（心跳、OSD、避障、时延）
//     在业务层跳过 DjiDrcUpEvent 持久化，仅保留关键 command 回执以便链路排障。
//  3. 收到 heart_beat 上行时刷新 DrcManager 存活时间。
//  4. 若配置了 SocketPush，设备心跳上行按房间广播到前端。
func NewDrcUpHandler(db *gormx.DB, drcMgr *drc.Manager, pushCli socketpush.SocketPushClient) djisdk.DrcUpHandler {
	return func(ctx context.Context, gatewaySn string, msg *djisdk.DrcUpMessage, parsed any) error {
		if msg == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] drc/up: nil message, sn=%s", gatewaySn)
			return nil
		}
		logx.WithContext(ctx).Debugf("[dji-cloud] drc/up: sn=%s method=%s ts=%d", gatewaySn, msg.Method, msg.Timestamp)
		reportedAt := time.Now()
		if msg.Timestamp > 0 {
			reportedAt = reportTime(msg.Timestamp)
		}
		// 高频周期上报（心跳、OSD、避障、时延）及初始状态订阅频次极高，不持久化到数据库
		if msg.Method != djisdk.MethodDrcHeartBeat && msg.Method != djisdk.MethodDrcOsdInfoPush &&
			msg.Method != djisdk.MethodDrcHsiInfoPush && msg.Method != djisdk.MethodDrcDelayInfoPush &&
			msg.Method != djisdk.MethodDrcInitialStateSubscribe {
			if err := gormx.CreateRecord(db.WithContext(ctx), &gormmodel.DjiDrcUpEvent{
				GatewaySn:  gatewaySn,
				Method:     msg.Method,
				RawJSON:    toJSONString(parsed),
				Summary:    djisdk.DrcUpPayloadSummary(parsed),
				ReportedAt: reportedAt,
			}); err != nil {
				logx.WithContext(ctx).Errorf("[dji-cloud] create drc/up event failed: %v", err)
			}
		}

		// 心跳上行：刷新 DRC 存活时间
		if msg.Method == djisdk.MethodDrcHeartBeat && drcMgr != nil {
			drcMgr.OnDeviceHeartbeat(ctx, gatewaySn)
			if pushCli != nil {
				pushCtx := context.WithoutCancel(ctx)
				threading.GoSafe(func() {
					reqId, _ := tool.SimpleUUID()
					room := "drc:heartbeat:" + gatewaySn
					_, err := pushCli.BroadcastRoom(pushCtx, &socketpush.BroadcastRoomReq{
						ReqId:   reqId,
						Room:    room,
						Event:   "drc:" + msg.Method,
						Payload: string(msg.Data),
					})
					if err != nil {
						logx.WithContext(pushCtx).Errorf("[dji-cloud] socket push drc/up failed: sn=%s method=%s err=%v", gatewaySn, msg.Method, err)
					}
				})
			}
		}

		return nil
	}
}
