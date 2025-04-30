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
		sessionLen := s.svcCtx.ClientManager.GetSessionLen()
		clients := s.svcCtx.ClientManager.GetClients()
		clientsLen := len(clients)
		logx.Statf("(iec104) clientLen: %d, sessionLen: %d", clientsLen, sessionLen)
		loss := 0
		for v := range clients {
			if !v.IsConnected() {
				loss++
			}
		}
		logx.Statf("(iec104) clientLen: %d, sessionLen: %d, loss: %d", clientsLen, sessionLen, loss)
	})
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
