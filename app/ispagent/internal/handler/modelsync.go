package handler

import (
	"context"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

var modelSyncCommandName = map[int32]string{
	1:  "区域主机及边缘节点装置模型",
	2:  "机器人模型",
	3:  "摄像机模型及硬盘录像机模型",
	4:  "点位模型",
	5:  "无人机模型及无人机机巢模型",
	6:  "声纹模型",
	7:  "任务文件",
	8:  "检修区域配置文件",
	9:  "地图文件",
	10: "维护记录文件",
	11: "联动配置文件",
	12: "告警阈值模型",
}

var modelSyncFilePathKey = map[int32]string{
	2: "robot_file_path",
	4: "device_file_path",
	7: "task_file_path",
	9: "map_file_path",
}

func HandleModelSync(ctx context.Context, msg *isp.Message) ([]isp.Item, error) {
	name := modelSyncCommandName[msg.Command]
	logx.WithContext(ctx).Infof("[ispagent] 模型同步拉取 code=%s command=%d(%s)",
		msg.Code, msg.Command, name)

	fileKey := modelSyncFilePathKey[msg.Command]
	if fileKey == "" {
		fileKey = "file_path"
	}

	path := msg.Code + modelSyncDefaultPath[msg.Command]
	logx.WithContext(ctx).Infof("[ispagent] 模型同步 %s=%s", fileKey, path)
	return []isp.Item{{fileKey: path}}, nil
}

var modelSyncDefaultPath = map[int32]string{
	2: "/robot_model.xml",
	4: "/device_model.xml",
	9: "/map_model.xml",
}
