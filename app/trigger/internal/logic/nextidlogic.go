package logic

import (
	"context"
	"fmt"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
)

type NextIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNextIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NextIdLogic {
	return &NextIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *NextIdLogic) NextId(in *trigger.NextIdReq) (*trigger.NextIdRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	var nextId string
	if in.Separate {
		nextId, err = l.svcCtx.IdUtil.NextId(in.OutDescType, l.svcCtx.Config.Name)
		if err != nil {
			return nil, err
		}
	} else {
		nextId, err = l.svcCtx.IdUtil.NextId("P", l.svcCtx.Config.Name)
		if err != nil {
			return nil, err
		}
		nextId = fmt.Sprintf("%s%s", in.OutDescType, strutil.After(nextId, "P"))
	}
	return &trigger.NextIdRes{
		NextId: nextId,
	}, nil
}
