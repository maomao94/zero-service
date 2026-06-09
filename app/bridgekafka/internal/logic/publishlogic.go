package logic

import (
	"context"
	"fmt"

	"zero-service/app/bridgekafka/bridgekafka"
	"zero-service/app/bridgekafka/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PublishLogic) Publish(in *bridgekafka.PublishReq) (*bridgekafka.PublishRes, error) {
	pusher, ok := l.svcCtx.Pushers[in.Topic]
	if !ok {
		return nil, fmt.Errorf("kafka topic %s not configured", in.Topic)
	}
	if in.Key != "" {
		if err := pusher.PushWithKey(l.ctx, in.Key, string(in.Value)); err != nil {
			return nil, err
		}
	} else {
		if err := pusher.Push(l.ctx, string(in.Value)); err != nil {
			return nil, err
		}
	}
	return &bridgekafka.PublishRes{}, nil
}
