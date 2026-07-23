package logic

import (
	"context"

	"zero-service/app/ispserver/internal/svc"
	"zero-service/app/ispserver/ispserver"
	"zero-service/common/gnetx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionsLogic {
	return &ListSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListSessionsLogic) ListSessions(in *ispserver.ListSessionsReq) (*ispserver.ListSessionsRes, error) {
	sessions := l.svcCtx.IspServer.Manager().All()
	infos := make([]*ispserver.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		clientID := ""
		if sc, ok := s.(gnetx.ServerConn); ok {
			clientID = sc.ClientID()
		}
		infos = append(infos, &ispserver.SessionInfo{
			SessionId:   s.SessionID(),
			DeviceCode:  clientID,
			RemoteAddr:  s.RemoteAddr().String(),
			ConnectedAt: s.CreatedAt().Unix(),
		})
	}
	return &ispserver.ListSessionsRes{Sessions: infos}, nil
}
