package modbusx

import "testing"

func TestWithDefaults(t *testing.T) {
	conf := (&ModbusClientConf{}).withDefaults()
	if conf.Slave != defaultSlave {
		t.Fatalf("Slave = %d, want %d", conf.Slave, defaultSlave)
	}
	if conf.Timeout != defaultTimeoutMs {
		t.Fatalf("Timeout = %d, want %d", conf.Timeout, defaultTimeoutMs)
	}
	if conf.IdleTimeout != defaultIdleTimeoutMs {
		t.Fatalf("IdleTimeout = %d, want %d", conf.IdleTimeout, defaultIdleTimeoutMs)
	}
	if conf.LinkRecoveryTimeout != defaultLinkRecoveryTimeoutMs {
		t.Fatalf("LinkRecoveryTimeout = %d, want %d", conf.LinkRecoveryTimeout, defaultLinkRecoveryTimeoutMs)
	}
	if conf.ProtocolRecoveryTimeout != defaultProtocolRecoveryTimeoutMs {
		t.Fatalf("ProtocolRecoveryTimeout = %d, want %d", conf.ProtocolRecoveryTimeout, defaultProtocolRecoveryTimeoutMs)
	}
	if conf.ConnectDelay != defaultConnectDelayMs {
		t.Fatalf("ConnectDelay = %d, want %d", conf.ConnectDelay, defaultConnectDelayMs)
	}

	custom := (&ModbusClientConf{
		Slave:                   5,
		Timeout:                 100,
		IdleTimeout:             200,
		LinkRecoveryTimeout:     300,
		ProtocolRecoveryTimeout: 400,
		ConnectDelay:            500,
	}).withDefaults()
	if custom.Slave != 5 || custom.Timeout != 100 ||
		custom.IdleTimeout != 200 || custom.LinkRecoveryTimeout != 300 ||
		custom.ProtocolRecoveryTimeout != 400 || custom.ConnectDelay != 500 {
		t.Fatalf("withDefaults() overrode explicit values: %+v", custom)
	}
}

func TestWithDefaultsPreservesNegativeIdleTimeout(t *testing.T) {
	// grid-x 用负值表示空闲连接永不关闭，兜底逻辑不能把它改成正数。
	conf := (&ModbusClientConf{IdleTimeout: -1}).withDefaults()
	if conf.IdleTimeout != -1 {
		t.Fatalf("IdleTimeout = %d, want -1", conf.IdleTimeout)
	}
}

func TestWithDefaultsNilReceiver(t *testing.T) {
	var conf *ModbusClientConf
	if got := conf.withDefaults(); got == nil || got.Timeout != defaultTimeoutMs {
		t.Fatalf("nil withDefaults() = %+v, want default timeout", got)
	}
}

func TestWithDefaultsPreservesDisabledRecoveryAndDelay(t *testing.T) {
	conf := (&ModbusClientConf{
		LinkRecoveryTimeout: -1, ProtocolRecoveryTimeout: -1, ConnectDelay: -1,
	}).withDefaults()
	if conf.LinkRecoveryTimeout != -1 || conf.ProtocolRecoveryTimeout != -1 || conf.ConnectDelay != -1 {
		t.Fatalf("explicit disable values overridden: %+v", conf)
	}
}

func TestWithDefaultsClampsTimeout(t *testing.T) {
	// 毫秒值过大时 time.Duration 会溢出成负数，必须截断到上限。
	conf := (&ModbusClientConf{Timeout: 9_000_000_000_000_000}).withDefaults()
	if conf.Timeout != maxTimeoutMs {
		t.Fatalf("Timeout = %d, want %d", conf.Timeout, maxTimeoutMs)
	}

	// 上限内的显式值保持不变。
	if got := (&ModbusClientConf{Timeout: 30000}).withDefaults().Timeout; got != 30000 {
		t.Fatalf("Timeout = %d, want 30000", got)
	}
}
