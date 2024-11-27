package logic

import (
	"context"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"github.com/zeromicro/go-zero/core/executors"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/iec"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/iec104"
	"zero-service/iec104/iec104client"
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
		return nil, err
	}
	option.SetParams(asdu.ParamsWide)
	handler := &iec104client.ClientHandler{Call: &iec.ClientCall{}}
	client := cs104.NewClient(handler, option)
	client.LogMode(true)
	client.SetLogProvider(iec104.NewLogProvider())
	client.SetOnConnectHandler(func(c *cs104.Client) {
		// 连接成功以后做的操作
		l.Logger.Infof("connected %s iec104 server", "127.0.0.1:2404")
		c.SendStartDt()
	})
	client.SetServerActiveHandler(func(c *cs104.Client) {
		// 连接成功以后做的操作
		l.Logger.Infof("server active %s iec104 server", "IP_ADDRESS:2404")
		//coa := asdu.CauseOfTransmission{
		//	IsTest:     false,
		//	IsNegative: false,
		//	Cause:      asdu.Activation,
		//}
		//
		//err = client.InterrogationCmd(coa, asdu.CommonAddr(1), asdu.QOIStation)
		//if err != nil {
		//	l.Logger.Errorf("interrogation cmd error: %s", err.Error())
		//}
	})
	err = client.Start()
	if err != nil {
		return nil, err
	}
	executors.NewDelayExecutor(func() {
		client.SendStopDt()
		client.Close()
	}, 5*time.Second).Trigger()
	return &ieccaller.SendTestCmdRes{}, nil
}
