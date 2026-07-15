package isp

import (
	"fmt"
	"strconv"

	"github.com/dromara/carbon/v2"
)

// Item 为动态 key-value 属性容器，对应 XML 中 <Item attr="value"/> 的属性映射。
type Item map[string]string

// Message 为 ISP 协议消息，同时实现 gnetx 的消息路由和请求-响应接口。
//
// 协议帧中 sendSerialNo / receiveSerialNo 的语义：
//   - SendSeq（sendSerialNo）：本端自增发送序号，每次发消息时取 NextSendSeq()
//   - RecvSeq（receiveSerialNo）：对端回执序号 = 上次从对端收到的 SendSeq，类似于 TCP ACK
//
// 响应匹配：服务端回复时将其 receiveSerialNo 设为本端请求的 SendSeq；
// gnetx 通过 TID()=SendSeq 注册、ResponseTID()=RecvSeq 解回完成请求-响应关联。
type Message struct {
	RootName      string // XML 根元素（PatrolHost 或 PatrolDevice），可配置
	SendSeq       uint64 // 本端发送序号 sendSerialNo（8 字节小端）
	RecvSeq       uint64 // 对端回执序号 receiveSerialNo（8 字节小端）= 上次收到的对端 SendSeq
	SessionSource byte   // 会话源：0x00=客户端，0x01=服务端

	SendCode    string // 发送方唯一标识
	ReceiveCode string // 接收方唯一标识（注册后从服务端学习）
	Type        int32  // 消息类型（高 16 位 messageId）
	Code        string // 目标对象唯一标识，含义随消息类型变化：变电站编码/任务编码/巡视ID 等
	Command     int32  // 命令（低 16 位 messageId）
	Time        string // 时间戳
	Items       []Item // 业务数据列表

	RawXML string // 原始 XML，用于诊断
}

// MessageID 返回 messageId = (Type << 16) | Command，供 gnetx.Router 路由。
func (m *Message) MessageID() int {
	return EncodeMessageID(m.Type, m.Command)
}

// MessageName 返回消息的中文名称，用于日志输出。
func (m *Message) MessageName() string {
	switch m.MessageID() {
	// 系统消息 (Type 251)
	case MessageIDRegister:
		return "注册指令(251-1)"
	case MessageIDHeartbeat:
		return "心跳指令(251-2)"
	case MessageIDGenericResponseWithoutItems:
		return "通用应答(251-3)"
	case MessageIDGenericResponseWithItems:
		return "注册应答(251-4)"

	// 巡视设备上报
	case MessageIDPatrolDeviceStatusData:
		return "巡视设备状态数据(1-0)"
	case MessageIDPatrolDeviceRunData:
		return "巡视设备运行数据(2-0)"
	case MessageIDPatrolDeviceCoordinates:
		return "巡视设备坐标(3-0)"
	case MessageIDPatrolRoute:
		return "巡视路线(4-0)"
	case MessageIDPatrolDeviceAlarm:
		return "巡视设备异常告警(5-0)"

	// 模型更新
	case MessageIDModelUpdateReport:
		return "模型更新上报(11-0)"
	case MessageIDEnvData:
		return "环境数据(21-0)"
	case MessageIDTaskStatusData:
		return "任务状态数据(41-0)"
	case MessageIDPatrolResult:
		return "巡视结果(61-0)"
	case MessageIDAlarmData:
		return "告警数据(62-0)"
	case MessageIDSilentAlarmData:
		return "静默监视告警(63-0)"
	case MessageIDPatrolStatistics:
		return "巡视设备统计上报(81-0)"
	case MessageIDDroneNestStatus:
		return "无人机机巢状态(20001-0)"
	case MessageIDDroneNestRunData:
		return "无人机机巢运行(10004-0)"

	// 任务下发
	case MessageIDTaskDispatch:
		return "任务下发_任务配置(101-1)"
	case MessageIDLinkageTaskDispatch:
		return "联动任务下发_任务配置(102-1)"
	}
	return fmt.Sprintf("%d-%d", m.Type, m.Command)
}

// TID 返回请求关联 ID（用 SendSeq），供 gnetx.Request 进行请求-响应匹配。
func (m *Message) TID() string {
	return strconv.FormatUint(m.SendSeq, 10)
}

// ResponseTID 返回实际应答消息的回包关联 ID（用 RecvSeq），供 gnetx 匹配在途请求。
// 服务端回复时将 receiveSerialNo 设为本端请求的 SendSeq，实现回包匹配。
func (m *Message) ResponseTID() string {
	switch m.MessageID() {
	case MessageIDGenericResponseWithoutItems, MessageIDGenericResponseWithItems:
		return strconv.FormatUint(m.RecvSeq, 10)
	default:
		return ""
	}
}

// EnsureDefaults 填充默认根元素和会话源。
func (m *Message) EnsureDefaults(rootName string) {
	m.RootName = NormalizeRootName(firstNonEmpty(m.RootName, rootName))
	if m.SessionSource == 0 {
		m.SessionSource = SessionSourceClient
	}
}

// NewResponse 基于请求消息构造 ISP 系统应答，自动处理 SendCode/ReceiveCode 互换和 RecvSeq 回执。
// sessionSource: 本端会话源（SessionSourceClient 或 SessionSourceServer）
// code: 应答状态码（200/400/500）
// command: 应答指令（CommandGenericResponseWithoutItems 或 CommandGenericResponseWithItems）
// items: 可选业务数据
//
// 调用方需填充 SendSeq（从 conn.NextSendSeq()）并可覆盖 RootName。
func NewResponse(req *Message, sessionSource byte, code string, command int32, items []Item) *Message {
	return &Message{
		RootName:      req.RootName,
		SessionSource: sessionSource,
		SendCode:      req.ReceiveCode,
		ReceiveCode:   req.SendCode,
		Type:          TypeSystem,
		Code:          code,
		Command:       command,
		Time:          carbon.Now().ToDateTimeString(),
		RecvSeq:       req.SendSeq,
		Items:         items,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
