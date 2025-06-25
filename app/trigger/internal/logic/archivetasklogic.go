package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type ArchiveTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewArchiveTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArchiveTaskLogic {
	return &ArchiveTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 归档任务
func (l *ArchiveTaskLogic) ArchiveTask(in *trigger.ArchiveTaskReq) (*trigger.ArchiveTaskRes, error) {
	if err := l.svcCtx.AsynqInspector.ArchiveTask(in.Queue, in.Id); err != nil {
		return nil, err
	}
	return &trigger.ArchiveTaskRes{}, nil
}
