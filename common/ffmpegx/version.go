package ffmpegx

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const versionTimeout = 3 * time.Second

// PrintVersions 打印 ffmpeg/ffprobe 版本，便于现场核对外部命令环境。
// 获取失败只记录日志，不影响服务启动（本地开发环境可能未安装 ffmpeg）。
func PrintVersions() {
	printCommandVersion("ffmpeg", "FFmpeg")
	printCommandVersion("ffprobe", "FFprobe")
}

func printCommandVersion(bin, label string) {
	version, err := binaryVersion(bin)
	if err != nil {
		logx.Errorf("[ffmpegx] 获取 %s 版本失败: %v", bin, err)
		return
	}
	fmt.Printf("%s Version: %s\n", label, version)
}

func binaryVersion(bin string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, bin, "-version").Output()
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(string(out), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("empty version output")
	}
	return line, nil
}
