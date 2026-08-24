package relay

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// buildFfmpegCmd 构建 ffmpeg 命令（等价形式）：
//
//	ffmpeg -i <source> -c copy -f flv <target>
//
// 通过可取消 ctx 绑定进程生命周期：Compile() 内部使用 exec.CommandContext，
// cancel() 即 kill 进程（停止转推语义）。
// stderr 通过 WithErrorOutput 捕获，进程退出时打印。
// Silent(true) 关闭 ffmpeg-go 包内对标准库 log 的"compiled command"打印，
// 改由本函数用 logx 记录，保证日志走 go-zero 体系。
func buildFfmpegCmd(ctx context.Context, source, target string, stderr *bytes.Buffer) *exec.Cmd {
	in := ffmpeg.Input(source)
	out := ffmpeg.OutputContext(ctx, []*ffmpeg.Stream{in}, target, ffmpeg.KwArgs{
		"c": "copy",
		"f": "flv",
	})
	out = out.WithErrorOutput(stderr)
	cmd := out.Silent(true).Compile()
	logx.WithContext(ctx).Infof("compiled command: ffmpeg %s", strings.Join(cmd.Args[1:], " "))
	return cmd
}

// watch 后台监听进程退出：异常退出打印 error 日志（含 task_id/源地址/stderr）并清理任务
func (m *Manager) watch(task *Task) {
	go func() {
		defer m.wg.Done()
		err := task.Cmd.Wait()
		m.tasks.Delete(task.TaskID)
		// 由 Stop 主动 cancel → 正常停止，不视为异常
		if task.Ctx.Err() != nil {
			logx.Infof("stream relay task stopped: task_id=%s source=%s target=%s",
				task.TaskID, task.Source, task.Target)
			return
		}
		if err == nil {
			logx.Infof("stream relay task ended: task_id=%s source=%s target=%s",
				task.TaskID, task.Source, task.Target)
			return
		}
		// 异常退出（流断开、拉流失败等）：打印 error 日志
		logx.Errorf("stream relay task exited with error: task_id=%s source=%s target=%s err=%v stderr=%s",
			task.TaskID, task.Source, task.Target, err, task.Stderr.String())
	}()
}
