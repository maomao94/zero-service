package handler

import (
	"context"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

var modelTypeName = map[string]string{
	"1":  "区域主机及边缘节点装置模型",
	"2":  "机器人模型",
	"3":  "摄像机模型及硬盘录像机模型",
	"4":  "点位模型",
	"5":  "无人机模型及无人机机巢模型",
	"6":  "声纹模型",
	"7":  "任务文件",
	"8":  "检修区域配置文件",
	"9":  "地图文件",
	"10": "维护记录文件",
	"11": "联动配置文件",
	"12": "告警阈值模型",
}

func HandleModelUpdateReport(ctx context.Context, msg *isp.Message) error {
	logx.WithContext(ctx).Infof("[ispagent] 模型更新上报 code=%s items=%d", msg.Code, len(msg.Items))
	for i, item := range msg.Items {
		typ := item["type"]
		fp := item["file_path"]
		logx.WithContext(ctx).Infof("[ispagent] 模型[%d] type=%s(%s) file_path=%s",
			i, typ, modelTypeName[typ], fp)
	}
	return nil
}
