package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type SrsSummariesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSrsSummariesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SrsSummariesLogic {
	return &SrsSummariesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 SRS 系统状态（GET /api/v1/summaries）
func (l *SrsSummariesLogic) SrsSummaries(in *oryxserver.SrsSummariesReq) (*oryxserver.SrsSummariesRes, error) {
	data, err := l.svcCtx.OryxClient.SrsSummaries(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 SRS API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 SRS API 失败")
	}

	sys := &oryxserver.SrsSystemInfo{
		CpuPercent:      data.System.CPUPercent,
		DiskReadKbps:    data.System.DiskReadKBps,
		DiskWriteKbps:   data.System.DiskWriteKBps,
		DiskBusyPercent: data.System.DiskBusyPercent,
		MemRamKbyte:     data.System.MemRamKbyte,
		MemRamPercent:   data.System.MemRamPercent,
		MemSwapKbyte:    data.System.MemSwapKbyte,
		MemSwapPercent:  data.System.MemSwapPercent,
		Cpus:            data.System.Cpus,
		CpusOnline:      data.System.CpusOnline,
		Uptime:          data.System.Uptime,
		IdleTime:        data.System.IdleTime,
		Load_1M:         data.System.Load1m,
		Load_5M:         data.System.Load5m,
		Load_15M:        data.System.Load15m,
		NetSampleTime:   data.System.NetSampleTime,
		NetRecvBytes:    data.System.NetRecvBytes,
		NetSendBytes:    data.System.NetSendBytes,
		NetRecviBytes:   data.System.NetRecviBytes,
		NetSendiBytes:   data.System.NetSendiBytes,
		SrsSampleTime:   data.System.SrsSampleTime,
		SrsRecvBytes:    data.System.SrsRecvBytes,
		SrsSendBytes:    data.System.SrsSendBytes,
		ConnSys:         data.System.ConnSys,
		ConnSysEt:       data.System.ConnSysET,
		ConnSysTw:       data.System.ConnSysTW,
		ConnSysUdp:      data.System.ConnSysUDP,
		ConnSrs:         data.System.ConnSrs,
	}

	return &oryxserver.SrsSummariesRes{
		Ok:    data.OK,
		NowMs: data.NowMs,
		Self: &oryxserver.SrsSelfInfo{
			Version:    data.Self.Version,
			Pid:        data.Self.Pid,
			Ppid:       data.Self.Ppid,
			Argv:       data.Self.Argv,
			Cwd:        data.Self.Cwd,
			MemKbyte:   data.Self.MemKbyte,
			MemPercent: data.Self.MemPercent,
			CpuPercent: data.Self.CPUPercent,
			SrsUptime:  data.Self.SrsUptime,
		},
		System: sys,
	}, nil
}
