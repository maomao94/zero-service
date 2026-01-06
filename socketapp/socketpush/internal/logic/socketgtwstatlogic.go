package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type SocketGtwStatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSocketGtwStatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SocketGtwStatLogic {
	return &SocketGtwStatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

type NodeSource struct {
	Node string
	Cli  socketgtw.SocketGtwClient
}

// 获取网关统计信息
func (l *SocketGtwStatLogic) SocketGtwStat(in *socketpush.SocketGtwStatReq) (*socketpush.SocketGtwStatRes, error) {
	stats, err := mr.MapReduce(
		func(source chan<- NodeSource) {
			for key, cli := range l.svcCtx.SocketContainer.GetClients() {
				source <- NodeSource{
					Node: key,
					Cli:  cli,
				}
			}
		},
		func(node NodeSource, writer mr.Writer[socketpush.PbSocketGtwStat], cancel func(error)) {
			res, err := node.Cli.SocketGtwStat(l.ctx, &socketgtw.SocketGtwStatReq{})
			if err == nil {
				writer.Write(socketpush.PbSocketGtwStat{
					Node:     node.Node,
					Sessions: res.Sessions,
				})
			}
		},
		func(pipe <-chan socketpush.PbSocketGtwStat, writer mr.Writer[[]*socketpush.PbSocketGtwStat], cancel func(error)) {
			stats := make([]*socketpush.PbSocketGtwStat, 0)
			for nodeStat := range pipe {
				stats = append(stats, &nodeStat)
			}
			writer.Write(stats)
		},
	)
	if err != nil {
		return nil, err
	}
	return &socketpush.SocketGtwStatRes{
		Stats: stats,
	}, nil
}
