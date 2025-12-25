package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearPointMappingCacheLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClearPointMappingCacheLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearPointMappingCacheLogic {
	return &ClearPointMappingCacheLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 清除点位绑定缓存（支持批量）
func (l *ClearPointMappingCacheLogic) ClearPointMappingCache(in *ieccaller.ClearPointMappingCacheReq) (*ieccaller.ClearPointMappingCacheRes, error) {
	clearedCount := int64(0)
	if l.svcCtx.DevicePointMappingModel == nil {
		return &ieccaller.ClearPointMappingCacheRes{}, nil
	}
	if len(in.Keys) > 0 {
		for _, key := range in.Keys {
			if _, exists := l.svcCtx.DevicePointMappingModel.GetCache(l.ctx, key); exists {
				if err := l.svcCtx.DevicePointMappingModel.RemoveCache(l.ctx, key); err != nil {
					return nil, err
				}
				clearedCount++
			}
		}
	}
	if len(in.KeyInfos) > 0 {
		for _, info := range in.KeyInfos {
			key := l.svcCtx.DevicePointMappingModel.GenerateCacheKey(info.TagStation, info.Coa, info.Ioa)
			if _, exists := l.svcCtx.DevicePointMappingModel.GetCache(l.ctx, key); exists {
				if err := l.svcCtx.DevicePointMappingModel.RemoveCache(l.ctx, key); err != nil {
					return nil, err
				}
				clearedCount++
			}
		}
	}
	if err := l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_ClearPointMappingCache_FullMethodName, in); err != nil {
		l.Errorf("Broadcast cache clear failed, err: %v", err)
	}
	l.Infof("Clear point mapping cache, cleared count: %d", clearedCount)
	return &ieccaller.ClearPointMappingCacheRes{
		ClearedCount: clearedCount,
	}, nil
}
