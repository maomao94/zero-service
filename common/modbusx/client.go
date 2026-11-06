package modbusx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"zero-service/common/tool"

	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
)

// ErrClientInvalidated 表示客户端已被管理器淘汰，调用方应重新获取客户端。
// 写操作不会自动重试。
var ErrClientInvalidated = errors.New("modbus: 客户端已失效，请重新获取客户端")

// ModbusClient 封装 SDK 读写与连接关闭能力，不依赖具体传输方式。
type ModbusClient struct {
	mu          sync.Mutex // 同一客户端串行调用 SDK，取得锁后检查失效状态和 context
	client      *clientWithCloser
	invalidated atomic.Bool
}

// clientWithCloser 组合 SDK 读写接口和连接关闭接口，不依赖具体传输方式。
// grid-x 的 Client 接口不含 Close，由构造阶段提供对应连接的 io.Closer。
type clientWithCloser struct {
	modbus.Client
	io.Closer
}

// callClient 统一加锁并检查失效状态和 context，兼容字节及设备标识结果。
// 等待锁不即时取消；已通过失效检查的调用允许结束，仍遵循 SDK 的超时语义。
func callClient[T any](ctx context.Context, m *ModbusClient, call func() (T, error)) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var zero T
	if m.invalidated.Load() {
		return zero, ErrClientInvalidated
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return call()
}

var _ modbus.Client = (*ModbusClient)(nil)

// ReadCoils 读取线圈状态 (Function Code 0x01)
func (m *ModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.ReadCoils(ctx, address, quantity) })
}

// ReadDiscreteInputs 读取离散输入状态 (Function Code 0x02)
func (m *ModbusClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.ReadDiscreteInputs(ctx, address, quantity) })
}

// WriteSingleCoil 写单个线圈 (Function Code 0x05)
func (m *ModbusClient) WriteSingleCoil(ctx context.Context, address, value uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.WriteSingleCoil(ctx, address, value) })
}

// WriteMultipleCoils 写多个线圈 (Function Code 0x0F)
func (m *ModbusClient) WriteMultipleCoils(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.WriteMultipleCoils(ctx, address, quantity, value) })
}

// ReadInputRegisters 读取输入寄存器 (Function Code 0x04)
func (m *ModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.ReadInputRegisters(ctx, address, quantity) })
}

// ReadHoldingRegisters 读取保持寄存器 (Function Code 0x03)
func (m *ModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.ReadHoldingRegisters(ctx, address, quantity) })
}

// WriteSingleRegister 写单个保持寄存器 (Function Code 0x06)
func (m *ModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.WriteSingleRegister(ctx, address, value) })
}

// WriteMultipleRegisters 写多个保持寄存器 (Function Code 0x10)
func (m *ModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.WriteMultipleRegisters(ctx, address, quantity, value) })
}

// ReadWriteMultipleRegisters 读写多个保持寄存器 (Function Code 0x17)
func (m *ModbusClient) ReadWriteMultipleRegisters(ctx context.Context, readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) {
		return m.client.ReadWriteMultipleRegisters(ctx, readAddress, readQuantity, writeAddress, writeQuantity, value)
	})
}

// MaskWriteRegister 屏蔽写保持寄存器 (Function Code 0x16)
func (m *ModbusClient) MaskWriteRegister(ctx context.Context, address, andMask, orMask uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.MaskWriteRegister(ctx, address, andMask, orMask) })
}

// ReadFIFOQueue 读取 FIFO 队列 (Function Code 0x18)
func (m *ModbusClient) ReadFIFOQueue(ctx context.Context, address uint16) (results []byte, err error) {
	return callClient(ctx, m, func() ([]byte, error) { return m.client.ReadFIFOQueue(ctx, address) })
}

// ReadDeviceIdentification 读取设备标识 (Function Code 0x2B / 0x0E)
func (m *ModbusClient) ReadDeviceIdentification(ctx context.Context, readDeviceIDCode modbus.ReadDeviceIDCode) (results map[byte][]byte, err error) {
	return callClient(ctx, m, func() (map[byte][]byte, error) {
		return m.client.ReadDeviceIdentification(ctx, readDeviceIDCode)
	})
}

// ReadDeviceIdentificationSpecificObject 读取特定 Object ID 的设备标识 (Function Code 0x2B / 0x0E)
func (m *ModbusClient) ReadDeviceIdentificationSpecificObject(ctx context.Context, objectID byte) (results map[byte][]byte, err error) {
	return callClient(ctx, m, func() (map[byte][]byte, error) {
		return m.client.ReadDeviceIdentificationSpecificObject(ctx, objectID)
	})
}

// invalidate 永久作废客户端，不等待在途 I/O，可在管理器缓存锁内调用。
// 管理器随后在缓存锁外调用 Close，等待在途调用结束并关闭连接。
func (m *ModbusClient) invalidate() {
	m.invalidated.Store(true)
}

// Close 关闭当前 TCP；未被管理器作废的客户端后续仍可重新建立连接。
func (m *ModbusClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client.Close()
}

// MustNewModbusClient 创建 Modbus 客户端，如果出错则 panic
func MustNewModbusClient(c *ModbusClientConf) *ModbusClient {
	cli, err := NewModbusClient(c)
	logx.Must(err)
	return cli
}

// NewModbusClient 创建 Modbus 客户端
func NewModbusClient(c *ModbusClientConf) (*ModbusClient, error) {
	conf := c.withDefaults()
	if conf.Slave > MaxSlave {
		return nil, fmt.Errorf("slave 超出 Modbus 从站地址范围 [%d, %d]（当前 %d）", MinSlave, MaxSlave, conf.Slave)
	}
	timeout := time.Millisecond * time.Duration(conf.Timeout)

	// 建连超时必须显式设置：grid-x 在 NewTCPClientHandler 内部用当时的 h.Timeout
	// （默认 10s）固化 DialFunc，构造后再修改 h.Timeout 不会影响建连超时。
	opts := []modbus.TCPClientHandlerOption{
		modbus.WithDialer((&net.Dialer{Timeout: timeout}).DialContext),
	}
	if conf.TLS.Enable {
		cert, err := tls.LoadX509KeyPair(conf.TLS.CertFile, conf.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate failed: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if conf.TLS.CAFile != "" {
			caCert, err := os.ReadFile(conf.TLS.CAFile)
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

	h := modbus.NewTCPClientHandler(conf.Address, opts...)
	h.SetSlave(byte(conf.Slave))
	h.Timeout = timeout
	h.IdleTimeout = time.Millisecond * time.Duration(conf.IdleTimeout)
	h.LinkRecoveryTimeout = time.Millisecond * time.Duration(conf.LinkRecoveryTimeout)
	h.ProtocolRecoveryTimeout = time.Millisecond * time.Duration(conf.ProtocolRecoveryTimeout)
	h.ConnectDelay = time.Millisecond * time.Duration(conf.ConnectDelay)
	sid, err := tool.SimpleUUID()
	if err != nil {
		return nil, fmt.Errorf("generate client id failed: %w", err)
	}
	h.Logger = newModbusLogger(conf, sid)

	return &ModbusClient{
		client: &clientWithCloser{Client: modbus.NewClient(h), Closer: h},
	}, nil
}
