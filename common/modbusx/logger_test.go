package modbusx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestClassifyModbusLog(t *testing.T) {
	tests := []struct {
		name   string
		format string
		op     string
		desc   string
		level  modbusLogLevel
	}{
		{"send frame", "modbus: send % x", "send_frame", "发送请求帧", modbusLogDebug},
		{"recv frame", "modbus: recv % x\n", "recv_frame", "接收响应帧", modbusLogDebug},
		{"idle close", "modbus: closing connection due to idle timeout: %v", "idle_close", "空闲超时关闭连接", modbusLogDebug},
		{"write error", "modbus: write error, closing connection: %v", "write_error", "写入请求失败，关闭连接", modbusLogError},
		{"read error", "modbus: read response error: %v, reconnect: %v", "read_error", "读取响应失败", modbusLogInfo},
		{"read error closing", "modbus: read response error: closing connection: %v", "read_error_closing", "读取响应失败，关闭连接", modbusLogError},
		{"retry reissue", "modbus: close connection and retry reading response, because of %v", "retry_reissue", "关闭连接并重试读取响应", modbusLogInfo},
		{"read timeout", "modbus: read response timeout, keeping connection: %v", "read_timeout", "读取响应超时，保留连接", modbusLogInfo},
		{"remote closed", "modbus: connection closed by remote side: %v", "remote_closed", "对端已关闭连接", modbusLogInfo},
		{"verify error", "modbus: verify error: %v", "verify_error", "响应帧校验失败", modbusLogError},
		{"unknown", "modbus: brand new message from a future release: %v", "unknown", "未识别的 Modbus 事件", modbusLogInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyModbusLog(tt.format)
			if got != (modbusLogKind{op: tt.op, desc: tt.desc, level: tt.level}) {
				t.Fatalf("classifyModbusLog(%q) = %+v, want op=%q desc=%q level=%d",
					tt.format, got, tt.op, tt.desc, tt.level)
			}
		})
	}
}

func TestModbusLoggerFrameLogIsDebugOnly(t *testing.T) {
	logFrame := func(l *ModbusLogger) {
		l.Printf("modbus: send % x", []byte{0x00, 0x01, 0xFF})
	}

	if out := captureModbusLog(t, logx.InfoLevel, logFrame); strings.Contains(out, "send 00 01 ff") {
		t.Fatalf("帧日志不应在 info 级别输出: %s", out)
	}

	out := captureModbusLog(t, logx.DebugLevel, logFrame)
	if !strings.Contains(out, "发送请求帧") {
		t.Fatalf("帧日志缺少中文描述: %s", out)
	}
	if !strings.Contains(out, "send 00 01 ff") {
		t.Fatalf("帧日志缺少原始帧内容: %s", out)
	}
}

func TestModbusLoggerConnectionEventHasFields(t *testing.T) {
	out := captureModbusLog(t, logx.InfoLevel, func(l *ModbusLogger) {
		l.Printf("modbus: write error, closing connection: %v", errors.New("broken pipe"))
	})

	wants := []string{
		"写入请求失败，关闭连接",
		"write error, closing connection: broken pipe",
		`"transport":"tcp"`,
		`"address":"127.0.0.1:502"`,
		`"slave":1`,
		`"clientId":"client-1"`,
		`"op":"write_error"`,
		`"component":"modbus"`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("日志缺少 %q: %s", want, out)
		}
	}
}

func TestModbusLogFieldsAddressMD5(t *testing.T) {
	// 连接级字段已在日志输出用例中断言，这里只校验脱敏字段的取值。
	fields := modbusLogFields(&ModbusClientConf{Address: "127.0.0.1:502", Slave: 7}, "client-1")
	for _, field := range fields {
		if field.Key != "addressMd5" {
			continue
		}
		if want := cryptor.Md5String("127.0.0.1:502"); field.Value != want {
			t.Fatalf("addressMd5 = %v, want %s", field.Value, want)
		}
		return
	}
	t.Fatal("missing addressMd5 field")
}

func TestModbusLoggerKeepsArgsAfterError(t *testing.T) {
	out := captureModbusLog(t, logx.InfoLevel, func(l *ModbusLogger) {
		l.Printf("modbus: read response error: %v, reconnect: %v", errors.New("EOF"), true)
	})

	if !strings.Contains(out, "read response error: EOF, reconnect: true") {
		t.Fatalf("error 参数之后的参数被丢弃: %s", out)
	}
}

func TestModbusLoggerErrorFields(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		out := captureModbusLog(t, logx.InfoLevel, func(l *ModbusLogger) {
			l.Printf("modbus: read response timeout, keeping connection: %v", context.DeadlineExceeded)
		})
		for _, want := range []string{`"timeout":true`, `"errDesc":"读写或建连超时"`} {
			if !strings.Contains(out, want) {
				t.Fatalf("日志缺少 %q: %s", want, out)
			}
		}
	})

	t.Run("desynced", func(t *testing.T) {
		out := captureModbusLog(t, logx.InfoLevel, func(l *ModbusLogger) {
			l.Printf("modbus: read response error: %v, reconnect: %v", modbus.ErrStreamDesynced, false)
		})
		for _, want := range []string{`"desynced":true`, `"errDesc":"响应流与请求失配"`} {
			if !strings.Contains(out, want) {
				t.Fatalf("日志缺少 %q: %s", want, out)
			}
		}
	})

	t.Run("exception", func(t *testing.T) {
		out := captureModbusLog(t, logx.InfoLevel, func(l *ModbusLogger) {
			l.Printf("modbus: verify error: %v", &modbus.Error{
				FunctionCode:  0x83,
				ExceptionCode: modbus.ExceptionCodeIllegalDataAddress,
			})
		})
		wants := []string{
			`"functionCode":131`,
			`"exceptionCode":2`,
			`"exceptionMessage":"非法数据地址"`,
			`"errDesc":"设备返回异常响应：非法数据地址"`,
		}
		for _, want := range wants {
			if !strings.Contains(out, want) {
				t.Fatalf("日志缺少 %q: %s", want, out)
			}
		}
	})
}

// Level=error 时仍需保留确认失败的传输错误日志，可恢复事件保持 Info。
func TestModbusLoggerKeepsTransportErrorsAtErrorLevel(t *testing.T) {
	out := captureModbusLog(t, logx.ErrorLevel, func(l *ModbusLogger) {
		l.Printf("modbus: write error, closing connection: %v", errors.New("broken pipe"))
		l.Printf("modbus: read response timeout, keeping connection: %v", context.DeadlineExceeded)
	})

	if !strings.Contains(out, "写入请求失败，关闭连接") {
		t.Fatalf("error 级别丢失传输错误: %s", out)
	}
	if strings.Contains(out, "读取响应超时，保留连接") {
		t.Fatalf("可恢复事件不应在 error 级别输出: %s", out)
	}
}

// captureModbusLog 把 logx 输出重定向到缓冲区，执行 fn 后返回日志文本。
func captureModbusLog(t *testing.T, level uint32, fn func(l *ModbusLogger)) string {
	t.Helper()

	var buf bytes.Buffer
	logx.SetLevel(level)
	logx.SetWriter(logx.NewWriter(&buf))
	t.Cleanup(func() {
		logx.Reset()
		logx.SetLevel(logx.DebugLevel)
	})

	fn(newModbusLogger(&ModbusClientConf{Address: "127.0.0.1:502", Slave: 1}, "client-1"))

	return buf.String()
}
