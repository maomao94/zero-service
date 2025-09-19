package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// Screenshotter 视频截图工具
type Screenshotter struct {
	inputPath string // 视频源路径（本地文件或网络流）
}

// NewScreenshotter 创建截图工具实例
func NewScreenshotter(inputPath string) *Screenshotter {
	return &Screenshotter{
		inputPath: inputPath,
	}
}

// CaptureFrameAtTime 从指定时间点截取视频帧（返回JPEG字节流）
// ctx: 上下文（用于日志追踪）
// timePoint: 截图时间点（单位：秒）
func (s *Screenshotter) CaptureFrameAtTime(ctx context.Context, timePoint float64) ([]byte, error) {
	logx.WithContext(ctx).Infof("开始截取视频帧, 视频路径: %s, 时间点: %.2fs", s.inputPath, timePoint)

	// 缓冲区用于接收截图数据
	outputBuf := bytes.NewBuffer(nil)
	// 缓冲区用于捕获ffmpeg的错误输出
	stderrBuf := bytes.NewBuffer(nil)

	// 构建ffmpeg命令
	err := ffmpeg.Input(s.inputPath, ffmpeg.KwArgs{"ss": timePoint}).
		Output("pipe:", ffmpeg.KwArgs{
			"vframes": 1, // 仅截取1帧
			"format":  "image2",
			"vcodec":  "mjpeg", // 输出JPEG格式
		}).
		WithOutput(outputBuf, stderrBuf). // 输出到缓冲区，错误信息到stderrBuf
		Run()

	// 记录ffmpeg的原始输出（无论成功与否，便于调试）
	if stderrBuf.Len() > 0 {
		logx.WithContext(ctx).Infof("ffmpeg输出: %s", stderrBuf.String())
	}

	// 处理命令执行错误
	if err != nil {
		logx.WithContext(ctx).Errorf("截图失败, 视频路径: %s, 时间点: %.2fs, 错误: %v, ffmpeg输出: %s",
			s.inputPath, timePoint, err, stderrBuf.String())
		return nil, fmt.Errorf("ffmpeg执行失败: %w", err)
	}

	logx.WithContext(ctx).Infof("截图成功, 视频路径: %s, 时间点: %.2fs, 数据大小: %d字节",
		s.inputPath, timePoint, outputBuf.Len())

	return outputBuf.Bytes(), nil
}

// SaveFrameToFile 截取视频帧并保存到文件
// ctx: 上下文
// timePoint: 截图时间点（秒）
// outputPath: 输出文件路径（如./output.jpg）
func (s *Screenshotter) SaveFrameToFile(ctx context.Context, timePoint float64, outputPath string) error {
	// 先获取截图字节流
	imgData, err := s.CaptureFrameAtTime(ctx, timePoint)
	if err != nil {
		return fmt.Errorf("获取截图数据失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, imgData, 0644); err != nil {
		logx.WithContext(ctx).Errorf("保存截图到文件失败, 路径: %s, 错误: %v", outputPath, err)
		return fmt.Errorf("文件写入失败: %w", err)
	}

	logx.WithContext(ctx).Infof("截图已保存到文件, 路径: %s", outputPath)
	return nil
}

// CaptureFrameByIndex 按帧索引截取视频帧（返回JPEG字节流）
// frameIndex: 帧索引（第n帧，从0开始）
func (s *Screenshotter) CaptureFrameByIndex(ctx context.Context, frameIndex int) ([]byte, error) {
	logx.WithContext(ctx).Infof("开始截取视频帧, 视频路径: %s, 帧索引: %d", s.inputPath, frameIndex)

	outputBuf := bytes.NewBuffer(nil)
	stderrBuf := bytes.NewBuffer(nil)

	// 通过filter按帧索引筛选
	err := ffmpeg.Input(s.inputPath).
		Filter("select", ffmpeg.Args{fmt.Sprintf("eq(n,%d)", frameIndex)}). // 筛选第n帧
		Output("pipe:", ffmpeg.KwArgs{
			"vframes": 1,
			"format":  "image2",
			"vcodec":  "mjpeg",
		}).
		WithOutput(outputBuf, stderrBuf).
		Run()

	if stderrBuf.Len() > 0 {
		logx.WithContext(ctx).Infof("ffmpeg输出: %s", stderrBuf.String())
	}

	if err != nil {
		logx.WithContext(ctx).Errorf("按帧索引截图失败, 视频路径: %s, 帧索引: %d, 错误: %v, ffmpeg输出: %s",
			s.inputPath, frameIndex, err, stderrBuf.String())
		return nil, fmt.Errorf("ffmpeg执行失败: %w", err)
	}

	logx.WithContext(ctx).Infof("按帧索引截图成功, 视频路径: %s, 帧索引: %d, 数据大小: %d字节",
		s.inputPath, frameIndex, outputBuf.Len())

	return outputBuf.Bytes(), nil
}

// ProbeVideoInfo 获取视频基本信息（时长、分辨率等）
func (s *Screenshotter) ProbeVideoInfo(ctx context.Context) (map[string]interface{}, error) {
	logx.WithContext(ctx).Infof("开始探测视频信息, 路径: %s", s.inputPath)

	info, err := ffmpeg.Probe(s.inputPath)
	if err != nil {
		logx.WithContext(ctx).Errorf("视频信息探测失败, 路径: %s, 错误: %v", s.inputPath, err)
		return nil, fmt.Errorf("probe失败: %w", err)
	}

	// 解析probe结果（实际使用时可根据需要反序列化为结构体）
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(info), &result); err != nil {
		logx.WithContext(ctx).Errorf("解析视频信息失败, 路径: %s, 错误: %v", s.inputPath, err)
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	logx.WithContext(ctx).Infof("视频信息探测成功, 路径: %s", s.inputPath)
	return result, nil
}
