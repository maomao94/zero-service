package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type BroadcastGlobalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBroadcastGlobalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BroadcastGlobalLogic {
	return &BroadcastGlobalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向所有在线前端广播消息
func (l *BroadcastGlobalLogic) BroadcastGlobal(in *socketpush.BroadcastGlobalReq) (*socketpush.BroadcastGlobalRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			//testpayload := map[string]any{}
			//testpayload["string"] = "string"
			//testpayload["list"] = []string{"1", "2", "3"}
			//testpayload["map"] = map[string]string{"1": "1", "2": "2", "3": "3"}
			//testpayload["int"] = 1
			//testpayload["float"] = 1.1
			//testpayload["bool"] = true
			//testpayload["nil"] = nil
			//testpayload["struct"] = struct {
			//	Name string
			//	Age  int
			//}{
			//	Name: "test",
			//	Age:  1,
			//}
			//list := []map[string]any{}
			//list = append(list, testpayload)
			//list = append(list, testpayload)
			//b, _ := jsonx.Marshal(list)
			threading.GoSafe(func() {
				cli.BroadcastGlobal(baseCtx, &socketgtw.BroadcastGlobalReq{
					ReqId:   in.ReqId,
					Event:   in.Event,
					Payload: in.Payload,
				})
			})
		})
	}
	return &socketpush.BroadcastGlobalRes{
		ReqId: in.ReqId,
	}, nil
}
