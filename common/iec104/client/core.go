package client

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"
	"time"
	"zero-service/common/iec104"

	"github.com/spf13/cast"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
)

type ClientConfig struct {
	Host              string
	Port              int
	AutoConnect       bool           `json:",default=true"`
	ReconnectInterval time.Duration  `json:",default=1m"`
	LogEnable         bool           `json:",default=true"`
	MetaData          map[string]any `json:",optional"`
}

// Validate 验证配置
func (cfg ClientConfig) Validate() error {
	if cfg.Host == "" {
		return fmt.Errorf("host is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	return nil
}

// ConnectionEvent 连接事件类型
type ConnectionEvent int

const (
	EventConnected ConnectionEvent = iota
	EventDisconnected
	EventServerActive
)

// Client 104客户端
type Client struct {
	client104 *cs104.Client
	cfg       ClientConfig
	asduCall  ASDUCall
	running   atomic.Bool
	handler   *ClientHandler
}

// Option 客户端选项
type Option func(*Client)

// WithASDUHandler 设置ASDU处理器
func WithASDUHandler(handler ASDUCall) Option {
	return func(c *Client) {
		c.asduCall = handler
	}
}

// WithMetaData 设置元数据
func WithMetaData(metaData map[string]any) Option {
	return func(c *Client) {
		c.cfg.MetaData = metaData
	}
}

// WithAutoConnect 设置自动重连
func WithAutoConnect(autoConnect bool) Option {
	return func(c *Client) {
		c.cfg.AutoConnect = autoConnect
	}
}

func MustNewClient(cfg ClientConfig, opts ...Option) *Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	return cli
}

// NewClient 创建新的104客户端
func NewClient(cfg ClientConfig, opts ...Option) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		cfg:      cfg,
		asduCall: &emptyASDUCall{},
	}

	// 应用选项
	for _, opt := range opts {
		opt(client)
	}

	// 初始化客户端处理器
	client.handler = &ClientHandler{
		call:    client.asduCall,
		metrics: stat.NewMetrics(fmt.Sprintf("tcp-%s_%d", cfg.Host, cfg.Port)),
	}

	// 初始化104客户端
	client104, err := client.initClient104()
	if err != nil {
		return nil, err
	}
	client.client104 = client104

	return client, nil
}

// initClient104 初始化104客户端
func (c *Client) initClient104() (*cs104.Client, error) {
	opts := newClientOption(c.cfg)
	client104 := cs104.NewClient(c.handler, opts)

	// 设置日志配置
	client104.LogMode(c.cfg.LogEnable)
	ctx := logx.ContextWithFields(context.Background(), logx.Field("host", c.cfg.Host), logx.Field("port", c.cfg.Port))
	client104.SetLogProvider(iec104.NewLogProvider(ctx))

	// 设置连接事件处理器
	client104.SetOnConnectHandler(func(_ *cs104.Client) {
		c.running.Store(true)
		// 发送START_DT帧，建立数据传输
		client104.SendStartDt()
		c.onConnectionEvent(EventConnected)
	})

	client104.SetConnectionLostHandler(func(_ *cs104.Client) {
		c.running.Store(false)
		c.onConnectionEvent(EventDisconnected)
	})

	client104.SetServerActiveHandler(func(_ *cs104.Client) {
		c.onConnectionEvent(EventServerActive)
	})

	return client104, nil
}

func (c *Client) Start() {
	err := c.Connect()
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Client) Stop() {
	c.running.Store(false)
	c.Close()
	logx.Infof("IEC104 client %s:%d is closed", c.cfg.Host, c.cfg.Port)
}

// Connect 连接到104服务器
func (c *Client) Connect() error {
	return c.client104.Start()
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	c.client104.SendStopDt()
	return c.client104.Close()
}

// IsConnected 检查客户端是否连接
func (c *Client) IsConnected() bool {
	return c.client104.IsConnected()
}

// IsRunning 检查客户端是否运行
func (c *Client) IsRunning() bool {
	return c.running.Load()
}

// SendInterrogationCmd 发送总召唤命令
func (c *Client) SendInterrogationCmd(addr uint16) error {
	return c.doSend(&command{typeId: asdu.C_IC_NA_1, ca: asdu.CommonAddr(addr)})
}

// SendCounterInterrogationCmd 发送计数器召唤命令
func (c *Client) SendCounterInterrogationCmd(addr uint16) error {
	return c.doSend(&command{typeId: asdu.C_CI_NA_1, ca: asdu.CommonAddr(addr)})
}

// SendClockSynchronizationCmd 发送时钟同步命令
func (c *Client) SendClockSynchronizationCmd(addr uint16, t time.Time) error {
	return c.doSend(&command{typeId: asdu.C_CS_NA_1, ca: asdu.CommonAddr(addr), t: t})
}

// SendReadCmd 发送读命令
func (c *Client) SendReadCmd(addr uint16, ioa uint) error {
	return c.doSend(&command{typeId: asdu.C_RD_NA_1, ioa: asdu.InfoObjAddr(ioa), ca: asdu.CommonAddr(addr)})
}

// SendResetProcessCmd 发送复位进程命令
func (c *Client) SendResetProcessCmd(addr uint16) error {
	return c.doSend(&command{typeId: asdu.C_RP_NA_1, ca: asdu.CommonAddr(addr)})
}

// SendTestCmd 发送测试命令
func (c *Client) SendTestCmd(addr uint16) error {
	return c.doSend(&command{typeId: asdu.C_TS_TA_1, ca: asdu.CommonAddr(addr)})
}

// SendCmd 发送控制命令
func (c *Client) SendCmd(addr uint16, typeId asdu.TypeID, ioa asdu.InfoObjAddr, value any) error {
	cmd := &command{
		typeId: typeId,
		ioa:    ioa,
		ca:     asdu.CommonAddr(addr),
		value:  value,
		qoc: asdu.QualifierOfCommand{
			Qual:     asdu.QOCNoAdditionalDefinition,
			InSelect: false,
		},
		qos: asdu.QualifierOfSetpointCmd{
			Qual:     0,
			InSelect: false,
		},
		t: time.Now(),
	}

	return c.doSend(cmd)
}

// GetServerURL 获取服务器URL
func (c *Client) GetServerURL() string {
	return formatServerUrl(c.cfg)
}

// GetHost 获取服务器主机
func (c *Client) GetHost() string {
	return c.cfg.Host
}

// GetPort 获取服务器端口
func (c *Client) GetPort() int {
	return c.cfg.Port
}

// GetMetaData 获取元数据
func (c *Client) GetMetaData() map[string]any {
	return c.cfg.MetaData
}

// onConnectionEvent 处理连接事件，内部直接打印日志
func (c *Client) onConnectionEvent(event ConnectionEvent) {
	var eventName string
	switch event {
	case EventConnected:
		eventName = "Connected"
	case EventDisconnected:
		eventName = "Disconnected"
	case EventServerActive:
		eventName = "ServerActive"
	default:
		eventName = "Unknown"
	}
	logx.Infof("IEC104 client %s:%d %s", c.cfg.Host, c.cfg.Port, eventName)
}

// newClientOption 创建客户端选项
func newClientOption(cfg ClientConfig) *cs104.ClientOption {
	opts := cs104.NewOption()
	cfg104 := cs104.DefaultConfig()
	opts.SetConfig(cfg104)

	opts.SetParams(asdu.ParamsWide)
	opts.SetAutoReconnect(cfg.AutoConnect)
	opts.SetReconnectInterval(cfg.ReconnectInterval)

	server := formatServerUrl(cfg)
	_ = opts.AddRemoteServer(server)

	return opts
}

// formatServerUrl 格式化服务器URL
func formatServerUrl(cfg ClientConfig) string {
	var server string
	// 暂时只支持tcp协议，因为没有TLS字段
	server = "tcp://" + cfg.Host + ":" + strconv.Itoa(cfg.Port)
	return server
}

// command 命令结构体
type command struct {
	typeId asdu.TypeID
	ca     asdu.CommonAddr
	ioa    asdu.InfoObjAddr
	t      time.Time
	qoc    asdu.QualifierOfCommand
	qos    asdu.QualifierOfSetpointCmd
	value  any
}

// doSend 执行发送命令
func (c *Client) doSend(cmd *command) error {
	if !c.IsConnected() {
		return NotConnected
	}

	coa := activationCoa()
	var err error

	switch cmd.typeId {
	case asdu.C_IC_NA_1:
		err = c.client104.InterrogationCmd(coa, cmd.ca, asdu.QOIStation)
	case asdu.C_CI_NA_1:
		qcc := asdu.QualifierCountCall{Request: asdu.QCCTotal, Freeze: asdu.QCCFrzRead}
		err = c.client104.CounterInterrogationCmd(coa, cmd.ca, qcc)
	case asdu.C_CS_NA_1:
		err = c.client104.ClockSynchronizationCmd(coa, cmd.ca, cmd.t)
	case asdu.C_RD_NA_1:
		err = c.client104.ReadCmd(coa, cmd.ca, cmd.ioa)
	case asdu.C_RP_NA_1:
		err = c.client104.ResetProcessCmd(coa, cmd.ca, asdu.QPRGeneralRest)
	case asdu.C_TS_TA_1:
		err = c.client104.TestCommand(coa, cmd.ca)
	case asdu.C_SC_NA_1, asdu.C_SC_TA_1:
		var value bool
		value, err = cast.ToBoolE(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.SingleCommandInfo{
			Ioa:   cmd.ioa,
			Value: value,
			Qoc:   cmd.qoc,
		}
		if cmd.typeId == asdu.C_SC_TA_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.SingleCmd(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_DC_NA_1, asdu.C_DC_TA_1:
		var value uint8
		value, err = cast.ToUint8E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.DoubleCommandInfo{
			Ioa:   cmd.ioa,
			Value: asdu.DoubleCommand(value),
			Qoc:   cmd.qoc,
		}
		if cmd.typeId == asdu.C_DC_TA_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.DoubleCmd(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_RC_NA_1, asdu.C_RC_TA_1:
		var value uint8
		value, err = cast.ToUint8E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.StepCommandInfo{
			Ioa:   cmd.ioa,
			Value: asdu.StepCommand(value),
			Qoc:   cmd.qoc,
		}
		if cmd.typeId == asdu.C_RC_TA_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.StepCmd(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_SE_NA_1, asdu.C_SE_TA_1:
		var value int16
		value, err = cast.ToInt16E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.SetpointCommandNormalInfo{
			Ioa:   cmd.ioa,
			Value: asdu.Normalize(value),
			Qos:   cmd.qos,
		}
		if cmd.typeId == asdu.C_SE_TA_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.SetpointCmdNormal(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_SE_NB_1, asdu.C_SE_TB_1:
		var value int16
		value, err = cast.ToInt16E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.SetpointCommandScaledInfo{
			Ioa:   cmd.ioa,
			Value: value,
			Qos:   cmd.qos,
		}
		if cmd.typeId == asdu.C_SE_TB_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.SetpointCmdScaled(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_SE_NC_1, asdu.C_SE_TC_1:
		var value float32
		value, err = cast.ToFloat32E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.SetpointCommandFloatInfo{
			Ioa:   cmd.ioa,
			Value: value,
			Qos:   cmd.qos,
		}
		if cmd.typeId == asdu.C_SE_TC_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.SetpointCmdFloat(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	case asdu.C_BO_NA_1, asdu.C_BO_TA_1:
		var value uint32
		value, err = cast.ToUint32E(cmd.value)
		if err != nil {
			return err
		}
		asduCmd := asdu.BitsString32CommandInfo{
			Ioa:   cmd.ioa,
			Value: value,
		}
		if cmd.typeId == asdu.C_BO_TA_1 {
			asduCmd.Time = cmd.t
		}
		err = asdu.BitsString32Cmd(c.client104, cmd.typeId, coa, cmd.ca, asduCmd)
	default:
		err = fmt.Errorf("unknown type id %d", cmd.typeId)
	}

	return err
}

// activationCoa 获取激活COA
func activationCoa() asdu.CauseOfTransmission {
	return asdu.CauseOfTransmission{
		IsTest:     false,
		IsNegative: false,
		Cause:      asdu.Activation,
	}
}
