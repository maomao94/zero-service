package logic

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// ListSessions 列出会话
func (l *ListSessionsLogic) ListSessions(in *aisolo.ListSessionsRequest) (*aisolo.ListSessionsResponse, error) {
	// 验证必填参数
	userID := in.UserId
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	page := int(in.Page)
	if page <= 0 {
		page = 1
	}

	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	sessions, total, err := GlobalSessionStore.List(l.ctx, userID, page, pageSize)
	if err != nil {
		l.Errorf("list sessions failed: %v", err)
		return nil, err
	}

	return &aisolo.ListSessionsResponse{
		Sessions:   sessions,
		Total:      total,
		Page:       int32(page),
		TotalPages: int32((int(total) + pageSize - 1) / pageSize),
	}, nil
}
