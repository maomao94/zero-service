package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsStreamsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsStreamsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsStreamsLogic {
	return &SrsStreamsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 流列表（GET /api/v1/streams，start/count 分页透传）
func (l *SrsStreamsLogic) SrsStreams(in *oryxserver.SrsStreamsReq) (*oryxserver.SrsStreamsRes, error) {
	data, err := l.svcCtx.OryxClient.SrsStreams(l.ctx, in.Start, in.Count)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}

	items := make([]*oryxserver.SrsStreamItem, 0, len(data.Streams))
	for _, s := range data.Streams {
		item := &oryxserver.SrsStreamItem{
			Id:        s.ID,
			Name:      s.Name,
			Vhost:     s.Vhost,
			App:       s.App,
			TcUrl:     s.TcUrl,
			Url:       s.URL,
			LiveMs:    s.LiveMs,
			Clients:   s.Clients,
			Frames:    s.Frames,
			SendBytes: s.SendBytes,
			RecvBytes: s.RecvBytes,
			Kbps: &oryxserver.SrsKbps{
				Recv_30S: s.Kbps.Recv30s,
				Send_30S: s.Kbps.Send30s,
			},
			Publish: &oryxserver.SrsPublishInfo{
				Active: s.Publish.Active,
				Cid:    s.Publish.Cid,
			},
		}
		if s.Video != nil {
			item.Video = &oryxserver.SrsVideoInfo{
				Codec:   s.Video.Codec,
				Profile: s.Video.Profile,
				Level:   s.Video.Level,
				Width:   s.Video.Width,
				Height:  s.Video.Height,
			}
		}
		if s.Audio != nil {
			item.Audio = &oryxserver.SrsAudioInfo{
				Codec:      s.Audio.Codec,
				SampleRate: s.Audio.SampleRate,
				Channel:    s.Audio.Channel,
				Profile:    s.Audio.Profile,
			}
		}
		items = append(items, item)
	}

	return &oryxserver.SrsStreamsRes{
		Server:  data.Server,
		Service: data.Service,
		Pid:     data.Pid,
		Streams: items,
	}, nil
}
