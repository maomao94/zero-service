package logic

import (
	"context"
	"zero-service/common/copierx"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

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
	// 构建查询
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.ModbusSlaveConfig{})
	if len(in.Keyword) > 0 {
		db = db.Where("modbus_code LIKE ?", in.Keyword+"%")
	}
	if in.Status > 0 {
		db = db.Where("status = ?", in.Status)
	}

	// 使用 gormx 分页
	var list []gormmodel.ModbusSlaveConfig
	pageResult, err := gormx.QueryPage(db, int(in.Page), int(in.PageSize), &list)
	if err != nil {
		return nil, err
	}

	var pbList []*bridgemodbus.PbModbusConfig
	_ = copier.CopyWithOption(&pbList, list, copierx.Option)
	return &bridgemodbus.PageListConfigRes{
		Total: uint32(pageResult.Total),
		Cfg:   pbList,
	}, nil
}
