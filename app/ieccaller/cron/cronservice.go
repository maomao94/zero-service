package cron

import (
	"fmt"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
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
	// 定时总召唤
	if len(s.svcCtx.Config.InterrogationCmdCron) > 0 {
		_, _ = s.c.AddFunc(s.svcCtx.Config.InterrogationCmdCron, func() {
			for _, serverCfg := range s.svcCtx.Config.IecServerConfig {
				// 获取客户端
				cli, err := s.svcCtx.ClientManager.GetClient(serverCfg.Host, serverCfg.Port)
				if err != nil {
					continue
				}
				if !cli.IsConnected() {
					continue
				}
				// 发送总召唤
				for _, v := range serverCfg.IcCoaList {
					if err := cli.SendInterrogationCmd(v); err != nil {
						logx.Errorf("send interrogation cmd error %v\n", err)
						continue
					}
					logx.Infof("send interrogation cmd, server: %s:%d, coa: %d", serverCfg.Host, serverCfg.Port, v)
				}
			}
		})
	}

	// 定时累计量召唤
	if len(s.svcCtx.Config.CounterInterrogationCmd) > 0 {
		_, _ = s.c.AddFunc(s.svcCtx.Config.CounterInterrogationCmd, func() {
			for _, serverCfg := range s.svcCtx.Config.IecServerConfig {
				// 获取客户端
				cli, err := s.svcCtx.ClientManager.GetClient(serverCfg.Host, serverCfg.Port)
				if err != nil {
					continue
				}
				if !cli.IsConnected() {
					continue
				}
				// 累计量召唤
				for _, v := range serverCfg.CcCoaList {
					if err := cli.SendCounterInterrogationCmd(v); err != nil {
						logx.Errorf("send counter interrogation cmd error %v\n", err)
						continue
					}
					logx.Infof("send counter interrogation cmd, server: %s:%d, coa: %d", serverCfg.Host, serverCfg.Port, v)
				}
			}
		})
	}
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
