package iec104server

import (
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
)

type IecServer struct {
	cs104Server *cs104.Server
	addr        string
}

func NewIecServer(host string, port int, logMode bool, handler *ServerHandler) *IecServer {
	cs104Server := cs104.NewServer(handler)
	cfg104 := cs104.DefaultConfig()
	cs104Server.SetConfig(cfg104)
	cs104Server.SetParams(asdu.ParamsWide)
	if logMode {
		cs104Server.LogMode(true)
		cs104Server.SetLogProvider(NewLogProvider())
	}
	addr := host + ":" + strconv.Itoa(port)
	return &IecServer{cs104Server: cs104Server, addr: addr}
}

func (q *IecServer) Start() {
	q.cs104Server.ListenAndServer(q.addr)
}

func (q *IecServer) Stop() {
	q.cs104Server.Close()
}

type LogProvider struct {
	logx.Logger
}

func NewLogProvider() *LogProvider {
	return &LogProvider{
		Logger: logx.WithContext(nil),
	}
}

func (l *LogProvider) Critical(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Error(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Warn(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Debug(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Debug(format)
	} else {
		l.Logger.Debugf(format, v)
	}
}
