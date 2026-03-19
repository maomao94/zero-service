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
	"zero-service/common/tool"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
)

// ModbusClient 封装了 modbus.Client，提供额外的功能
type ModbusClient struct {
	sid     string
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

var _ modbus.Client = (*ModbusClient)(nil)

// ReadCoils 读取线圈状态 (Function Code 0x01)
func (m *ModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadCoils(ctx, address, quantity)
}

// ReadDiscreteInputs 读取离散输入状态 (Function Code 0x02)
func (m *ModbusClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadDiscreteInputs(ctx, address, quantity)
}

// WriteSingleCoil 写单个线圈 (Function Code 0x05)
func (m *ModbusClient) WriteSingleCoil(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleCoil(ctx, address, value)
}

// WriteMultipleCoils 写多个线圈 (Function Code 0x0F)
func (m *ModbusClient) WriteMultipleCoils(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleCoils(ctx, address, quantity, value)
}

// ReadInputRegisters 读取输入寄存器 (Function Code 0x04)
func (m *ModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadInputRegisters(ctx, address, quantity)
}

// ReadHoldingRegisters 读取保持寄存器 (Function Code 0x03)
func (m *ModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadHoldingRegisters(ctx, address, quantity)
}

// WriteSingleRegister 写单个保持寄存器 (Function Code 0x06)
func (m *ModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleRegister(ctx, address, value)
}

// WriteMultipleRegisters 写多个保持寄存器 (Function Code 0x10)
func (m *ModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleRegisters(ctx, address, quantity, value)
}

// ReadWriteMultipleRegisters 读写多个保持寄存器 (Function Code 0x17)
func (m *ModbusClient) ReadWriteMultipleRegisters(ctx context.Context, readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return m.client.ReadWriteMultipleRegisters(ctx, readAddress, readQuantity, writeAddress, writeQuantity, value)
}

// MaskWriteRegister 屏蔽写保持寄存器 (Function Code 0x16)
func (m *ModbusClient) MaskWriteRegister(ctx context.Context, address, andMask, orMask uint16) (results []byte, err error) {
	return m.client.MaskWriteRegister(ctx, address, andMask, orMask)
}

// ReadFIFOQueue 读取 FIFO 队列 (Function Code 0x18)
func (m *ModbusClient) ReadFIFOQueue(ctx context.Context, address uint16) (results []byte, err error) {
	return m.client.ReadFIFOQueue(ctx, address)
}

// ReadDeviceIdentification 读取设备标识 (Function Code 0x2B / 0x0E)
func (m *ModbusClient) ReadDeviceIdentification(ctx context.Context, readDeviceIDCode modbus.ReadDeviceIDCode) (results map[byte][]byte, err error) {
	return m.client.ReadDeviceIdentification(ctx, readDeviceIDCode)
}

// ReadDeviceIdentificationSpecificObject 读取特定 Object ID 的设备标识 (Function Code 0x2B / 0x0E)
func (m *ModbusClient) ReadDeviceIdentificationSpecificObject(ctx context.Context, objectID byte) (results map[byte][]byte, err error) {
	return m.client.ReadDeviceIdentificationSpecificObject(ctx, objectID)
}

// Close 关闭客户端连接
func (m *ModbusClient) Close() error {
	return m.handler.Close()
}

// MustNewModbusClient 创建 Modbus 客户端，如果出错则 panic
func MustNewModbusClient(c *ModbusClientConf) *ModbusClient {
	cli, err := NewModbusClient(c)
	logx.Must(err)
	return cli
}

// NewModbusClient 创建 Modbus 客户端
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
	h.SetSlave(byte(c.Slave))
	h.Timeout = time.Millisecond * time.Duration(c.Timeout)
	h.IdleTimeout = time.Millisecond * time.Duration(c.IdleTimeout)
	h.LinkRecoveryTimeout = time.Millisecond * time.Duration(c.LinkRecoveryTimeout)
	h.ProtocolRecoveryTimeout = time.Millisecond * time.Duration(c.ProtocolRecoveryTimeout)
	h.ConnectDelay = time.Millisecond * time.Duration(c.ConnectDelay)
	sid, _ := tool.SimpleUUID()
	h.Logger = &ModbusLogger{conf: c, sid: sid}
	return &ModbusClient{
		sid:     sid,
		client:  modbus.NewClient(h),
		handler: h,
	}, nil
}

// ModbusClientPool 管理 Modbus TCP 客户端连接复用
type ModbusClientPool struct {
	pool     *syncx.Pool
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
			logx.Infof("create modbus client: %s", conf.Address)
			cli, err := NewModbusClient(conf)
			logx.Must(err)
			return cli
		},
		func(x any) {
			logx.Infof("close modbus client: %s", conf.Address)
			if h, ok := x.(*ModbusClient); ok {
				h.handler.Close()
			}
		},
		syncx.WithMaxAge(time.Minute*10), // 资源 10 分钟未使用自动销毁
	)

	return p
}

// Get 获取一个 Modbus 客户端
func (p *ModbusClientPool) Get() *ModbusClient {
	p.mu.Lock()
	p.lastUsed = time.Now() // 更新最后使用时间
	p.mu.Unlock()
	return p.pool.Get().(*ModbusClient)
}

// Put 归还一个 Modbus 客户端
func (p *ModbusClientPool) Put(cli *ModbusClient) {
	p.pool.Put(cli)
}

// ModbusLogger 自定义 Modbus 日志记录器
type ModbusLogger struct {
	conf *ModbusClientConf
	sid  string
}

// Printf 打印日志
func (l *ModbusLogger) Printf(format string, v ...any) {
	ctx := logx.ContextWithFields(context.Background(), logx.Field("address", l.conf.Address))
	ctx = logx.ContextWithFields(ctx, logx.Field("addressMd5", cryptor.Md5String(l.conf.Address)))
	ctx = logx.ContextWithFields(ctx, logx.Field("session", l.sid))
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
