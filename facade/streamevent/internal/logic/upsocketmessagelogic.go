package logic

import (
	"context"
	"zero-service/common/ctxdata"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpSocketMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpSocketMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpSocketMessageLogic {
	return &UpSocketMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 上行socket标准消息, 可以用于__connection__/__disconnect__/__join_room_up__/__up__/和自定义up事件
func (l *UpSocketMessageLogic) UpSocketMessage(in *streamevent.UpSocketMessageReq) (*streamevent.UpSocketMessageRes, error) {
	token := ctxdata.GetAuthorization(l.ctx)
	l.Logger.Infof("token: %s", token)
	// 给一个 json  string  测试
	var downPayload = struct {
		Str_0   string            `json:"str"`
		Int_1   int               `json:"int"`
		Slice_2 []string          `json:"slice"`
		Map_3   map[string]string `json:"map"`
	}{
		Str_0:   "hello world",
		Int_1:   123,
		Slice_2: []string{"a", "b", "c"},
		Map_3: map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
		},
	}
	data, err := jsonx.Marshal(&downPayload)
	if err != nil {
		return nil, err
	}
	return &streamevent.UpSocketMessageRes{
		Payload: string(data),
	}, nil
}
