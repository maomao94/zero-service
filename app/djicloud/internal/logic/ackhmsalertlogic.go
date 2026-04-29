package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type AckHmsAlertLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAckHmsAlertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AckHmsAlertLogic {
	return &AckHmsAlertLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AckHmsAlert 确认 HMS 告警。
func (l *AckHmsAlertLogic) AckHmsAlert(in *djicloud.AckHmsAlertReq) (*djicloud.CommonRes, error) {
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).Where("id = ?", in.Id).Updates(map[string]any{
		"acked":       1,
		"acked_at":    sql.NullTime{Time: time.Now(), Valid: true},
		"acked_by":    in.AckedBy,
		"update_time": time.Now(),
	})
	if result.Error != nil {
		return &djicloud.CommonRes{Code: -1, Message: result.Error.Error()}, nil
	}
	if result.RowsAffected == 0 {
		return &djicloud.CommonRes{Code: -1, Message: "alert not found"}, nil
	}
	return &djicloud.CommonRes{Code: 0, Message: "success"}, nil
}
