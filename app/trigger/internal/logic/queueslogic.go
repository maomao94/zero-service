package logic

import (
	"context"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueuesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueuesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueuesLogic {
	return &QueuesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取队列列表
func (l *QueuesLogic) Queues(in *trigger.QueuesReq) (*trigger.QueuesRes, error) {
	queues, err := l.svcCtx.AsynqInspector.Queues()
	if err != nil {
		return nil, err
	}
	return &trigger.QueuesRes{
		Queues: queues,
	}, nil
}
