package logic

import (
	"context"
	"strconv"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsVhostsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsVhostsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsVhostsLogic {
	return &SrsVhostsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 虚拟主机列表（GET /api/v1/vhosts，无分页全量）
func (l *SrsVhostsLogic) SrsVhosts(in *oryxserver.SrsVhostsReq) (*oryxserver.SrsVhostsRes, error) {
	data, err := l.svcCtx.OryxClient.SrsVhosts(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}

	items := make([]*oryxserver.SrsVhostItem, 0, len(data.Vhosts))
	for _, v := range data.Vhosts {
		id, _ := strconv.ParseInt(v.ID, 10, 64)
		items = append(items, &oryxserver.SrsVhostItem{
			Id:        id,
			Name:      v.Name,
			Enabled:   v.Enabled,
			Clients:   v.Clients,
			Streams:   v.Streams,
			SendBytes: v.SendBytes,
			RecvBytes: v.RecvBytes,
			Kbps: &oryxserver.SrsKbps{
				Recv_30S: v.Kbps.Recv30s,
				Send_30S: v.Kbps.Send30s,
			},
			HlsEnabled:  v.HLS.Enabled,
			HlsFragment: int32(v.HLS.Fragment),
		})
	}

	return &oryxserver.SrsVhostsRes{
		Server:  data.Server,
		Service: data.Service,
		Pid:     data.Pid,
		Vhosts:  items,
	}, nil
}
