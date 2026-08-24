package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordDeleteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordDeleteLogic {
	return &RecordDeleteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除录制记录（平台级接口，仅删除本地生命周期记录，不影响 Oryx 文件）
func (l *RecordDeleteLogic) RecordDelete(in *oryxserver.RecordDeleteReq) (*oryxserver.RecordDeleteRes, error) {
	if in.Uuid == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "录制任务 UUID 不能为空")
	}

	db := l.svcCtx.DB.WithContext(l.ctx)
	if err := db.Where("uuid = ?", in.Uuid).Delete(&gormmodel.Record{}).Error; err != nil {
		l.Logger.Errorf("删除录制记录失败: %v, uuid=%s", err, in.Uuid)
		return nil, err
	}

	l.Logger.Infof("删除录制记录成功: uuid=%s", in.Uuid)
	return &oryxserver.RecordDeleteRes{}, nil
}
