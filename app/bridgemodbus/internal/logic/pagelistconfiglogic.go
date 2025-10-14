package logic

import (
	"context"
	"zero-service/common/copierx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/Masterminds/squirrel"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type PageListConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPageListConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageListConfigLogic {
	return &PageListConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询配置列表
func (l *PageListConfigLogic) PageListConfig(in *bridgemodbus.PageListConfigReq) (*bridgemodbus.PageListConfigRes, error) {
	whereBuilder := l.svcCtx.ModbusSlaveConfigModel.SelectBuilder()
	if len(in.Keyword) > 0 {
		whereBuilder = whereBuilder.Where(squirrel.Like{
			"modbus_code": in.Keyword + "%",
		})
	}
	if in.Status > 0 {
		whereBuilder = whereBuilder.Where(squirrel.Eq{
			"status": in.Status,
		})
	}
	list, total, err := l.svcCtx.ModbusSlaveConfigModel.FindPageListByPageWithTotal(l.ctx, whereBuilder, in.Page, in.PageSize)
	if err != nil {
		return nil, err
	}
	var pbList []*bridgemodbus.PbModbusConfig
	_ = copier.CopyWithOption(&pbList, list, copierx.Option)
	return &bridgemodbus.PageListConfigRes{
		Total: uint32(total),
		Cfg:   pbList,
	}, nil
}
