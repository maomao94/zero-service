package logic

import (
	"context"
	"zero-service/app/bridgedump/bridgedump"
	"zero-service/app/bridgedump/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

var cableWorkListDataFile = "cable_work_list"

type CableWorkListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCableWorkListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CableWorkListLogic {
	return &CableWorkListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 电缆设备运行数据接入
func (l *CableWorkListLogic) CableWorkList(in *bridgedump.CableWorkListReq) (*bridgedump.CableWorkListRes, error) {
	_, err := l.svcCtx.DumpBridgeData(l.ctx, l.svcCtx.Config.DumpPath, cableWorkListDataFile, in)
	if err != nil {
		return nil, err
	}
	return &bridgedump.CableWorkListRes{
		Code: 200,
		Msg:  "成功",
	}, nil
}
