package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
)

type BatchNextIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchNextIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchNextIdLogic {
	return &BatchNextIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量顺序生成业务唯一编码，用于业务批量插入前预生成编码
func (l *BatchNextIdLogic) BatchNextId(in *trigger.BatchNextIdReq) (*trigger.BatchNextIdRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	var nextIds []string
	var err error
	if in.Separate {
		nextIds, err = l.svcCtx.IdUtil.NextIds(in.OutDescType, l.svcCtx.Config.Name, int64(in.Count))
		if err != nil {
			return nil, err
		}
	} else {
		nextIds, err = l.svcCtx.IdUtil.NextIds("P", l.svcCtx.Config.Name, int64(in.Count))
		if err != nil {
			return nil, err
		}
		for i, nextId := range nextIds {
			nextIds[i] = fmt.Sprintf("%s%s", in.OutDescType, strutil.After(nextId, "P"))
		}
	}

	return &trigger.BatchNextIdRes{
		NextIds: nextIds,
	}, nil
}
