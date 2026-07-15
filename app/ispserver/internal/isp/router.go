package isp

import (
	"zero-service/app/ispserver/internal/config"
	"zero-service/app/ispserver/internal/handler"
	"zero-service/common/gnetx"
	isp "zero-service/common/isp"
)

func NewRouter(conf config.IspConf) *gnetx.Router {
	r := gnetx.NewRouter()

	// ═══════════════════════════════════════════════════════════════════
	// 上行消息 (Client → Server) — 服务端接收处理
	// ═══════════════════════════════════════════════════════════════════

	// ── 系统消息 ──
	r.Register(isp.MessageIDRegister, gnetx.Async(isp.ServerWrap(handler.NewRegisterHandler(conf)))) // 251-1 注册指令
	r.Register(isp.MessageIDHeartbeat, gnetx.Async(isp.ServerWrap(handler.HandleHeartbeat)))          // 251-2 心跳指令
	// 251-3/251-4 通用应答由 gnetx OnTraffic 通过 Response 接口匹配在途请求，未匹配的静默丢弃。

	unimplemented := gnetx.Async(isp.ServerWrap(handler.HandleUnimplemented))

	// ── 巡视设备上报 ──
	r.Register(isp.MessageIDPatrolDeviceStatusData, unimplemented)  // 1-0 巡视设备状态数据
	r.Register(isp.MessageIDPatrolDeviceRunData, unimplemented)     // 2-0 巡视设备运行数据
	r.Register(isp.MessageIDPatrolDeviceCoordinates, unimplemented) // 3-0 巡视设备坐标
	r.Register(isp.MessageIDPatrolRoute, unimplemented)             // 4-0 巡视路线
	r.Register(isp.MessageIDPatrolDeviceAlarm, unimplemented)       // 5-0 巡视设备异常告警数据

	// ── 模型/环境/任务上报 ──
	r.Register(isp.MessageIDModelUpdateReport, unimplemented) // 11-0 模型更新上报
	r.Register(isp.MessageIDEnvData, unimplemented)           // 21-0 环境/微气象数据
	r.Register(isp.MessageIDTaskStatusData, unimplemented)    // 41-0 任务状态数据
	r.Register(isp.MessageIDPatrolResult, unimplemented)      // 61-0 巡视结果

	// ── 告警与统计上报 ──
	r.Register(isp.MessageIDAlarmData, unimplemented)        // 62-0 告警数据
	r.Register(isp.MessageIDSilentAlarmData, unimplemented)  // 63-0 静默监视告警数据
	r.Register(isp.MessageIDPatrolStatistics, unimplemented) // 81-0 巡视设备统计信息上报

	// ── 无人机上报 ──
	r.Register(isp.MessageIDDroneNestStatus, unimplemented)  // 20001-0 无人机机巢状态数据
	r.Register(isp.MessageIDDroneNestRunData, unimplemented) // 10004-0 无人机机巢运行数据

	// ── 双向确认 (64-0, 67-0 可由任一端发起) ──
	r.Register(isp.EncodeMessageID(isp.TypeAlarmConfirm, isp.CommandReport), unimplemented)  // 64-0 巡视告警确认
	r.Register(isp.EncodeMessageID(isp.TypeResultConfirm, isp.CommandReport), unimplemented) // 67-0 巡视结果确认

	// ═══════════════════════════════════════════════════════════════════
	// 下行指令 (Server → Client) — 不在服务端注册 handler
	// 这些指令由 ispagent 作为 client 接收处理，ispserver 通过 SendCommand 主动下发:
	//   机器人控制 (1-1~7, 2-1~11, 3-1~9, 4-1~5, 21-1~12, 22-5~8, 23-1~4)
	//   任务控制 (41-1~4) / 任务下发 (101-1, 102-1)
	//   模型同步 (61-1~12) / 设备模型下发 (110-41~43)
	//   检修区域配置 (81-4) / 统计查询 (121-1~5)
	//   无人机控制 (20001-1~6)
	// ═══════════════════════════════════════════════════════════════════

	// ── 未匹配消息兜底 ──
	r.Fallback(unimplemented)

	return r
}
