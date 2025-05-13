package cron

import (
	"fmt"
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
		sessionCli := s.svcCtx.ClientManager.GetSessionClients()
		clientsLen := len(clients)
		sessionLen := len(sessionCli)
		loss := 0
		sessionLoss := 0
		for v := range clients {
			if !v.IsConnected() {
				loss++
			}
		}

		for _, v := range sessionCli {
			if !v.GetCli().IsConnected() {
				sessionLoss++
			}
		}
		logx.Statf("(iec-104) client: %d, loss: %d, session: %d, loss: %d", clientsLen, loss, sessionLen, sessionLoss)
	})

	// 定时总召唤
	_, _ = s.c.AddFunc(s.svcCtx.Config.InterrogationCmdCron, func() {
		sessionCli := s.svcCtx.ClientManager.GetSessionClients()
		for _, v := range sessionCli {
			if v.GetCli().IsConnected() {
				// 发送总召唤
				if err := v.GetCli().SendInterrogationCmd(uint16(v.GetConfig().Coa)); err != nil {
					logx.Errorf("send interrogation cmd error %v\n", err)
					continue
				}
				logx.Infof("send interrogation cmd, host: %s, port: %d, coa: %d", v.GetConfig().Host, v.GetConfig().Port, v.GetConfig().Coa)
			}
		}
	})
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
