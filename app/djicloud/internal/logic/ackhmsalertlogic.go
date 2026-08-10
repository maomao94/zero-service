package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
func (l *AckHmsAlertLogic) AckHmsAlert(in *djicloud.AckHmsAlertReq) (*djicloud.AckHmsAlertRes, error) {
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiHmsAlert{}).Where("id = ?", in.Id).Updates(map[string]any{
		"acked":       1,
		"acked_at":    sql.NullTime{Time: time.Now(), Valid: true},
		"acked_by":    in.AckedBy,
		"update_time": time.Now(),
	})
	if result.Error != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, result.Error, "确认 HMS 告警失败")
	}
	if result.RowsAffected == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "HMS 告警不存在")
	}
	return &djicloud.AckHmsAlertRes{}, nil
}
