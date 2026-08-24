package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/antsx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type StreamRelayStopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStreamRelayStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamRelayStopLogic {
	return &StreamRelayStopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 停止转推（本地任务直接停止；未命中且 cluster 模式时 MQTT 广播停止）
func (l *StreamRelayStopLogic) StreamRelayStop(in *oryxserver.StreamRelayStopReq) (*oryxserver.StreamRelayStopRes, error) {
	if strings.TrimSpace(in.TaskId) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "task_id 不能为空")
	}
	// 1. 本地任务直接停止
	found, err := l.svcCtx.RelayManager.Stop(in.TaskId)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "停止转推失败")
	}
	if found {
		l.Logger.Infof("停止转推成功(本地): task_id=%s", in.TaskId)
		return &oryxserver.StreamRelayStopRes{}, nil
	}
	// 2. 未命中本地：非集群直接报错，集群走 MQTT 广播（payload=taskId 原文，与 executor 解码一致）
	if !l.svcCtx.IsBroadcast() {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, "任务不在当前节点")
	}
	if l.svcCtx.MqttClient == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, "MQTT 未配置，无法停止集群内其他节点的任务")
	}

	_, err = l.svcCtx.Broadcaster.BroadcastReply(l.ctx, relay.MethodStreamRelayStop, []byte(in.TaskId), 10*time.Second)
	if err != nil {
		if errors.Is(err, antsx.ErrReplyExpired) {
			// 超时 = 集群内未命中任何节点（任务不存在或已停止）
			return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, "任务不存在或已停止")
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "广播停止转推失败")
	}
	l.Logger.Infof("停止转推成功(集群广播): task_id=%s", in.TaskId)
	return &oryxserver.StreamRelayStopRes{}, nil
}
