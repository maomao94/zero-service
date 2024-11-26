package logic

import (
	"context"
	"fmt"
	"github.com/wendy512/go-iecp5/cs104"
	"time"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/iec104/iec104client"
	"zero-service/iec104/iec104server"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendTestCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendTestCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTestCmdLogic {
	return &SendTestCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendTestCmdLogic) SendTestCmd(in *ieccaller.SendTestCmdReq) (*ieccaller.SendTestCmdRes, error) {
	var err error

	option := cs104.NewOption()
	if err = option.AddRemoteServer("127.0.0.1:2404"); err != nil {
		panic(err)
	}

	handler := &iec104client.ClientHandler{Call: &iec.ClientCall{}}

	client := cs104.NewClient(handler, option)

	client.LogMode(true)
	client.SetLogProvider(iec104server.NewLogProvider())

	client.SetOnConnectHandler(func(c *cs104.Client) {
		c.SendStartDt() // 发送startDt激活指令
	})
	err = client.Start()
	if err != nil {
		panic(fmt.Errorf("Failed to connect. error:%v\n", err))
	}

	for {
		time.Sleep(time.Second * 100)
	}
	return &ieccaller.SendTestCmdRes{}, nil
}
