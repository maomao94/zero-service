package logic

import (
	"context"
	"errors"
	"time"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/oryxserver"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RecordBeginHookLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordBeginHookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordBeginHookLogic {
	return &RecordBeginHookLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// on_record_begin 回调落库（由 oryxgtw 调用，仅内部 Hook）
// 幂等：按 uuid 存在则更新（Oryx 可能重发回调），不存在则插入。
func (l *RecordBeginHookLogic) RecordBeginHook(in *oryxserver.RecordBeginHookReq) (*oryxserver.RecordBeginHookRes, error) {
	now := time.Now()
	db := l.svcCtx.DB.WithContext(l.ctx)

	var record gormmodel.Record
	err := db.Where("uuid = ?", in.Uuid).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = gormmodel.Record{
			UUID:      in.Uuid,
			Vhost:     in.Vhost,
			App:       in.App,
			Stream:    in.Stream,
			Opaque:    in.Opaque,
			Status:    gormmodel.RecordStatusRecording,
			BeginTime: now,
		}
		if err := db.Create(&record).Error; err != nil {
			l.Logger.Errorf("record_begin 落库失败: %v, uuid=%s", err, in.Uuid)
			return nil, err
		}
	} else if err != nil {
		l.Logger.Errorf("record_begin 查询失败: %v, uuid=%s", err, in.Uuid)
		return nil, err
	} else {
		// 重发 begin：更新基本信息并回到录制中
		if err := db.Model(&record).Updates(map[string]any{
			"vhost":      in.Vhost,
			"app":        in.App,
			"stream":     in.Stream,
			"opaque":     in.Opaque,
			"status":     gormmodel.RecordStatusRecording,
			"begin_time": now,
		}).Error; err != nil {
			l.Logger.Errorf("record_begin 更新失败: %v, uuid=%s", err, in.Uuid)
			return nil, err
		}
	}

	l.Logger.Infof("record_begin 落库成功: uuid=%s, vhost=%s, app=%s, stream=%s", in.Uuid, in.Vhost, in.App, in.Stream)
	return &oryxserver.RecordBeginHookRes{}, nil
}
