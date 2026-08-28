package ffmpegx

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// RelayProgressInterval ffmpeg progress 输出间隔（-stats_period 单位秒）
const RelayProgressInterval = 5

// BuildRelayCmd 构建 relay copy 推流命令：
//
//	ffmpeg -progress pipe:1 -stats_period 5 -timeout 10000000 -i <source> -c copy -f flv <target>
//
// 返回 cmd 未 Start。stderr 由 Manager 通过 StderrPipe 消费（WithStderrHandler）；
// 无 handler 时 Manager 不碰 stderr，ffmpeg-go 默认丢弃 stderr 输出。
func BuildRelayCmd(ctx context.Context, source, target string) *exec.Cmd {
	in := ffmpeg.Input(source, ffmpeg.KwArgs{
		"rw_timeout": "10000000", // 10s read/write timeout (microseconds)
	})
	out := ffmpeg.OutputContext(ctx, []*ffmpeg.Stream{in}, target, ffmpeg.KwArgs{
		"c": "copy",
		"f": "flv",
	})
	cmd := out.Silent(true).Compile()
	cmd.Args = append([]string{
		cmd.Args[0],
		"-progress", "pipe:1",
		"-stats_period", fmt.Sprintf("%d", RelayProgressInterval),
	}, cmd.Args[1:]...)
	logx.WithContext(ctx).Infof("[ffmpegx] compiled command: ffmpeg %s", strings.Join(cmd.Args[1:], " "))
	return cmd
}
