package cron

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"zero-service/app/xfusionmock/internal/logic"
	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"
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

	_, _ = s.c.AddFunc(s.svcCtx.Config.PushCron, func() {
		in := xfusionmock.ReqPushTest{}
		logic.NewPushTestLogic(context.Background(), s.svcCtx).PushTest(&in)
	})
	_, _ = s.c.AddFunc(s.svcCtx.Config.PushCronPoint, func() {
		in := xfusionmock.ReqPushPoint{}
		logic.NewPushPointLogic(context.Background(), s.svcCtx).PushPoint(&in)
	})
	_, _ = s.c.AddFunc(s.svcCtx.Config.PushCron, func() {
		in := xfusionmock.ReqPushAlarm{}
		logic.NewPushAlarmLogic(context.Background(), s.svcCtx).PushAlarm(&in)
	})
	_, _ = s.c.AddFunc(s.svcCtx.Config.PushCron, func() {
		in := xfusionmock.ReqPushEvent{}
		logic.NewPushEventLogic(context.Background(), s.svcCtx).PushEvent(&in)
	})
	_, _ = s.c.AddFunc(s.svcCtx.Config.PushCron, func() {
		in := xfusionmock.ReqPushTerminalBind{}
		logic.NewPushTerminalBindLogic(context.Background(), s.svcCtx).PushTerminalBind(&in)
	})
	s.c.Start()
	fmt.Print("Starting cron server \n")
}

func (s *CronService) Stop() {
	s.c.Stop()
}
