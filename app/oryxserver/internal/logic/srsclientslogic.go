package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsClientsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsClientsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsClientsLogic {
	return &SrsClientsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 客户端列表（GET /api/v1/clients，start/count 分页透传）
func (l *SrsClientsLogic) SrsClients(in *oryxserver.SrsClientsReq) (*oryxserver.SrsClientsRes, error) {
	data, err := l.svcCtx.OryxClient.SrsClients(l.ctx, in.Start, in.Count)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}

	items := make([]*oryxserver.SrsClientItem, 0, len(data.Clients))
	for _, cl := range data.Clients {
		items = append(items, &oryxserver.SrsClientItem{
			Id:        cl.ID,
			Vhost:     cl.Vhost,
			Stream:    cl.Stream,
			Ip:        cl.IP,
			PageUrl:   cl.PageURL,
			SwfUrl:    cl.SwfURL,
			TcUrl:     cl.TcURL,
			Url:       cl.URL,
			Name:      cl.Name,
			Type:      cl.Type,
			Publish:   cl.Publish,
			Alive:     cl.Alive,
			SendBytes: cl.SendBytes,
			RecvBytes: cl.RecvBytes,
			Kbps: &oryxserver.SrsKbps{
				Recv_30S: cl.Kbps.Recv30s,
				Send_30S: cl.Kbps.Send30s,
			},
		})
	}

	return &oryxserver.SrsClientsRes{
		Server:  data.Server,
		Service: data.Service,
		Pid:     data.Pid,
		Clients: items,
	}, nil
}
