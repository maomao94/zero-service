package iec

import (
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
	"zero-service/app/iecrpc/internal/svc"
)

type IecServer struct {
	cs104Server *cs104.Server
	addr        string
}

func NewIecServer(svcCtx *svc.ServiceContext) *IecServer {
	cs104Server := cs104.NewServer(&ServerHandler{h: NewIecHandler(svcCtx)})
	cfg104 := cs104.DefaultConfig()
	cs104Server.SetConfig(cfg104)
	cs104Server.SetParams(asdu.ParamsWide)
	if svcCtx.Config.IecSetting.Enable {
		cs104Server.LogMode(true)
		cs104Server.SetLogProvider(&LogProvider{})
	}
	addr := svcCtx.Config.IecSetting.Host + ":" + strconv.Itoa(svcCtx.Config.IecSetting.Port)
	return &IecServer{cs104Server: cs104Server, addr: addr}
}

func (q *IecServer) Start() {
	q.cs104Server.ListenAndServer(q.addr)
}

func (q *IecServer) Stop() {
	q.cs104Server.Close()
}

type LogProvider struct {
}

func (l *LogProvider) Critical(format string, v ...interface{}) {
	if v == nil {
		logx.Error(format)
	} else {
		logx.Errorf(format, v)
	}
}

func (l *LogProvider) Error(format string, v ...interface{}) {
	if v == nil {
		logx.Error(format)
	} else {
		logx.Errorf(format, v)
	}
}

func (l *LogProvider) Warn(format string, v ...interface{}) {
	if v == nil {
		logx.Error(format)
	} else {
		logx.Errorf(format, v)
	}
}

func (l *LogProvider) Debug(format string, v ...interface{}) {
	if v == nil {
		logx.Debug(format)
	} else {
		logx.Debugf(format, v)
	}
}
