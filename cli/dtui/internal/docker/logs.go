package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// LogOptions 日志拉取选项。
type LogOptions struct {
	Tail       string
	Since      string
	Follow     bool
	Timestamps bool
}

// StreamLogs 流式获取容器日志（逐行推送）。
// 自动检测 TTY/多路复用模式：peek 首字节决定解码方式，避免 stdcopy 错误混入输出。
func (c *Client) StreamLogs(id string, opts LogOptions) (<-chan string, <-chan error) {
	logCh := make(chan string, 200)
	errCh := make(chan error, 1)

	go func() {
		defer close(logCh)
		defer close(errCh)

		ctx, cancel := context.WithCancel(c.ctx)
		defer cancel()

		reader, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       opts.Tail,
			Since:      opts.Since,
			Follow:     opts.Follow,
			Timestamps: opts.Timestamps,
		})
		if err != nil {
			errCh <- fmt.Errorf("获取日志失败: %w", err)
			return
		}
		defer reader.Close()

		br := bufio.NewReaderSize(reader, 4096)
		hdr, err := br.Peek(8)
		if err != nil || len(hdr) < 8 || (hdr[0] != 0 && hdr[0] != 1) {
			scanner := bufio.NewScanner(br)
			for scanner.Scan() {
				select {
				case logCh <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil {
				errCh <- err
			}
			return
		}

		pipeReader, pipeWriter := io.Pipe()
		parseDone := make(chan error, 1)
		go func() {
			_, copyErr := stdcopy.StdCopy(pipeWriter, pipeWriter, br)
			_ = pipeWriter.CloseWithError(copyErr)
			if copyErr != nil {
				parseDone <- copyErr
			}
			close(parseDone)
		}()

		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			select {
			case logCh <- scanner.Text():
			case <-ctx.Done():
				_ = pipeReader.Close()
				return
			}
		}

		scannerErr := scanner.Err()
		if scannerErr == nil {
			select {
			case pErr := <-parseDone:
				scannerErr = pErr
			default:
			}
		}
		if scannerErr != nil {
			errCh <- scannerErr
		}
	}()

	return logCh, errCh
}

// FetchLogs 批量获取容器日志。
// 优先使用 stdcopy 解码多路复用流，失败时回退到原始逐行读取（TTY 模式）。
func (c *Client) FetchLogs(id string, opts LogOptions) ([]string, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	reader, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Timestamps: opts.Timestamps,
	})
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %w", err)
	}
	defer reader.Close()

	// 先读出全部原始数据
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取日志失败: %w", err)
	}

	// 尝试 stdcopy 解码
	var decBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&decBuf, &decBuf, bytes.NewReader(raw)); err == nil {
		var lines []string
		scanner := bufio.NewScanner(&decBuf)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return lines, nil
	}

	// fallback: TTY 原始文本
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}
