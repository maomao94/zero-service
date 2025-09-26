package modbusx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
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
	SlaveID byte `json:"slaveID"`

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

type ModbusClient struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

var _ modbus.Client = (*ModbusClient)(nil)

func (m *ModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadCoils(ctx, address, quantity)
}

func (m *ModbusClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadDiscreteInputs(ctx, address, quantity)
}

func (m *ModbusClient) WriteSingleCoil(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleCoil(ctx, address, value)
}

func (m *ModbusClient) WriteMultipleCoils(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleCoils(ctx, address, quantity, value)
}

func (m *ModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadInputRegisters(ctx, address, quantity)
}

func (m *ModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadHoldingRegisters(ctx, address, quantity)
}

func (m *ModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleRegister(ctx, address, value)
}

func (m *ModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleRegisters(ctx, address, quantity, value)
}

func (m *ModbusClient) ReadWriteMultipleRegisters(ctx context.Context, readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return m.client.ReadWriteMultipleRegisters(ctx, readAddress, readQuantity, writeAddress, writeQuantity, value)
}

func (m *ModbusClient) MaskWriteRegister(ctx context.Context, address, andMask, orMask uint16) (results []byte, err error) {
	return m.client.MaskWriteRegister(ctx, address, andMask, orMask)
}

func (m *ModbusClient) ReadFIFOQueue(ctx context.Context, address uint16) (results []byte, err error) {
	return m.client.ReadFIFOQueue(ctx, address)
}

func (m *ModbusClient) ReadDeviceIdentification(ctx context.Context, readDeviceIDCode modbus.ReadDeviceIDCode) (results map[byte][]byte, err error) {
	return m.client.ReadDeviceIdentification(ctx, readDeviceIDCode)
}

func (m *ModbusClient) Close() error {
	return m.handler.Close()
}

func MustNewModbusClient(c *ModbusClientConf) *ModbusClient {
	cli, err := NewModbusClient(c)
	logx.Must(err)
	return cli
}

func NewModbusClient(c *ModbusClientConf) (*ModbusClient, error) {
	var opts []modbus.TCPClientHandlerOption
	if c.TLS.Enable {
		cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate failed: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if c.TLS.CAFile != "" {
			caCert, err := os.ReadFile(c.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("read CA file failed: %w", err)
			}
			caCertPool.AppendCertsFromPEM(caCert)
		}

		opts = append(opts, modbus.WithTLSConfig(&tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}))
	}
	h := modbus.NewTCPClientHandler(c.Address, opts...)
	h.SetSlave(c.SlaveID)
	h.Timeout = time.Millisecond * time.Duration(c.Timeout)
	h.IdleTimeout = time.Millisecond * time.Duration(c.IdleTimeout)
	h.LinkRecoveryTimeout = time.Millisecond * time.Duration(c.LinkRecoveryTimeout)
	h.ProtocolRecoveryTimeout = time.Millisecond * time.Duration(c.ProtocolRecoveryTimeout)
	h.ConnectDelay = time.Millisecond * time.Duration(c.ConnectDelay)
	h.Logger = &ModbusLogger{conf: c}
	return &ModbusClient{
		client:  modbus.NewClient(h),
		handler: h,
	}, nil
}

// ModbusClientPool 管理 Modbus TCP 客户端连接复用
type ModbusClientPool struct {
	pool     *syncx.Pool
	lock     sync.Mutex
	conf     *ModbusClientConf
	mu       sync.Mutex
	lastUsed time.Time
}

// NewModbusClientPool 初始化一个 Modbus 客户端连接池
func NewModbusClientPool(conf *ModbusClientConf, size int) *ModbusClientPool {
	p := &ModbusClientPool{
		conf:     conf,
		lastUsed: time.Now(),
	}

	p.pool = syncx.NewPool(
		size,
		func() any {
			logx.Debugf("create modbus client: %s", conf.Address)
			cli, err := NewModbusClient(conf)
			logx.Must(err)
			return cli
		},
		func(x any) {
			logx.Debug("close modbus client: %s", conf.Address)
			if h, ok := x.(*ModbusClient); ok {
				h.handler.Close()
			}
		},
		syncx.WithMaxAge(time.Minute*10), // 资源 10 分钟未使用自动销毁
	)

	return p
}

func (p *ModbusClientPool) Get() *ModbusClient {
	p.mu.Lock()
	p.lastUsed = time.Now() // 更新最后使用时间
	p.mu.Unlock()
	return p.pool.Get().(*ModbusClient)
}

func (p *ModbusClientPool) Put(cli *ModbusClient) {
	p.pool.Put(cli)
}

type ModbusLogger struct {
	conf *ModbusClientConf
}

func (l *ModbusLogger) Printf(format string, v ...any) {
	ctx := logx.ContextWithFields(context.Background(), logx.Field("address", l.conf.Address))
	ctx = logx.ContextWithFields(ctx, logx.Field("session", cryptor.Md5String(l.conf.Address)))
	for _, val := range v {
		if err, ok := val.(error); ok && err != nil {
			logx.Error(err)
			return
		}
	}
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimRight(msg, "\n")
	if strings.Contains(strings.ToLower(msg), "error") || strings.Contains(strings.ToLower(msg), "err") {
		logx.WithContext(ctx).Error(msg)
	} else {
		logx.WithContext(ctx).Info(msg)
	}
}

func BytesToBools(data []byte, quantity int) []bool {
	bools := make([]bool, quantity)
	for i := 0; i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bools[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}
	return bools
}

// PoolManager 连接池管理器：管理多个 Modbus 连接池（按 sessionID 区分）
type PoolManager struct {
	pools   map[string]*ModbusClientPool // key: sessionID，包内私有
	confMap map[string]*ModbusClientConf // 记录 session 与配置的映射，包内私有
	mu      sync.RWMutex                 // 并发安全锁，包内私有
}

// NewPoolManager 创建基础连接池管理器（无超时清理）
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools:   make(map[string]*ModbusClientPool),
		confMap: make(map[string]*ModbusClientConf),
	}
}

// AddPool 新增连接池（按 sessionID 关联）
// 入参：sessionID（唯一标识）、conf（Modbus 配置）、poolSize（池大小）
// 返回：error（明确错误信息）
func (m *PoolManager) AddPool(sessionID string, conf *ModbusClientConf, poolSize int) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID 不能为空")
	}
	if conf == nil {
		return fmt.Errorf("ModbusClientConf 不能为空")
	}
	if poolSize <= 0 {
		return fmt.Errorf("poolSize 必须大于 0（当前：%d）", poolSize)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 若已存在相同 sessionID，先关闭旧池避免泄漏
	if _, exists := m.pools[sessionID]; exists {
		logx.Errorf("sessionID [%s] 已存在（地址：%s）", sessionID, conf.Address)
		return nil
	}

	// 创建新连接池
	newPool := NewModbusClientPool(conf, poolSize)
	m.pools[sessionID] = newPool
	m.confMap[sessionID] = conf
	logx.Infof("sessionID [%s] 连接池创建成功（地址：%s，池大小：%d）", sessionID, conf.Address, poolSize)
	return nil
}

// GetPool 获取指定 sessionID 的连接池
// 场景：业务逻辑中获取连接池，进而获取客户端执行 Modbus 操作
func (m *PoolManager) GetPool(sessionID string) (*ModbusClientPool, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID 不能为空")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.pools[sessionID]
	if !exists {
		return nil, fmt.Errorf("sessionID [%s] 不存在，请先调用 AddPool 创建", sessionID)
	}
	return pool, nil
}
