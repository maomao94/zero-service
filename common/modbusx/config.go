package modbusx

import (
	"context"
	"fmt"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

// Standard Modbus Device Identification object IDs
const (
	DeviceIDVendorName          byte = 0x00 // 厂商名称
	DeviceIDProductCode         byte = 0x01 // 产品代码
	DeviceIDMajorMinorRevision  byte = 0x02 // 版本号 (Major.Minor)
	DeviceIDVendorURL           byte = 0x03 // 厂商网址
	DeviceIDProductName         byte = 0x04 // 产品名称
	DeviceIDModelName           byte = 0x05 // 型号名称
	DeviceIDUserApplicationName byte = 0x06 // 用户应用名称
)

var DeviceIDObjectNames = map[byte]string{
	DeviceIDVendorName:          "VendorName",
	DeviceIDProductCode:         "ProductCode",
	DeviceIDMajorMinorRevision:  "MajorMinorRevision",
	DeviceIDVendorURL:           "VendorURL",
	DeviceIDProductName:         "ProductName",
	DeviceIDModelName:           "ModelName",
	DeviceIDUserApplicationName: "UserApplicationName",
}

type ModbusClientConf struct {
	// TCP 设备地址，格式 IP:Port
	Address string `json:"address"`

	// Modbus 从站地址（Slave ID / Unit ID）
	Slave int64 `json:"slave,default=1"`

	// 发送/接收超时，单位毫秒
	Timeout int64 `json:"timeout,default=10000"`

	// 空闲连接自动关闭时间，单位毫秒
	IdleTimeout int64 `json:"idleTimeout,default=60000"`

	// TCP 连接出错后的重连间隔，单位毫秒
	LinkRecoveryTimeout int64 `json:"linkRecoveryTimeout,default=3000"`

	// 协议异常时的重试间隔，单位毫秒
	ProtocolRecoveryTimeout int64 `json:"protocolRecoveryTimeout,default=2000"`

	// 连接建立后等待时间，避免立即发送请求，单位毫秒
	ConnectDelay int64 `json:"connectDelay,default=100"`

	// TLS 配置，如果不需要可保持 Enable: false
	TLS struct {
		Enable   bool   `json:"enable"`
		CertFile string `json:"certFile"`
		KeyFile  string `json:"keyFile"`
		CAFile   string `json:"caFile"`
	} `json:"tls,optional"`
}

// PoolManager 连接池管理器：管理多个 Modbus 连接池（按 modbusCode 区分）
type PoolManager struct {
	pools   map[string]*ModbusClientPool // key: modbusCode，包内私有
	confMap map[string]*ModbusClientConf // 记录 modbusCode 与配置的映射，包内私有
	mu      sync.RWMutex                 // 并发安全锁，包内私有
}

// NewPoolManager 创建基础连接池管理器（无超时清理）
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools:   make(map[string]*ModbusClientPool),
		confMap: make(map[string]*ModbusClientConf),
	}
}

// AddPool 新增连接池（按 modbusCode 关联）
// 入参：modbusCode（唯一标识）、conf（Modbus 配置）、poolSize（池大小）
// 返回：error（明确错误信息）
func (m *PoolManager) AddPool(ctx context.Context, modbusCode string, conf *ModbusClientConf, poolSize int) (*ModbusClientPool, error) {
	if modbusCode == "" {
		return nil, fmt.Errorf("modbusCode 不能为空")
	}
	if conf == nil {
		return nil, fmt.Errorf("ModbusClientConf 不能为空")
	}
	if poolSize <= 0 {
		return nil, fmt.Errorf("poolSize 必须大于 0（当前：%d）", poolSize)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 若已存在相同 modbusCode，先关闭旧池避免泄漏
	if pool, exists := m.pools[modbusCode]; exists {
		logx.Errorf("modbusCode [%s] 已存在（地址：%s）", modbusCode, conf.Address)
		return pool, nil
	}

	// 创建新连接池
	newPool := NewModbusClientPool(conf, poolSize)
	m.pools[modbusCode] = newPool
	m.confMap[modbusCode] = conf
	logx.WithContext(ctx).Infof("modbusCode [%s] 连接池创建成功（地址：%s，池大小：%d）", modbusCode, conf.Address, poolSize)
	return newPool, nil
}

// GetPool 获取指定 modbusCode 的连接池
// 场景：业务逻辑中获取连接池，进而获取客户端执行 Modbus 操作
func (m *PoolManager) GetPool(modbusCode string) (*ModbusClientPool, bool) {
	if modbusCode == "" {
		return nil, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.pools[modbusCode]
	if !exists {
		return nil, false
	}
	return pool, true
}
