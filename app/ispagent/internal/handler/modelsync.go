package handler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"zero-service/common/ftps"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

var modelSyncFilePathKey = map[int32]string{
	1:  "host_file_path",
	2:  "robot_file_path",
	3:  "video_file_path",
	4:  "device_file_path",
	5:  "drone_file_path",
	6:  "voice_file_path",
	7:  "task_file_path",
	8:  "overhaularea_file_path",
	9:  "map_file_path",
	10: "maintain_file_path",
	11: "source_file_path",
	12: "alarm_file_path",
}

var modelSyncDefaultPath = map[int32]string{
	1:  "/host_model.xml",
	2:  "/robot_model.xml",
	3:  "/video_model.xml",
	4:  "/device_model.xml",
	5:  "/drone_model.xml",
	6:  "/voice_model.xml",
	7:  "/task_model.xml",
	8:  "/overhaularea_model.xml",
	9:  "/map_model.jpeg",
	10: "/maintain_model.xml",
	11: "/source_model.xml",
	12: "/alarm_model.xml",
}

// HandleModelSync 处理模型同步拉取指令 (61-1~12)：
//   - 61-1 区域主机及边缘节点装置模型 → host_file_path
//   - 61-2 机器人模型 → robot_file_path
//   - 61-3 摄像机模型及硬盘录像机模型 → video_file_path
//   - 61-4 点位模型 → device_file_path
//   - 61-5 无人机模型及无人机机巢模型 → drone_file_path
//   - 61-6 声纹模型 → voice_file_path
//   - 61-7 任务文件 → task_file_path
//   - 61-8 检修区域配置文件 → overhaularea_file_path
//   - 61-9 地图文件 → map_file_path
//   - 61-10 维护记录文件 → maintain_file_path
//   - 61-11 联动配置文件 → source_file_path
//   - 61-12 告警阈值模型 → alarm_file_path
//
// 处理流程：从 data provider 获取模型数据 → 流式生成 XML → FTPS 上传 → 返回路径
func HandleModelSync(ctx context.Context, msg *isp.Message, uploader *ftps.Uploader, provider ModelDataProvider) ([]isp.Item, error) {
	name := modelSyncCommandName[msg.Command]
	logx.WithContext(ctx).Infof("[ispagent] 模型同步拉取 code=%s command=%d(%s)", msg.Code, msg.Command, name)

	fileKey := modelSyncFilePathKey[msg.Command]
	if fileKey == "" {
		fileKey = "file_path"
	}

	remotePath, err := syncModel(ctx, msg, uploader, provider)
	if err != nil {
		return nil, fmt.Errorf("sync %s model: %w", name, err)
	}

	logx.WithContext(ctx).Infof("[ispagent] 模型同步 %s=%s", fileKey, remotePath)
	return []isp.Item{{fileKey: remotePath}}, nil
}

func syncModel(ctx context.Context, msg *isp.Message, uploader *ftps.Uploader, provider ModelDataProvider) (string, error) {
	remoteName := msg.Code + modelSyncDefaultPath[msg.Command]

	switch msg.Command {
	case isp.CommandModelRobot:
		return syncPatrolDeviceModel(ctx, uploader, provider, remoteName, msg.Code)
	case isp.CommandModelPoint:
		return syncDevicePointModel(ctx, uploader, provider, remoteName, msg.Code)
	case isp.CommandModelMap:
		return syncMapModel(ctx, uploader, remoteName, msg.Code)
	default:
		return path.Join(uploader.Config().RemoteDir, remoteName), nil
	}
}

func syncMapModel(ctx context.Context, uploader *ftps.Uploader, remoteName, stationCode string) (string, error) {
	localPath := filepath.Join("local", stationCode, "map.jpeg")
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("map file not found: %s", localPath)
	}
	result, err := uploader.UploadFile(ctx, localPath, remoteName)
	if err != nil {
		return "", fmt.Errorf("upload map model %s: %w", remoteName, err)
	}
	return result.RemotePath, nil
}

func syncPatrolDeviceModel(ctx context.Context, uploader *ftps.Uploader, provider ModelDataProvider, remoteName, stationCode string) (string, error) {
	items, err := provider.PatrolDevices(ctx, stationCode)
	if err != nil {
		return "", fmt.Errorf("get patrol devices: %w", err)
	}

	var buf bytes.Buffer
	if err := isp.WritePatrolDeviceModel(&buf, items); err != nil {
		return "", fmt.Errorf("generate patrol device xml: %w", err)
	}

	result, err := uploader.Upload(ctx, remoteName, &buf, int64(buf.Len()))
	if err != nil {
		return "", fmt.Errorf("upload patrol device model %s: %w", remoteName, err)
	}
	return result.RemotePath, nil
}

func syncDevicePointModel(ctx context.Context, uploader *ftps.Uploader, provider ModelDataProvider, remoteName, stationCode string) (string, error) {
	items, err := provider.DevicePoints(ctx, stationCode)
	if err != nil {
		return "", fmt.Errorf("get device points: %w", err)
	}

	var buf bytes.Buffer
	if err := isp.WriteDeviceModel(&buf, items); err != nil {
		return "", fmt.Errorf("generate device point xml: %w", err)
	}

	result, err := uploader.Upload(ctx, remoteName, &buf, int64(buf.Len()))
	if err != nil {
		return "", fmt.Errorf("upload device point model %s: %w", remoteName, err)
	}
	return result.RemotePath, nil
}
