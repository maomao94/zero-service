package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/live/internal/svc"
	"zero-service/app/live/live"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/carbonx"

	"github.com/zeromicro/go-zero/core/logx"
)

// toSipProviderInfo 将 GORM 模型转换为 Proto SipProviderInfo。
func toSipProviderInfo(p *gormmodel.LiveSipProvider) *live.SipProviderInfo {
	var numbers []string
	_ = json.Unmarshal([]byte(p.Numbers), &numbers)
	return &live.SipProviderInfo{
		Id:          p.Id,
		Code:        p.Code,
		Name:        p.Name,
		Address:     p.Address,
		Numbers:     numbers,
		Status:      p.Status,
		CreateTime:  carbonx.FormatDateTimeOrEmpty(p.CreateTime),
		SipTrunkId:  p.SipTrunkId,
	}
}

type ListSipProvidersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListSipProvidersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSipProvidersLogic {
	return &ListSipProvidersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListSipProvidersLogic) ListSipProviders(in *live.ListSipProvidersReq) (*live.ListSipProvidersRes, error) {
	providers, err := l.svcCtx.MeetingRepo.ListSipProviders(l.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*live.SipProviderInfo, 0, len(providers))
	for i := range providers {
		items = append(items, toSipProviderInfo(&providers[i]))
	}
	return &live.ListSipProvidersRes{Providers: items}, nil
}
