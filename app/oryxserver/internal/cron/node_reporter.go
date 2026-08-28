package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zero-service/app/oryxserver/internal/svc"

	"github.com/duke-git/lancet/v2/netutil"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	nodeSetKey    = "oryx:nodes"
	nodeInfoKey   = "oryx:node:%s"  // nodeID 本身不含 "oryx:node:" 前缀
	nodeExpireSec = 5
	reportInterval = time.Second
)

// NodeReporter 每秒上报当前节点信息到 Redis。
type NodeReporter struct {
	svcCtx     *svc.ServiceContext
	cancelChan chan struct{}
	nodeID     string
	ip         string
}

func NewNodeReporter(svcCtx *svc.ServiceContext) *NodeReporter {
	// nodeID 格式 "oryx-node-{uuid}"，Redis key 用短 ID（去掉前缀避免重复）
	nodeID := svcCtx.NodeID
	if strings.HasPrefix(nodeID, "oryx-node-") {
		nodeID = strings.TrimPrefix(nodeID, "oryx-node-")
	}
	return &NodeReporter{
		svcCtx: svcCtx,
		nodeID: nodeID,
		ip:     netutil.GetInternalIp(),
	}
}

func (r *NodeReporter) Start() {
	if r.cancelChan != nil {
		return
	}
	r.cancelChan = make(chan struct{})
	logx.Infof("[node-reporter] started: node_id=%s ip=%s", r.nodeID, r.ip)
	go r.loop()
}

func (r *NodeReporter) Stop() {
	if r.cancelChan != nil {
		close(r.cancelChan)
		r.cancelChan = nil
		logx.Infof("[node-reporter] stopped: node_id=%s", r.nodeID)
	}
}

func (r *NodeReporter) loop() {
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.cancelChan:
			return
		case <-ticker.C:
			r.report()
		}
	}
}

func (r *NodeReporter) report() {
	ctx := context.Background()
	rc := r.svcCtx.RelayRedis

	membersKey := fmt.Sprintf(nodeInfoKey, r.nodeID)
	ts := fmt.Sprintf("%d", time.Now().Unix())

	if err := rc.HsetCtx(ctx, membersKey, "ip", r.ip); err != nil {
		logx.Errorf("[node-reporter] hset ip failed: %v", err)
		return
	}
	if err := rc.HsetCtx(ctx, membersKey, "ts", ts); err != nil {
		logx.Errorf("[node-reporter] hset ts failed: %v", err)
		return
	}
	if err := rc.HsetCtx(ctx, membersKey, "ffmpeg_count", fmt.Sprintf("%d", r.svcCtx.FFmpegManager.Count())); err != nil {
		logx.Errorf("[node-reporter] hset relays failed: %v", err)
		return
	}
	if err := rc.ExpireCtx(ctx, membersKey, nodeExpireSec); err != nil {
		logx.Errorf("[node-reporter] expire failed: %v", err)
		return
	}
	if _, err := rc.SaddCtx(ctx, nodeSetKey, r.nodeID); err != nil {
		logx.Errorf("[node-reporter] sadd failed: %v", err)
	}
}
