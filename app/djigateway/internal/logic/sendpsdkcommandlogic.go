package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendPsdkCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendPsdkCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPsdkCommandLogic {
	return &SendPsdkCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SendPsdkCommand 通过 psdk_write 向 PSDK 负载写入数据。
func (l *SendPsdkCommandLogic) SendPsdkCommand(in *djigateway.PsdkCommandReq) (*djigateway.CommonRes, error) {
	l.Infof("[psdk-write] sn=%s payload_index=%s data_len=%d", in.DeviceSn, in.PayloadIndex, len(in.Data))

	if in.Data == "" {
		return &djigateway.CommonRes{
			Code:    -1,
			Message: "data is required",
		}, nil
	}

	var tid string
	var err error
	if in.PayloadIndex == "" {
		tid, err = l.svcCtx.DjiClient.SendPsdkCommand(l.ctx, in.DeviceSn, in.Data)
	} else {
		tid, err = l.svcCtx.DjiClient.SendPsdkCommandWithIndex(l.ctx, in.DeviceSn, in.PayloadIndex, in.Data)
	}
	if err != nil {
		l.Errorf("[psdk-write] send failed: %v", err)
		return errRes(tid, err), nil
	}

	return okRes(tid), nil
}
