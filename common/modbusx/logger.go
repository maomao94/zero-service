package modbusx

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/zeromicro/go-zero/core/logx"
)

// modbusLogLevel 是归类后使用的日志级别。
type modbusLogLevel int

const (
	// modbusLogDebug 用于帧级 dump 与常态事件，默认级别下不输出。
	modbusLogDebug modbusLogLevel = iota
	// modbusLogInfo 用于连接事件与可恢复的通信抖动。
	modbusLogInfo
	// modbusLogError 用于确认失败的传输错误，保证 Level=error 时仍可见。
	modbusLogError
)

// modbusLogKind 是 grid-x 一条日志文案的归类结果。
type modbusLogKind struct {
	op    string         // op 字段值，用于按事件类型检索
	desc  string         // 中文事件描述，作为日志正文
	level modbusLogLevel // 输出级别
}

// modbusLogKinds 把 grid-x 的日志文案映射到中文描述、op 与级别。
//
// 文案集合来自 github.com/grid-x/modbus v1.5.1 tcpclient.go 中全部 mb.logf 调用点，
// SDK 原始文案保留在 detail 字段，便于对照源码定位。
// 匹配顺序敏感：更长的前缀必须排在前面，例如
// "read response error: closing connection" 要早于 "read response error:"。
// 未匹配的文案兜底为 unknown + Info，避免 SDK 升级后日志静默丢失。
var modbusLogKinds = []struct {
	prefix string
	kind   modbusLogKind
}{
	{"modbus: read response error: closing connection", modbusLogKind{op: "read_error_closing", desc: "读取响应失败，关闭连接", level: modbusLogError}},
	{"modbus: read response error:", modbusLogKind{op: "read_error", desc: "读取响应失败", level: modbusLogInfo}},
	{"modbus: close connection and retry reading response", modbusLogKind{op: "retry_reissue", desc: "关闭连接并重试读取响应", level: modbusLogInfo}},
	{"modbus: read response timeout, keeping connection", modbusLogKind{op: "read_timeout", desc: "读取响应超时，保留连接", level: modbusLogInfo}},
	{"modbus: write error, closing connection", modbusLogKind{op: "write_error", desc: "写入请求失败，关闭连接", level: modbusLogError}},
	{"modbus: connection closed by remote side", modbusLogKind{op: "remote_closed", desc: "对端已关闭连接", level: modbusLogInfo}},
	{"modbus: verify error", modbusLogKind{op: "verify_error", desc: "响应帧校验失败", level: modbusLogError}},
	{"modbus: closing connection due to idle timeout", modbusLogKind{op: "idle_close", desc: "空闲超时关闭连接", level: modbusLogDebug}},
	{"modbus: send", modbusLogKind{op: "send_frame", desc: "发送请求帧", level: modbusLogDebug}},
	{"modbus: recv", modbusLogKind{op: "recv_frame", desc: "接收响应帧", level: modbusLogDebug}},
}

// classifyModbusLog 按日志格式串归类事件类型与级别。
func classifyModbusLog(format string) modbusLogKind {
	for _, entry := range modbusLogKinds {
		if strings.HasPrefix(format, entry.prefix) {
			return entry.kind
		}
	}

	return modbusLogKind{op: "unknown", desc: "未识别的 Modbus 事件", level: modbusLogInfo}
}

// ModbusLogger 把 grid-x 的 Printf 日志适配到 go-zero logx。
//
// grid-x 的 Logger 接口既没有级别也没有 ctx，因此这里约定：
//   - 日志正文固定为中文事件描述，SDK 原始文案放进 detail 字段，便于对照源码定位；
//   - 帧级 dump 与空闲关闭走 Debug，默认级别下不输出，避免每事务两条全帧日志；
//   - 可恢复事件（迟到响应、对端断链、只读重发）走 Info；
//   - 确认失败的传输错误（写失败、读失败后关闭连接、响应校验失败）走 Error，
//     保证把日志级别调到 warn/error 时连接故障仍然可见。
type ModbusLogger struct {
	logger logx.Logger
}

// transportTCP 是当前唯一的连接类型。
// 新增 RTU/串口传输时，只需在 modbusLogFields 里按配置补充 transport 取值
// 与串口参数（baud/dataBits/parity/stopBits），调用方无需改动。
const transportTCP = "tcp"

// modbusLogFields 返回连接级日志字段，TCP 与后续 RTU 共用：
//   - transport：连接类型，区分 TCP 与串口
//   - address：TCP 为 IP:Port，RTU 为串口设备路径
//   - addressMd5：address 的 MD5，用于脱敏检索
//   - slave：从站号，RTU 总线上同一串口挂多个从站时据此区分设备
//   - clientId：连接实例标识，区分同一设备的并发连接
func modbusLogFields(conf *ModbusClientConf, clientID string) []logx.LogField {
	return []logx.LogField{
		logx.Field("component", "modbus"),
		logx.Field("transport", transportTCP),
		logx.Field("address", conf.Address),
		logx.Field("addressMd5", cryptor.Md5String(conf.Address)),
		logx.Field("slave", conf.Slave),
		logx.Field("clientId", clientID),
	}
}

// newModbusLogger 构建带连接标识的日志器，连接字段与 addressMd5 只在构造时计算。
func newModbusLogger(conf *ModbusClientConf, clientID string) *ModbusLogger {
	ctx := logx.ContextWithFields(context.Background(), modbusLogFields(conf, clientID)...)

	return &ModbusLogger{logger: logx.WithContext(ctx)}
}

// Printf 实现 modbus.Logger 接口。
func (l *ModbusLogger) Printf(format string, v ...any) {
	kind := classifyModbusLog(format)

	// 统一去掉 SDK 的 "modbus: " 前缀与结尾换行。
	// 必须整体格式化：error 参数之后的参数（如 reconnect 标志、响应帧）不能丢。
	detail := strings.TrimRight(fmt.Sprintf(format, v...), "\n")
	detail = strings.TrimPrefix(detail, "modbus: ")

	fields := make([]logx.LogField, 0, 6)
	fields = append(fields,
		logx.Field("op", kind.op),
		logx.Field("detail", detail),
	)
	fields = append(fields, errFields(v)...)

	switch kind.level {
	case modbusLogDebug:
		l.logger.Debugw(kind.desc, fields...)
	case modbusLogError:
		l.logger.Errorw(kind.desc, fields...)
	default:
		l.logger.Infow(kind.desc, fields...)
	}
}

// errFields 从 SDK 参数中提取结构化错误信息，便于检索与告警区分。
func errFields(v []any) []logx.LogField {
	err := firstError(v)
	if err == nil {
		return nil
	}

	var (
		fields []logx.LogField
		desc   string
	)
	if IsTimeout(err) {
		fields = append(fields, logx.Field("timeout", true))
		desc = "读写或建连超时"
	}
	if IsStreamDesynced(err) {
		fields = append(fields, logx.Field("desynced", true))
		desc = "响应流与请求失配"
	}
	if functionCode, exceptionCode, ok := Exception(err); ok {
		fields = append(fields,
			logx.Field("functionCode", functionCode),
			logx.Field("exceptionCode", exceptionCode),
			logx.Field("exceptionMessage", ExceptionMessage(exceptionCode)),
		)
		desc = "设备返回异常响应：" + ExceptionMessage(exceptionCode)
	}
	if desc != "" {
		fields = append(fields, logx.Field("errDesc", desc))
	}

	return fields
}

// firstError 返回参数里的第一个非 nil error。
func firstError(v []any) error {
	for _, val := range v {
		if err, ok := val.(error); ok && err != nil {
			return err
		}
	}

	return nil
}
