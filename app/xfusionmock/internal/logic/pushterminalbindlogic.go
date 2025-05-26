package logic

import (
	"context"
	"encoding/json"
	"github.com/duke-git/lancet/v2/random"
	"github.com/golang-module/carbon/v2"
	"zero-service/model"

	"zero-service/app/xfusionmock/internal/svc"
	"zero-service/app/xfusionmock/xfusionmock"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	terminalList = []string{"600000000001"}
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
	var jsonData []byte
	var err error
	if in.PushMode {
		jsonData, err = json.Marshal(in.Data)
		if err != nil {
			return nil, err
		}
	} else {
		data := model.TerminalBind{
			DataTagV1:     l.svcCtx.Config.Name,
			Action:        "BIND",
			TerminalID:    100001,
			TerminalNo:    randomTerminal(),
			StaffIdCardNo: "11011100011",
			TrackID:       5001,
			TrackNo:       randomUserId(),
			TrackType:     "CAR",
			TrackName:     l.svcCtx.Config.Name,
			ActionTime:    carbon.Now().Format("Y-m-d H:i:s"),
		}
		jsonData, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	l.svcCtx.KafkaTerminalBindPusher.Push(l.ctx, string(jsonData))
	return &xfusionmock.ResPushTerminalBind{}, nil
}

func randomTerminal() string {
	return string(random.RandFromGivenSlice(terminalList))
}
