package logic

import (
	"context"
	"strings"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/oryxx"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopRelayAndRecordingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopRelayAndRecordingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopRelayAndRecordingLogic {
	return &StopRelayAndRecordingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// StopRelayAndRecording 停止中继拉流并结束关联录制：
// 1. 停止中继流（复用 StopRelayPullLogic）
// 2. 查询录制中的记录（status=1）
// 3. 逐个结束录制（容错，单个失败不影响其他）
func (l *StopRelayAndRecordingLogic) StopRelayAndRecording(in *oryxserver.StopRelayAndRecordingReq) (*oryxserver.StopRelayAndRecordingRes, error) {
	app := in.GetApp()
	stream := in.GetStream()

	// 1. 停止中继流（复用 StopRelayPullLogic）
	stopLogic := NewStopRelayPullLogic(l.ctx, l.svcCtx)
	_, err := stopLogic.StopRelayPull(&oryxserver.StopRelayPullReq{
		App:    app,
		Stream: stream,
	})
	if err != nil {
		l.Errorf("StopRelayPull failed: app=%s, stream=%s, err=%v", app, stream, err)
	} else {
		l.Infof("StopRelayPull succeeded: app=%s, stream=%s", app, stream)
	}

	// 2. 查询所有录制中的记录（status=1: 录制中）
	var records []gormmodel.Record
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.Record{})
	db = db.Where("app = ?", app)
	db = db.Where("stream = ?", stream)
	db = db.Where("status = ?", int32(gormmodel.RecordStatusRecording))
	if err := db.Find(&records).Error; err != nil {
		l.Errorf("RecordList query failed: app=%s, stream=%s, err=%v", app, stream, err)
		return &oryxserver.StopRelayAndRecordingRes{}, nil
	}

	// 3. 逐个停止录制（容错，单个失败不影响其他）
	for _, record := range records {
		uuid := record.UUID
		if err := l.svcCtx.OryxClient.RecordEnd(l.ctx, uuid); err != nil {
			// 判断是否为 "no record task" 错误（幂等安全）
			if oe, ok := oryxx.IsOryxError(err); ok && oe.Code == 500 &&
				strings.Contains(oe.Message, "no record task") {
				l.Infof("RecordEnd skipped (no record task): uuid=%s", uuid)
				continue
			}
			l.Errorf("RecordEnd failed: uuid=%s, err=%v", uuid, err)
		} else {
			l.Infof("RecordEnd succeeded: uuid=%s", uuid)
		}
	}

	return &oryxserver.StopRelayAndRecordingRes{}, nil
}
