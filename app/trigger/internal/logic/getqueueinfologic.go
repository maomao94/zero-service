package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"zero-service/common/copierx"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetQueueInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetQueueInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetQueueInfoLogic {
	return &GetQueueInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取队列信息
func (l *GetQueueInfoLogic) GetQueueInfo(in *trigger.GetQueueInfoReq) (*trigger.GetQueueInfoRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	qinfo, err := l.svcCtx.AsynqInspector.GetQueueInfo(in.Queue)
	if err != nil {
		return nil, err
	}
	queueInfo := trigger.PbQueueInfo{}
	copier.CopyWithOption(&queueInfo, qinfo, copierx.Option)
	return &trigger.GetQueueInfoRes{
		QueueInfo: &queueInfo,
	}, nil
}
