package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateFenceCellsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateFenceCellsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateFenceCellsLogic {
	return &GenerateFenceCellsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 一次性生成围栏 cells（小围栏）
func (l *GenerateFenceCellsLogic) GenerateFenceCells(in *geo.GenFenceCellsReq) (*geo.GenFenceCellsRes, error) {
	// todo: add your logic here and delete this line

	return &geo.GenFenceCellsRes{}, nil
}
