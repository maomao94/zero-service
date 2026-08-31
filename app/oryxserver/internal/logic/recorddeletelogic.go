package logic

import (
	"context"
	"strings"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/model/gormmodel"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/oryxx"
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

// RecordDelete 删除录制记录（平台级接口）：
// 1. 调用 Oryx RecordRemove 删除对应录制文件
// 2. Oryx 返回 "no record for"（任务/文件已不存在）视为幂等安全，继续删除本地记录
// 3. 其他 Oryx 调用失败返回错误，不删除本地记录
func (l *RecordDeleteLogic) RecordDelete(in *oryxserver.RecordDeleteReq) (*oryxserver.RecordDeleteRes, error) {
	if in.Uuid == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "录制任务 UUID 不能为空")
	}

	// 1. 删除 Oryx 上的录制文件
	if err := l.svcCtx.OryxClient.RecordRemove(l.ctx, in.Uuid); err != nil {
		// 任务不存在（幂等安全）：Oryx 返回 code=500 + "no record for uuid=..."
		if oe, ok := oryxx.IsOryxError(err); ok && oe.Code == 500 &&
			strings.Contains(oe.Message, "no record for") {
			l.Infof("Oryx 无对应录制（幂等，跳过删除）: uuid=%s", in.Uuid)
		} else {
			l.Errorf("调用 Oryx RecordRemove 失败: %v, uuid=%s", err, in.Uuid)
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx 删除录制文件失败")
		}
	}

	db := l.svcCtx.DB.WithContext(l.ctx)
	if err := db.Where("uuid = ?", in.Uuid).Delete(&gormmodel.Record{}).Error; err != nil {
		l.Logger.Errorf("删除录制记录失败: %v, uuid=%s", err, in.Uuid)
		return nil, err
	}

	l.Logger.Infof("删除录制记录成功: uuid=%s", in.Uuid)
	return &oryxserver.RecordDeleteRes{}, nil
}
