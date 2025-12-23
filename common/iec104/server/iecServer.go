package server

import (
	"context"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
	"zero-service/common/iec104"
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
		ctx := logx.ContextWithFields(context.Background())
		cs104Server.SetLogProvider(iec104.NewLogProvider(ctx))
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
