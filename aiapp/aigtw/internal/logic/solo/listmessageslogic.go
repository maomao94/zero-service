package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMessagesLogic {
	return &ListMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMessagesLogic) ListMessages(req *types.SoloListMessagesRequest) (*types.SoloListMessagesResponse, error) {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return nil, errors.New("missing user id in context")
	}
	resp, err := l.svcCtx.AiSoloCli.ListMessages(l.ctx, &aisolo.ListMessagesReq{
		SessionId: req.SessionId,
		UserId:    userID,
		Limit:     int32(req.Limit),
	})
	if err != nil {
		return nil, err
	}
	out := &types.SoloListMessagesResponse{Total: int(resp.GetTotal())}
	for _, m := range resp.GetMessages() {
		out.Messages = append(out.Messages, messageToType(m))
	}
	return out, nil
}
