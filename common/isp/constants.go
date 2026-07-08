package isp

// ISP 协议常量。messageId = (type << 16) | command，高 16 位存 Type，低 16 位存 Command。
//
// Command=0 的消息为上报类（server→client），Command≠0 为指令类（client→server 或双向）。
// Type 在 client 指令和 server 上报中可能复用，通过 Command 区分方向。
//
// 对标 Java com.allcore.sip.transport.commons.TSip 接口定义。

const (
	// 帧标志：大端字节序 0xEB90，作为 ISP 协议帧的起始和结束标记
	FrameFlag uint16 = 0xEB90

	// 会话源标识，对应 Java TMessage.sessionSource
	SessionSourceClient byte = 0x00 // 客户端发起
	SessionSourceServer byte = 0x01 // 服务端响应
)

// XML 根元素名称，可配置切换。
// 对应 Java 侧根元素根据上级系统属性做切换的逻辑。
const (
	RootPatrolHost   = "PatrolHost"   // 巡视主机（上级系统为巡视主机侧时使用）
	RootPatrolDevice = "PatrolDevice" // 巡视设备（上级系统为设备/机器人侧时使用）
)

// 通用应答（251-3 / 251-4）的响应状态码，写入 XML <Code> 字段。
const (
	StatusRetry   = "100" // 需重发
	StatusSuccess = "200" // 成功
	StatusReject  = "400" // 拒绝
	StatusError   = "500" // 错误
)

// ═══════════════════════════════════════════════════════════════════════════
// Type 枚举 — messageId 高 16 位，标识消息大类
// ═══════════════════════════════════════════════════════════════════════════

const (
	// ── 系统消息（Type 251）─ 注册/心跳/通用应答 ──
	TypeSystem int32 = 251 // 系统消息

	// ── 巡视设备 ─ server→client 上报数据 ──
	TypePatrolDeviceStatusData  int32 = 1 // 巡视设备状态数据
	TypePatrolDeviceRunData     int32 = 2 // 巡视设备运行数据
	TypePatrolDeviceCoordinates int32 = 3 // 巡视设备坐标
	TypePatrolRoute             int32 = 4 // 巡视路线
	TypePatrolDeviceAlarm       int32 = 5 // 巡视设备异常告警数据

	// ── 机器人本体控制 ─ client→server 指令，Type 与巡视设备状态共用──
	TypeRobotBody int32 = 1 // 机器人本体控制（远方复位/系统自检/一键返航/充电/模式切换/控制权）

	// ── 机器人车体控制 ─ Type=2 ──
	TypeRobotChassis int32 = 2 // 机器人车体控制（前进/后退/转向/停止/升降/平移/步态）

	// ── 机器人云台控制 ─ Type=3 ──
	TypeRobotPTZ int32 = 3 // 机器人云台控制（上下俯仰/左右转/升降/预置位/停止/复位）

	// ── 机器人辅助设备 ─ Type=4 ──
	TypeRobotAux int32 = 4 // 机器人辅助设备（红外电源/雨刷/超声/红外射灯/辅助照明）

	// ── 模型更新上报 ─ server→client ──
	TypeModelUpdateReport int32 = 11 // 模型更新上报（11-0）

	// ── 可见光摄像机 ─ Type=21 ──
	TypeVisibleCamera int32 = 21 // 可见光摄像机控制（变焦/聚焦/自动聚焦/重启/倍率/聚焦值）

	// ── 环境数据 ─ server→client，Type 与可见光摄像机共用 ──
	TypeEnvData int32 = 21 // 环境数据（微气象数据，21-0）

	// ── 红外热像仪 ─ Type=22 ──
	TypeThermalCamera int32 = 22 // 红外热像仪控制（设定焦距/自动聚焦/重启）

	// ── 局放传感器 ─ Type=23 ──
	TypePartialDischarge int32 = 23 // 局放传感器控制（伸长/收缩/停止/复位）

	// ── 任务 ─ Type=41 ──
	TypeTaskControl   int32 = 41 // 任务控制指令（启动/暂停/继续/停止）
	TypeTaskStatusData int32 = 41 // 任务状态数据上报（Type 与指令共用）

	// ── 模型同步 ─ Type=61 ──
	TypeModelSync    int32 = 61 // 模型同步指令（区域主机/机器人/摄像机/点位/无人机/声纹/任务文件/检修/地图/维护/联动/告警阈值）
	TypePatrolResult int32 = 61 // 巡视结果上报（Type 与模型同步共用）

	// ── 告警与确认 ──
	TypeAlarmData       int32 = 62 // 告警数据上报（62-0）
	TypeSilentAlarmData int32 = 63 // 静默监视告警数据上报（63-0）
	TypeAlarmConfirm    int32 = 64 // 巡视告警确认（双向：64-0）
	TypeResultConfirm   int32 = 67 // 巡视结果确认发送（双向：67-0）

	// ── 巡视设备统计 & 检修区域 ─ Type=81 ──
	TypePatrolStatistics int32 = 81 // 巡视设备统计信息上报（81-0）
	TypeMaintenance      int32 = 81 // 检修区域配置（81-4，Type 与统计共用）

	// ── 任务下发 ──
	TypeTaskDispatch        int32 = 101 // 任务下发指令（101-1）
	TypeLinkageTaskDispatch int32 = 102 // 联动任务下发指令（102-1）

	// ── 设备模型下发 ─ Type=110 ──
	TypeDeviceModelDispatch int32 = 110 // 设备模型下发（标准点位/巡视设备/告警阈值模型）

	// ── 巡视结果统计查询 ─ Type=121 ──
	TypeStatisticsQuery int32 = 121 // 巡视结果统计查询（闭环率/审核率/准确率/漏检率）

	// ── 无人机 ──
	TypeDroneBody        int32 = 20001 // 无人机本体控制（保留/系统自检/一键返航/自动降落/模式切换/控制权）
	TypeDroneNestStatus  int32 = 20001 // 无人机机巢状态数据上报（Type 与无人机本体共用）
	TypeDroneNestRunData int32 = 10004 // 无人机机巢运行数据上报
)

// ═══════════════════════════════════════════════════════════════════════════
// Command 枚举 — messageId 低 16 位，标识消息子命令
// ═══════════════════════════════════════════════════════════════════════════

const (
	// ── 系统消息 Command（Type 251）──
	CommandRegister                    int32 = 1 // 注册指令（client→server），服务端回复 251-4 携带心跳间隔
	CommandHeartbeat                   int32 = 2 // 心跳指令（client→server），周期性保活
	CommandGenericResponseWithoutItems int32 = 3 // 通用应答_无Item（server→client），普通消息的通用回复
	CommandGenericResponseWithItems    int32 = 4 // 通用应答_有Item（server→client），注册等需要携带数据的回复

	// ── 上报类 ── 所有 server→client 上报消息的 Command 固定为 0
	CommandReport int32 = 0 // 上报类指令

	// ── 机器人本体控制（Type 1，Command 1~7）──
	CommandRobotRemoteReset    int32 = 1 // 远方复位（1-1）
	CommandRobotSelfCheck      int32 = 2 // 系统自检（1-2）
	CommandRobotReturnHome     int32 = 3 // 一键返航（1-3）
	CommandRobotManualCharge   int32 = 4 // 手动充电（1-4）
	CommandRobotModeSwitch     int32 = 5 // 控制模式切换（1-5）
	CommandRobotTakeControl    int32 = 6 // 控制权获得（1-6）
	CommandRobotReleaseControl int32 = 7 // 控制权释放（1-7）

	// ── 机器人车体控制（Type 2，Command 1~11）──
	CommandChassisForward    int32 = 1  // 前进（2-1）
	CommandChassisBackward   int32 = 2  // 后退（2-2）
	CommandChassisTurnLeft   int32 = 3  // 左转（2-3）
	CommandChassisTurnRight  int32 = 4  // 右转（2-4）
	CommandChassisStop       int32 = 6  // 停止（2-6）
	CommandChassisUp         int32 = 7  // 上升（2-7）
	CommandChassisDown       int32 = 8  // 下降（2-8）
	CommandChassisShiftLeft  int32 = 9  // 左平移（2-9）
	CommandChassisShiftRight int32 = 10 // 右平移（2-10）
	CommandChassisGaitSwitch int32 = 11 // 步态切换（2-11）

	// ── 机器人云台控制（Type 3，Command 1~9）──
	CommandPTZTiltUp   int32 = 1 // 上仰（3-1）
	CommandPTZTiltDown int32 = 2 // 下俯（3-2）
	CommandPTZPanLeft  int32 = 3 // 左转（3-3）
	CommandPTZPanRight int32 = 4 // 右转（3-4）
	CommandPTZRise     int32 = 5 // 上升（3-5）
	CommandPTZLower    int32 = 6 // 下降（3-6）
	CommandPTZPreset   int32 = 7 // 预置位调用（3-7）
	CommandPTZStop     int32 = 8 // 停止（3-8）
	CommandPTZReset    int32 = 9 // 复位（3-9）

	// ── 机器人辅助设备（Type 4，Command 1~5）──
	CommandAuxIRPower    int32 = 1 // 红外电源（4-1）
	CommandAuxWiper      int32 = 2 // 雨刷（4-2）
	CommandAuxUltrasound int32 = 3 // 超声（4-3）
	CommandAuxIRLamp     int32 = 4 // 红外射灯（4-4）
	CommandAuxLighting   int32 = 5 // 辅助照明（4-5）

	// ── 可见光摄像机（Type 21，Command 1~12）──
	CommandVisZoomIn    int32 = 1  // 镜头拉近（21-1）
	CommandVisZoomOut   int32 = 2  // 镜头拉远（21-2）
	CommandVisZoomStop  int32 = 3  // 镜头拉焦停止（21-3）
	CommandVisFocusInc  int32 = 4  // 焦距增加（21-4）
	CommandVisFocusDec  int32 = 5  // 焦距减少（21-5）
	CommandVisAutoFocus int32 = 6  // 自动聚焦（21-6）
	CommandVisReboot    int32 = 8  // 重启（21-8）
	CommandVisZoomSet   int32 = 11 // 倍率值设置（21-11）
	CommandVisFocusSet  int32 = 12 // 聚焦值设置（21-12）

	// ── 红外热像仪（Type 22，Command 5~8）──
	CommandThermalFocusSet   int32 = 5 // 设定焦距值（22-5）
	CommandThermalAutoFocus  int32 = 6 // 自动聚焦（22-6）
	CommandThermalReboot     int32 = 8 // 重启（22-8）

	// ── 局放传感器（Type 23，Command 1~4）──
	CommandPartialDischargeExtend  int32 = 1 // 伸长（23-1）
	CommandPartialDischargeRetract int32 = 2 // 收缩（23-2）
	CommandPartialDischargeStop    int32 = 3 // 停止（23-3）
	CommandPartialDischargeReset   int32 = 4 // 复位（23-4）

	// ── 无人机本体控制（Type 20001，Command 1~6）──
	CommandDroneReserved    int32 = 1 // 保留（20001-1）
	CommandDroneSelfCheck   int32 = 2 // 系统自检（20001-2）
	CommandDroneReturnHome  int32 = 3 // 一键返航（20001-3）
	CommandDroneAutoLand    int32 = 4 // 自动降落（20001-4）
	CommandDroneModeSwitch  int32 = 5 // 控制模式切换（20001-5）
	CommandDroneTakeControl int32 = 6 // 控制权获得（20001-6）

	// ── 任务控制（Type 41，Command 1~4）──
	CommandTaskStart  int32 = 1 // 任务启动（41-1）
	CommandTaskPause  int32 = 2 // 任务暂停（41-2）
	CommandTaskResume int32 = 3 // 任务继续（41-3）
	CommandTaskStop   int32 = 4 // 任务停止（41-4）

	// ── 任务下发（Type 101/102，Command 1）──
	CommandTaskConfig int32 = 1 // 任务配置（101-1 / 102-1）

	// ── 模型同步（Type 61，Command 1~12）──
	CommandModelRegionHost     int32 = 1  // 区域主机及边缘节点装置模型（61-1）
	CommandModelRobot          int32 = 2  // 机器人模型（61-2）
	CommandModelCamera         int32 = 3  // 摄像机模型及硬盘录像机模型（61-3）
	CommandModelPoint          int32 = 4  // 点位模型（61-4）
	CommandModelDrone          int32 = 5  // 无人机模型及无人机机巢模型（61-5）
	CommandModelVoice          int32 = 6  // 声纹模型（61-6）
	CommandModelTaskFile       int32 = 7  // 任务文件（61-7）
	CommandModelMaintenance    int32 = 8  // 检修区域配置文件（61-8）
	CommandModelMap            int32 = 9  // 地图文件（61-9）
	CommandModelMaintRecord    int32 = 10 // 维护记录文件（61-10）
	CommandModelLinkage        int32 = 11 // 联动配置文件（61-11）
	CommandModelAlarmThreshold int32 = 12 // 告警阈值模型（61-12）

	// ── 设备模型下发（Type 110，Command 41~43）──
	CommandDeviceStandardPoint  int32 = 41 // 标准点位模型下发（110-41）
	CommandDevicePatrolModel    int32 = 42 // 巡视设备模型下发（110-42）
	CommandDeviceAlarmThreshold int32 = 43 // 点位告警阈值配置模型下发（110-43）

	// ── 检修区域（Type 81，Command 4）──
	CommandMaintenanceConfig int32 = 4 // 检修区域配置（81-4）

	// ── 巡视结果统计查询（Type 121，Command 1~5）──
	CommandStatClosedLoopRate  int32 = 1 // 巡视任务执行闭环率（121-1）
	CommandStatAuditRate       int32 = 2 // 巡视告警人工审核完成率（121-2）
	CommandStatAlarmAccuracy   int32 = 3 // 巡视告警准确率（121-3）
	CommandStatResultAuditRate int32 = 4 // 巡视结果人工审核完成率（121-4）
	CommandStatMissRate        int32 = 5 // 巡检点位漏检率（121-5）
)

// ═══════════════════════════════════════════════════════════════════════════
// 预计算的 MessageID — 常用消息的 (Type << 16) | Command 值
// 非预计算的消息可通过 EncodeMessageID(Type, Command) 动态计算
// ═══════════════════════════════════════════════════════════════════════════

const (
	// ── 系统消息 ──
	MessageIDRegister                    = int((TypeSystem << 16) | CommandRegister)                    // 注册指令（251-1）
	MessageIDHeartbeat                   = int((TypeSystem << 16) | CommandHeartbeat)                   // 心跳指令（251-2）
	MessageIDGenericResponseWithoutItems = int((TypeSystem << 16) | CommandGenericResponseWithoutItems) // 通用应答_无Item（251-3）
	MessageIDGenericResponseWithItems    = int((TypeSystem << 16) | CommandGenericResponseWithItems)    // 通用应答_有Item（251-4）

	// ── 巡视设备上报（Command=0）──
	MessageIDPatrolDeviceStatusData  = int((TypePatrolDeviceStatusData << 16) | CommandReport)  // 巡视设备状态数据（1-0）
	MessageIDPatrolDeviceRunData     = int((TypePatrolDeviceRunData << 16) | CommandReport)     // 巡视设备运行数据（2-0）
	MessageIDPatrolDeviceCoordinates = int((TypePatrolDeviceCoordinates << 16) | CommandReport) // 巡视设备坐标（3-0）
	MessageIDPatrolRoute             = int((TypePatrolRoute << 16) | CommandReport)             // 巡视路线（4-0）
	MessageIDPatrolDeviceAlarm       = int((TypePatrolDeviceAlarm << 16) | CommandReport)       // 巡视设备异常告警数据（5-0）
	MessageIDModelUpdateReport       = int((TypeModelUpdateReport << 16) | CommandReport)       // 模型更新上报（11-0）
	MessageIDEnvData                 = int((TypeEnvData << 16) | CommandReport)                 // 环境数据（21-0）
	MessageIDTaskStatusData          = int((TypeTaskStatusData << 16) | CommandReport)          // 任务状态数据（41-0）
	MessageIDPatrolResult            = int((TypePatrolResult << 16) | CommandReport)            // 巡视结果（61-0）
	MessageIDAlarmData               = int((TypeAlarmData << 16) | CommandReport)               // 告警数据（62-0）
	MessageIDSilentAlarmData         = int((TypeSilentAlarmData << 16) | CommandReport)         // 静默监视告警数据（63-0）
	MessageIDPatrolStatistics        = int((TypePatrolStatistics << 16) | CommandReport)        // 巡视设备统计信息上报（81-0）
	MessageIDDroneNestStatus         = int((TypeDroneNestStatus << 16) | CommandReport)         // 无人机机巢状态数据（20001-0）
	MessageIDDroneNestRunData        = int((TypeDroneNestRunData << 16) | CommandReport)        // 无人机机巢运行数据（10004-0）

	// ── 任务下发 ──
	MessageIDTaskDispatch        = int((TypeTaskDispatch << 16) | CommandTaskConfig)        // 任务下发指令_任务配置（101-1）
	MessageIDLinkageTaskDispatch = int((TypeLinkageTaskDispatch << 16) | CommandTaskConfig) // 联动任务下发指令_任务配置（102-1）
)

// EncodeMessageID 将 Type（高16位）和 Command（低16位）编码为 32 位 messageId。
// 对标 Java TSip.encode(type, command)。
func EncodeMessageID(typ, command int32) int {
	return int((typ << 16) | command)
}

// DecodeMessageID 从 32 位 messageId 中解码出 Type 和 Command。
// 对标 Java TSip.decode(code)。
func DecodeMessageID(messageID int) (typ, command int32) {
	return int32(messageID >> 16), int32(messageID & 0xffff)
}

// NormalizeRootName 校验 XML 根元素名称，不合法时回退到默认的 PatrolDevice。
func NormalizeRootName(root string) string {
	switch root {
	case RootPatrolHost, RootPatrolDevice:
		return root
	default:
		return RootPatrolDevice
	}
}
