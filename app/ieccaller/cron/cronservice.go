package cron

import (
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
	if len(s.svcCtx.Config.InterrogationCmdCron) > 0 {
		if _, err := s.c.AddFunc(s.svcCtx.Config.InterrogationCmdCron, func() {
			for _, serverCfg := range s.svcCtx.Config.IecServerConfig {
				cli, err := s.svcCtx.ClientManager.GetClient(serverCfg.Host, serverCfg.Port)
				if err != nil {
					logx.Errorf("get iec104 client failed, host: %s, port: %d, err: %v", serverCfg.Host, serverCfg.Port, err)
					continue
				}
				if !cli.IsConnected() {
					continue
				}
				for _, v := range serverCfg.IcCoaList {
					if err := cli.SendInterrogationCmd(v); err != nil {
						logx.Errorf("send interrogation cmd failed, server: %s:%d, coa: %d, err: %v", serverCfg.Host, serverCfg.Port, v, err)
						continue
					}
					logx.Infof("send interrogation cmd, server: %s:%d, coa: %d", serverCfg.Host, serverCfg.Port, v)
				}
			}
		}); err != nil {
			logx.Errorf("add interrogation cron failed, cron: %s, err: %v", s.svcCtx.Config.InterrogationCmdCron, err)
		}
	}

	if len(s.svcCtx.Config.CounterInterrogationCmd) > 0 {
		if _, err := s.c.AddFunc(s.svcCtx.Config.CounterInterrogationCmd, func() {
			for _, serverCfg := range s.svcCtx.Config.IecServerConfig {
				cli, err := s.svcCtx.ClientManager.GetClient(serverCfg.Host, serverCfg.Port)
				if err != nil {
					logx.Errorf("get iec104 client failed, host: %s, port: %d, err: %v", serverCfg.Host, serverCfg.Port, err)
					continue
				}
				if !cli.IsConnected() {
					continue
				}
				for _, v := range serverCfg.CcCoaList {
					if err := cli.SendCounterInterrogationCmd(v); err != nil {
						logx.Errorf("send counter interrogation cmd failed, server: %s:%d, coa: %d, err: %v", serverCfg.Host, serverCfg.Port, v, err)
						continue
					}
					logx.Infof("send counter interrogation cmd, server: %s:%d, coa: %d", serverCfg.Host, serverCfg.Port, v)
				}
			}
		}); err != nil {
			logx.Errorf("add counter interrogation cron failed, cron: %s, err: %v", s.svcCtx.Config.CounterInterrogationCmd, err)
		}
	}
	s.c.Start()
	logx.Infof("starting cron server")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
