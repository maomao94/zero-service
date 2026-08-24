// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package hook

import (
	"context"

	"zero-service/app/oryxgtw/internal/svc"
	"zero-service/app/oryxgtw/internal/types"
	"zero-service/app/oryxserver/oryxserver"

	"github.com/zeromicro/go-zero/core/logx"
)

type HookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Oryx HTTP 回调统一入口（按 action 字段分发）
func NewHookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HookLogic {
	return &HookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HookLogic) Hook(req *types.HookRequest) (resp *types.HookReply, err error) {
	switch req.Action {
	case "on_publish":
		l.Logger.Infof("on_publish request_id=%s, vhost=%s, app=%s, stream=%s", req.RequestId, req.Vhost, req.App, req.Stream)
	case "on_unpublish":
		l.Logger.Infof("on_unpublish request_id=%s, vhost=%s, app=%s, stream=%s", req.RequestId, req.Vhost, req.App, req.Stream)
	case "on_record_begin":
		l.Logger.Infof("on_record_begin request_id=%s, stream=%s, uuid=%s", req.RequestId, req.Stream, req.Uuid)
		l.callRecordBegin(req)
	case "on_record_end":
		l.Logger.Infof("on_record_end request_id=%s, stream=%s, uuid=%s, artifact_code=%d, artifact_path=%s, artifact_url=%s",
			req.RequestId, req.Stream, req.Uuid, req.ArtifactCode, req.ArtifactPath, req.ArtifactUrl)
		l.callRecordEnd(req)
	case "on_ocr":
		l.Logger.Infof("on_ocr request_id=%s, stream=%s, uuid=%s", req.RequestId, req.Stream, req.Uuid)
	default:
		l.Logger.Infof("unknown action request_id=%s, action=%s, stream=%s", req.RequestId, req.Action, req.Stream)
	}

	return &types.HookReply{Code: 0}, nil
}

// callRecordBegin 回调 oryxserver 落库 on_record_begin（失败仅记日志，不影响回调响应）
func (l *HookLogic) callRecordBegin(req *types.HookRequest) {
	if _, err := l.svcCtx.OryxServerClient.RecordBeginHook(l.ctx, &oryxserver.RecordBeginHookReq{
		RequestId: req.RequestId,
		Opaque:    req.Opaque,
		Vhost:     req.Vhost,
		App:       req.App,
		Stream:    req.Stream,
		Uuid:      req.Uuid,
	}); err != nil {
		l.Logger.Errorf("RecordBeginHook 调用失败: %v, uuid=%s", err, req.Uuid)
	}
}

// callRecordEnd 回调 oryxserver 落库 on_record_end（失败仅记日志，不影响回调响应）
func (l *HookLogic) callRecordEnd(req *types.HookRequest) {
	if _, err := l.svcCtx.OryxServerClient.RecordEndHook(l.ctx, &oryxserver.RecordEndHookReq{
		RequestId:    req.RequestId,
		Opaque:       req.Opaque,
		Vhost:        req.Vhost,
		App:          req.App,
		Stream:       req.Stream,
		Uuid:         req.Uuid,
		ArtifactCode: int32(req.ArtifactCode),
		ArtifactPath: req.ArtifactPath,
		ArtifactUrl:  req.ArtifactUrl,
	}); err != nil {
		l.Logger.Errorf("RecordEndHook 调用失败: %v, uuid=%s", err, req.Uuid)
	}
}
