package modbusx

// 连接配置的安全缺省值。
//
// 这些缺省值主要覆盖两类场景：
//  1. 数据库历史行或 YAML 漏配，结构体字段为 0；
//  2. grid-x 对部分字段的 0 值有特殊语义（例如 IdleTimeout=0 表示每次请求后关闭连接）。
//
// 所有时间字段单位均为毫秒。
const (
	// MinSlave 是最小单播从站地址（Unit ID）。0 为广播地址，读写请求不应使用。
	MinSlave = 1

	// MaxSlave 是最大单播从站地址。248-255 为保留地址，读写请求不应发往这些地址。
	MaxSlave = 247

	// defaultSlave 缺省从站地址（Unit ID）。多从站网关下必须与设备实际配置一致。
	defaultSlave = MinSlave

	// defaultTimeoutMs 缺省读写/建连超时。
	// 必须兜底：grid-x 中 Timeout=0 表示不设置读写 deadline，设备无响应时
	// 请求会永久占住串行客户端，阻塞后续采集。
	defaultTimeoutMs = 10000

	// maxTimeoutMs 是 Timeout 上限（10 分钟）。
	// 超过该值按上限处理：毫秒转 time.Duration 时溢出会变成负值，
	// 使 SDK 不设置读写 deadline、Dialer 不设超时，单次请求可能永久占住客户端。
	maxTimeoutMs = 600000

	// defaultIdleTimeoutMs 缺省空闲断链时间。
	// 与 SDK 一致：空闲 60 秒后关闭 TCP，下次请求自动重连；保持连接可显式设 -1。
	defaultIdleTimeoutMs = 60000

	// defaultLinkRecoveryTimeoutMs 缺省链路恢复窗口。
	// 连接被对端关闭（EOF/RST）后的恢复窗口，窗口内可安全重连并重发只读请求；
	// 0 和负数都关闭恢复；写请求不自动重发，失败后应核实设备状态。
	defaultLinkRecoveryTimeoutMs = 0

	// defaultProtocolRecoveryTimeoutMs 缺省协议异常恢复窗口。
	// 收到迟到响应、事务 ID 不匹配等协议异常时，在该窗口内继续读取并丢弃迟到帧；
	// 0 和负数都关闭协议恢复。
	defaultProtocolRecoveryTimeoutMs = 0

	// defaultConnectDelayMs 缺省建连后静默时间。
	// 默认不额外等待；仅对明确要求建连静默期的网关配置正值。
	defaultConnectDelayMs = 0
)

// ModbusClientConf 是单个长期 Modbus TCP 客户端的配置。
//
// 所有字段都来自数据库中的 ModbusSlaveConfig 或服务 YAML。
// 数据库转换（model/gormmodel.ToClientConf）不会应用结构体 tag 上的 default，
// 因此 withDefaults 是最后一道兜底，调用方无需自行补零值。
type ModbusClientConf struct {
	// Address 是 TCP 设备地址，格式为 IP:Port（例如 192.168.1.10:502）。
	// 无缺省值，必须由配置提供；不同 modbusCode 各自缓存 Client，不共享 TCP。
	Address string `json:"address"`

	// Slave 是 Modbus 从站地址（Slave ID / Unit ID），取值范围 1-247。
	// 缺省 1。它决定一条 TCP 连接上默认访问哪个从站，由 grid-x handler 在发送前写入。
	Slave int64 `json:"slave,default=1"`

	// Timeout 是发送/接收超时，也是建连超时，单位毫秒，缺省 10000。
	// 它同时作用于 net.Dialer.Timeout 和每次请求的 conn deadline；
	// 这是单次 socket 读写的超时，不包含排队、建连静默期和链路恢复的额外耗时。
	Timeout int64 `json:"timeout,default=10000"`

	// IdleTimeout 是空闲连接自动关闭时间，单位毫秒，缺省 60000，与 SDK 一致。
	// grid-x 在连接空闲超过该值后关闭 TCP，下次请求重新建链；
	// 0 使用缺省值，负数表示 SDK 不主动按空闲超时断链；网关仍可能断开连接。
	IdleTimeout int64 `json:"idleTimeout,default=60000"`

	// LinkRecoveryTimeout 是链路恢复窗口，单位毫秒，缺省 0，与 SDK 一致（关闭恢复）。
	// 连接被对端关闭（EOF/ECONNRESET）后，在该时间窗口内自动重连并重发只读请求；
	// 0 或负数关闭自动恢复；写请求不自动重发。
	// 负值使本次 Send 的恢复截止时间立即过期，故障直接返回；下次调用仍可重新拨号。
	LinkRecoveryTimeout int64 `json:"linkRecoveryTimeout,optional"`

	// ProtocolRecoveryTimeout 是协议异常恢复窗口，单位毫秒，缺省 0（关闭恢复）。
	// 收到迟到响应或事务 ID 不匹配的帧时，在该窗口内继续读取并丢弃迟到帧，
	// 避免把一次时序抖动误判成通信故障；0 和负数都关闭协议恢复。
	ProtocolRecoveryTimeout int64 `json:"protocolRecoveryTimeout,optional"`

	// ConnectDelay 是 TCP 建连成功后的静默等待时间，单位毫秒，缺省 0（不等待）。
	// 部分串口网关在连接初始化期间会丢弃首帧，留出短暂静默期可提高首次请求成功率；
	// 0 和负数都不等待。
	ConnectDelay int64 `json:"connectDelay,optional"`

	// TLS 配置，如果不需要可保持 Enable: false。
	// 启用后使用 CertFile/KeyFile 做双向认证，CAFile 用于校验服务端证书。
	TLS struct {
		Enable   bool   `json:"enable"`
		CertFile string `json:"certFile"`
		KeyFile  string `json:"keyFile"`
		CAFile   string `json:"caFile"`
	} `json:"tls,optional"`
}

// withDefaults 返回补齐安全缺省值后的配置副本，不修改调用方传入的配置。
// Timeout=10000、IdleTimeout=60000、恢复窗口=0、ConnectDelay=0 参考 SDK 默认值。
// 项目额外默认 Slave=1；保持连接可设 IdleTimeout=-1，需要迟到响应恢复可设
// ProtocolRecoveryTimeout=2000，需要只读断链恢复可设 LinkRecoveryTimeout=3000。
//
// Slave/Timeout 非正值使用缺省值，Timeout 超过 maxTimeoutMs 时按上限截断，
// 避免毫秒转 time.Duration 溢出成负值后不再设置任何超时。
// 其余时间字段仅对零值兜底：
//   - IdleTimeout 为负表示空闲连接永不关闭；
//   - LinkRecoveryTimeout / ProtocolRecoveryTimeout 为负等价于关闭恢复；
//   - ConnectDelay 为负会在 time.After 中立即返回，等价于 0。
func (c *ModbusClientConf) withDefaults() *ModbusClientConf {
	if c == nil {
		c = &ModbusClientConf{}
	}
	cp := *c
	if cp.Slave <= 0 {
		cp.Slave = defaultSlave
	}
	if cp.Timeout <= 0 {
		cp.Timeout = defaultTimeoutMs
	}
	if cp.Timeout > maxTimeoutMs {
		cp.Timeout = maxTimeoutMs
	}
	if cp.IdleTimeout == 0 {
		cp.IdleTimeout = defaultIdleTimeoutMs
	}
	if cp.LinkRecoveryTimeout == 0 {
		cp.LinkRecoveryTimeout = defaultLinkRecoveryTimeoutMs
	}
	if cp.ProtocolRecoveryTimeout == 0 {
		cp.ProtocolRecoveryTimeout = defaultProtocolRecoveryTimeoutMs
	}
	if cp.ConnectDelay == 0 {
		cp.ConnectDelay = defaultConnectDelayMs
	}
	return &cp
}
