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
	// 统计 session
	_, _ = s.c.AddFunc("*/60 * * * * *", func() {
		sessionLen := s.svcCtx.ClientManager.GetSessionLen()
		logx.Infof("stats session len:%d", sessionLen)

		clis := s.svcCtx.ClientManager.GetSessionClients()
		for _, v := range clis {
			if !v.IsConnected() {
				logx.Errorf("stats iec104 server addr:%s connect error", v.GetServerUrl())
			}
		}
	})

	// 测试 发送一次 read
	_, _ = s.c.AddFunc("*/5 * * * * *", func() {
		// read cmd
		cli, err := s.svcCtx.ClientManager.GetDefaultSessionClient()
		if err != nil {
			logx.Errorf("error GetDefaultSessionClient %v", err)
		}
		// read cmd
		if err := cli.SendReadCmd(1, 1); err != nil {
			logx.Errorf("send counter interrogation cmd error %v\n", err)
		}
	})
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
