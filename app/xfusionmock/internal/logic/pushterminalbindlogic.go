package logic

import (
	"context"
	"encoding/json"
	"github.com/golang-module/carbon/v2"
	"zero-service/model"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushTerminalBindLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushTerminalBindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushTerminalBindLogic {
	return &PushTerminalBindLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PushTerminalBindLogic) PushTerminalBind(in *xfusionmock.ReqPushTerminalBind) (*xfusionmock.ResPushTerminalBind, error) {
	l.Info("PushTerminalBind")
	data := model.TerminalBind{
		DataTagV1:  l.svcCtx.Config.Name,
		Action:     "BIND",
		TerminalID: 100001,
		TerminalNo: "T12345678901",
		TrackID:    5001,
		TrackNo:    "沪A12345",
		TrackType:  "CAR",
		TrackName:  l.svcCtx.Config.Name,
		ActionTime: carbon.Now().Format("Y-m-d H:i:s"),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	l.svcCtx.KafkaTerminalBindPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushTerminalBind{}, nil
}
