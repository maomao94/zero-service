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

	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
)

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

func (m ModbusClient) ReadCoils(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadCoils(ctx, address, quantity)
}

func (m ModbusClient) ReadDiscreteInputs(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadDiscreteInputs(ctx, address, quantity)
}

func (m ModbusClient) WriteSingleCoil(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleCoil(ctx, address, value)
}

func (m ModbusClient) WriteMultipleCoils(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleCoils(ctx, address, quantity, value)
}

func (m ModbusClient) ReadInputRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadInputRegisters(ctx, address, quantity)
}

func (m ModbusClient) ReadHoldingRegisters(ctx context.Context, address, quantity uint16) (results []byte, err error) {
	return m.client.ReadHoldingRegisters(ctx, address, quantity)
}

func (m ModbusClient) WriteSingleRegister(ctx context.Context, address, value uint16) (results []byte, err error) {
	return m.client.WriteSingleRegister(ctx, address, value)
}

func (m ModbusClient) WriteMultipleRegisters(ctx context.Context, address, quantity uint16, value []byte) (results []byte, err error) {
	return m.client.WriteMultipleRegisters(ctx, address, quantity, value)
}

func (m ModbusClient) ReadWriteMultipleRegisters(ctx context.Context, readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	return m.client.ReadWriteMultipleRegisters(ctx, readAddress, readQuantity, writeAddress, writeQuantity, value)
}

func (m ModbusClient) MaskWriteRegister(ctx context.Context, address, andMask, orMask uint16) (results []byte, err error) {
	return m.client.MaskWriteRegister(ctx, address, andMask, orMask)
}

func (m ModbusClient) ReadFIFOQueue(ctx context.Context, address uint16) (results []byte, err error) {
	return m.client.ReadFIFOQueue(ctx, address)
}

func (m ModbusClient) ReadDeviceIdentification(ctx context.Context, readDeviceIDCode modbus.ReadDeviceIDCode) (results map[byte][]byte, err error) {
	return m.client.ReadDeviceIdentification(ctx, readDeviceIDCode)
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
	pool *syncx.Pool
	lock sync.Mutex
	conf *ModbusClientConf
}

// NewModbusClientPool 初始化一个 Modbus 客户端连接池
func NewModbusClientPool(conf *ModbusClientConf, size int) *ModbusClientPool {
	p := &ModbusClientPool{
		conf: conf,
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

// Get 获取一个 Modbus TCP 客户端连接
func (p *ModbusClientPool) Get() *ModbusClient {
	return p.pool.Get().(*ModbusClient)
}

// Put 归还一个客户端连接
func (p *ModbusClientPool) Put(cli *ModbusClient) {
	p.pool.Put(cli)
}

type ModbusLogger struct {
	conf *ModbusClientConf
}

func (l *ModbusLogger) Printf(format string, v ...any) {
	ctx := logx.ContextWithFields(context.Background(), logx.Field("address", l.conf.Address))
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
