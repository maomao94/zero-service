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

type RecordEndHookLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordEndHookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordEndHookLogic {
	return &RecordEndHookLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// on_record_end 回调落库（由 oryxgtw 调用，仅内部 Hook）
// 按 uuid 更新状态与产物信息；无记录则忽略（可能 begin 丢失，仅记日志）。
func (l *RecordEndHookLogic) RecordEndHook(in *oryxserver.RecordEndHookReq) (*oryxserver.RecordEndHookRes, error) {
	status := gormmodel.RecordStatusCompleted
	if in.ArtifactCode != 0 {
		status = gormmodel.RecordStatusFailed
	}

	now := time.Now()
	db := l.svcCtx.DB.WithContext(l.ctx)

	var record gormmodel.Record
	err := db.Where("uuid = ?", in.Uuid).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// begin 可能丢失（如回调乱序/服务重启），补插一条完整记录保证生命周期完整
		record = gormmodel.Record{
			UUID:         in.Uuid,
			Vhost:        in.Vhost,
			App:          in.App,
			Stream:       in.Stream,
			Opaque:       in.Opaque,
			Status:       status,
			EndTime:      now,
			ArtifactCode: in.ArtifactCode,
			ArtifactPath: in.ArtifactPath,
			ArtifactURL:  in.ArtifactUrl,
		}
		if err := db.Create(&record).Error; err != nil {
			l.Logger.Errorf("record_end 补插失败: %v, uuid=%s", err, in.Uuid)
			return nil, err
		}
		l.Logger.Infof("record_end 未找到 begin 记录，已补插: uuid=%s, stream=%s, status=%d", in.Uuid, in.Stream, status)
		return &oryxserver.RecordEndHookRes{}, nil
	} else if err != nil {
		l.Logger.Errorf("record_end 查询失败: %v, uuid=%s", err, in.Uuid)
		return nil, err
	}

	updates := map[string]any{
		"status":        status,
		"end_time":      now,
		"artifact_code": in.ArtifactCode,
		"artifact_path": in.ArtifactPath,
		"artifact_url":  in.ArtifactUrl,
	}
	if err := db.Model(&record).Updates(updates).Error; err != nil {
		l.Logger.Errorf("record_end 落库失败: %v, uuid=%s", err, in.Uuid)
		return nil, err
	}

	l.Logger.Infof("record_end 落库成功: uuid=%s, stream=%s, status=%d, artifact_code=%d, artifact_url=%s",
		in.Uuid, in.Stream, status, in.ArtifactCode, in.ArtifactUrl)
	return &oryxserver.RecordEndHookRes{}, nil
}
