package logic

import (
	"context"
	"errors"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/hooks"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type IsDeviceOnlineLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIsDeviceOnlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsDeviceOnlineLogic {
	return &IsDeviceOnlineLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// IsDeviceOnline 查询设备在线状态。
// 内存缓存用于高频在线判断，命中在线时直接返回；缓存未命中时读取数据库快照，并按 last_online_at 做懒过期清理。
func (l *IsDeviceOnlineLogic) IsDeviceOnline(in *djicloud.IsDeviceOnlineReq) (*djicloud.IsDeviceOnlineRes, error) {
	online := hooks.IsOnline(l.svcCtx.OnlineCache, in.DeviceSn)
	if online {
		return &djicloud.IsDeviceOnlineRes{IsOnline: true}, nil
	}
	now := time.Now()
	var device gormmodel.DjiDevice
	err := l.svcCtx.DB.WithContext(l.ctx).Where("device_sn = ?", in.DeviceSn).First(&device).Error
	if err == nil {
		if deviceOnlineExpired(&device, now) {
			return &djicloud.IsDeviceOnlineRes{IsOnline: false}, nil
		}
		return &djicloud.IsDeviceOnlineRes{IsOnline: true}, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &djicloud.IsDeviceOnlineRes{IsOnline: false}, nil
	}
	return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询设备在线状态失败")
}
