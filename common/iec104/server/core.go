package server

import (
	"context"
	"strconv"
	"sync"
	"zero-service/common/iec104"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/clog"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/logx"
)

// Settings keeps the upstream runtime settings shape for the IEC104 server.
type Settings struct {
	Host        string
	Port        int
	Cfg104      *cs104.Config `json:"-"` // 104协议规范配置
	Params      *asdu.Params  `json:"-"` // ASDU相关特定参数
	LogEnable   bool
	LogProvider clog.LogProvider `json:"-"`
}

// ServerConfig is the go-zero config shape. Keep it YAML-loadable only.
type ServerConfig struct {
	Host      string
	Port      int
	LogEnable bool `json:",default=true"`
}

type ServerOption func(*Settings)

type Server struct {
	settings              *Settings
	cs104Server           *cs104.Server
	connections           sync.Map // map[string]asdu.Connect
	connectionHandler     func(asdu.Connect)
	connectionLostHandler func(asdu.Connect)
}

func NewSettings() Settings {
	cfg104 := cs104.DefaultConfig()
	return Settings{
		Host:      "localhost",
		Port:      2404,
		Cfg104:    &cfg104,
		Params:    asdu.ParamsWide,
		LogEnable: true,
	}
}

func WithCS104Config(cfg cs104.Config) ServerOption {
	return func(settings *Settings) {
		settings.Cfg104 = &cfg
	}
}

func WithParams(params *asdu.Params) ServerOption {
	return func(settings *Settings) {
		if params != nil {
			settings.Params = params
		}
	}
}

func WithLogProvider(provider clog.LogProvider) ServerOption {
	return func(settings *Settings) {
		settings.LogProvider = provider
	}
}

func WithLogEnable(enable bool) ServerOption {
	return func(settings *Settings) {
		settings.LogEnable = enable
	}
}

func NewServer(cfg ServerConfig, handler CommandHandler, opts ...ServerOption) *Server {
	settings := NewSettings()
	settings.Host = cfg.Host
	settings.Port = cfg.Port
	settings.LogEnable = cfg.LogEnable
	for _, opt := range opts {
		opt(&settings)
	}
	return New(settings, handler)
}

func New(cfg Settings, handler CommandHandler) *Server {
	if handler == nil {
		panic("iec104 server command handler is nil")
	}
	cfg = normalizeSettings(cfg)
	cs104Server := cs104.NewServer(NewServerHandler(handler))
	cs104Server.SetConfig(*cfg.Cfg104)
	cs104Server.SetParams(cfg.Params)

	cs104Server.LogMode(cfg.LogEnable)
	if cfg.LogProvider != nil {
		cs104Server.SetLogProvider(cfg.LogProvider)
	} else if cfg.LogEnable {
		ctx := logx.ContextWithFields(context.Background(), logx.Field("host", cfg.Host), logx.Field("port", cfg.Port))
		cs104Server.SetLogProvider(iec104.NewLogProvider(ctx))
	}

	s := &Server{
		settings:    &cfg,
		cs104Server: cs104Server,
	}
	cs104Server.SetOnConnectionHandler(s.internalConnectionHandler)
	cs104Server.SetConnectionLostHandler(s.internalConnectionLostHandler)
	return s
}

func normalizeSettings(cfg Settings) Settings {
	defaults := NewSettings()
	if cfg.Host == "" {
		cfg.Host = defaults.Host
	}
	if cfg.Port == 0 {
		cfg.Port = defaults.Port
	}
	if cfg.Cfg104 == nil {
		cfg.Cfg104 = defaults.Cfg104
	}
	if cfg.Params == nil {
		cfg.Params = defaults.Params
	}
	return cfg
}

func (s *Server) Start() {
	addr := s.settings.Host + ":" + strconv.Itoa(s.settings.Port)
	s.cs104Server.ListenAndServer(addr)
}

func (s *Server) Stop() {
	_ = s.cs104Server.Close()
}

// SetOnConnectionHandler set on connect handler
func (s *Server) SetOnConnectionHandler(f func(asdu.Connect)) {
	s.connectionHandler = f
}

// SetConnectionLostHandler set connect lost handler
func (s *Server) SetConnectionLostHandler(f func(asdu.Connect)) {
	s.connectionLostHandler = f
}

// GetConnections get current connections
func (s *Server) GetConnections() []asdu.Connect {
	connects := make([]asdu.Connect, 0)
	s.connections.Range(func(key, value any) bool {
		connects = append(connects, value.(asdu.Connect))
		return true
	})
	return connects
}

func (s *Server) internalConnectionHandler(conn asdu.Connect) {
	addr := conn.UnderlyingConn().RemoteAddr().String()
	s.connections.Store(addr, conn)

	if s.connectionHandler != nil {
		s.connectionHandler(conn)
	}
}

func (s *Server) internalConnectionLostHandler(conn asdu.Connect) {
	addr := conn.UnderlyingConn().RemoteAddr().String()
	s.connections.Delete(addr)

	if s.connectionLostHandler != nil {
		s.connectionLostHandler(conn)
	}
}
