package cron

import (
	"fmt"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/ieccaller/internal/svc"
)

type CronService struct {
	c      *cron.Cron
	svcCtx *svc.ServiceContext
}

func NewCronService(svcCtx *svc.ServiceContext) *CronService {
	return &CronService{
		c:      cron.New(cron.WithSeconds()),
		svcCtx: svcCtx,
	}
}

func (s *CronService) Start() {
	s.c = cron.New(cron.WithSeconds()) // 支持秒级调度
	// stat
	_, _ = s.c.AddFunc("*/60 * * * * *", func() {
		clients := s.svcCtx.ClientManager.GetClients()
		clientsLen := len(clients)
		loss := 0
		for v := range clients {
			if !v.IsConnected() {
				loss++
			}
		}
		logx.Statf("(iec-104) client: %d, loss: %d", clientsLen, loss)
	})

	// 定时总召唤
	_, _ = s.c.AddFunc(s.svcCtx.Config.InterrogationCmdCron, func() {
		cliList := s.svcCtx.ClientManager.GetClients()
		for cli, _ := range cliList {
			if cli.IsConnected() {
				// 发送总召唤
				icCoaList := cli.GetIcCoaList()
				for _, v := range icCoaList {
					convertor.ToInt(v)
					if err := cli.SendInterrogationCmd(v); err != nil {
						logx.Errorf("send interrogation cmd error %v\n", err)
						continue
					}
					logx.Infof("send interrogation cmd, serverUrl: %s, coa: %d", cli.GetServerUrl(), v)
				}
			}
		}
	})
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
