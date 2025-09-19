package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// Screenshotter 视频截图工具（仅处理本地文件写入，不涉及OSS）
type Screenshotter struct {
	inputPath string // 视频源（本地文件路径或实时流地址）
}

// NewScreenshotter 创建截图工具实例
func NewScreenshotter(inputPath string) (*Screenshotter, error) {
	if inputPath == "" {
		return nil, errors.New("视频源地址不能为空")
	}
	return &Screenshotter{
		inputPath: inputPath,
	}, nil
}

// CaptureFrameToFile 按时间点截图并写入本地文件
// ctx: 上下文（用于日志追踪）
// timePoint: 截图时间点（秒；实时流传-1表示取当前帧）
// localFilePath: 目标文件路径（如 ./snapshots/20240920/123.jpg）
// 返回值: 成功写入的文件路径 / 错误信息
func (s *Screenshotter) CaptureFrameToFile(ctx context.Context, timePoint float64, localFilePath string) (string, error) {
	startTime := time.Now()
	logx.WithContext(ctx).Infof(
		"开始按时间点截图, 视频源: %s, 时间点: %.2fs, 目标路径: %s",
		s.inputPath, timePoint, localFilePath,
	)

	// 确保目标目录存在
	if err := ensureDir(localFilePath); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 捕获ffmpeg错误输出
	stderrBuf := &bytes.Buffer{}

	// 构建ffmpeg命令：直接输出到本地文件
	err := ffmpeg.Input(s.inputPath, ffmpeg.KwArgs{"ss": timePoint}).
		Output(localFilePath, ffmpeg.KwArgs{
			"vframes": 1, // 仅截取1帧
			"format":  "mjpeg",
			"vcodec":  "mjpeg", // 输出JPEG格式
			"q:v":     2,       // 图片质量（1-31，1最优）
		}).
		WithErrorOutput(stderrBuf). // 捕获错误日志
		OverWriteOutput().          // 覆盖已有文件
		Run()

	// 记录ffmpeg原始输出（调试用）
	if stderrBuf.Len() > 0 {
		logx.WithContext(ctx).Debugf("ffmpeg输出: %s", stderrBuf.String())
	}

	// 处理ffmpeg执行错误
	if err != nil {
		cleanupFile(localFilePath) // 清理无效文件
		return "", fmt.Errorf("ffmpeg执行失败: %w, 输出: %s", err, stderrBuf.String())
	}

	// 验证文件是否有效
	if err := validateFile(localFilePath); err != nil {
		cleanupFile(localFilePath)
		return "", fmt.Errorf("文件验证失败: %w", err)
	}

	// 成功日志
	logx.WithContext(ctx).Infof(
		"时间点截图成功, 路径: %s, 大小: %d字节, 耗时: %v",
		localFilePath, getFileSize(localFilePath), time.Since(startTime),
	)
	return localFilePath, nil
}

// CaptureFrameByIndexToFile 按帧索引截图并写入本地文件
// frameIndex: 帧索引（从0开始）
// localFilePath: 目标文件路径
func (s *Screenshotter) CaptureFrameByIndexToFile(ctx context.Context, frameIndex int, localFilePath string) (string, error) {
	startTime := time.Now()
	logx.WithContext(ctx).Infof(
		"开始按帧索引截图, 视频源: %s, 帧索引: %d, 目标路径: %s",
		s.inputPath, frameIndex, localFilePath,
	)

	// 确保目标目录存在
	if err := ensureDir(localFilePath); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 捕获ffmpeg错误输出
	stderrBuf := &bytes.Buffer{}

	// 构建ffmpeg命令：使用filter筛选指定帧索引
	err := ffmpeg.Input(s.inputPath).
		Filter("select", ffmpeg.Args{fmt.Sprintf("eq(n,%d)", frameIndex)}). // 筛选第n帧
		Output(localFilePath, ffmpeg.KwArgs{
			"vframes": 1,
			"format":  "mjpeg",
			"vcodec":  "mjpeg",
			"q:v":     2,
		}).
		WithErrorOutput(stderrBuf).
		OverWriteOutput().
		Run()

	// 记录ffmpeg输出
	if stderrBuf.Len() > 0 {
		logx.WithContext(ctx).Debugf("ffmpeg输出: %s", stderrBuf.String())
	}

	// 处理执行错误
	if err != nil {
		cleanupFile(localFilePath)
		return "", fmt.Errorf("ffmpeg执行失败: %w, 输出: %s", err, stderrBuf.String())
	}

	// 验证文件有效性
	if err := validateFile(localFilePath); err != nil {
		cleanupFile(localFilePath)
		return "", fmt.Errorf("文件验证失败: %w", err)
	}

	// 成功日志
	logx.WithContext(ctx).Infof(
		"帧索引截图成功, 路径: %s, 大小: %d字节, 耗时: %v",
		localFilePath, getFileSize(localFilePath), time.Since(startTime),
	)
	return localFilePath, nil
}

// GenerateTempFilePath 生成临时文件路径（避免手动指定路径时的冲突）
// baseDir: 基础目录（如 ./temp_snapshots）
// ext: 文件扩展名（如 .jpg）
func (s *Screenshotter) GenerateTempFilePath(baseDir, ext string) string {
	// 格式: baseDir/20060102/uuid.ext
	dateDir := time.Now().Format("20060102")
	fullDir := filepath.Join(baseDir, dateDir)
	_ = os.MkdirAll(fullDir, 0755) // 提前创建目录
	return filepath.Join(fullDir, fmt.Sprintf("%s%s", uuid.NewString(), ext))
}

// ---------------- 内部辅助函数 ----------------

// ensureDir 确保文件所在目录存在
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("MkdirAll failed: %w", err)
	}
	return nil
}

// validateFile 验证文件是否有效（存在且大小不为0）
func validateFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("os.Stat failed: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("file is empty")
	}
	return nil
}

// cleanupFile 清理无效文件
func cleanupFile(filePath string) {
	if err := os.Remove(filePath); err != nil {
		logx.Errorf("清理无效文件失败: %s, 错误: %v", filePath, err)
	}
}

// getFileSize 获取文件大小（字节）
func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}
