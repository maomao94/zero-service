package docker

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

// RunComposeUp 执行 docker compose up -d。
// compose 操作仍使用 exec.Command（Docker SDK 不支持 compose 子命令）。
func RunComposeUp(composeFile, serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "up", "-d", serviceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("compose up 失败: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// RunComposeDown 执行 docker compose down。
func RunComposeDown(composeFile, serviceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "down", serviceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("compose down 失败: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// RunDockerExec 在容器中执行命令。
func RunDockerExec(containerID string, cmdArgs ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := append([]string{"exec", containerID}, cmdArgs...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("docker exec 失败: %w\n%s", err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// RunDockerCommand 执行任意 docker 命令，返回合并输出。
func RunDockerCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("docker 命令超时 (Docker daemon 是否正在运行?)")
		}
		return out.String(), fmt.Errorf("docker %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// RunDockerCpToHost 将容器内文件拷贝到主机目录。
func RunDockerCpToHost(container, containerPath, hostPath string) (string, error) {
	return RunDockerCommand("cp", container+":"+containerPath+"/.", hostPath+"/")
}

// PathType 检测路径类型：folder / zip / invalid。
func PathType(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "invalid"
	}
	if info.IsDir() {
		return "folder"
	}
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		return "zip"
	}
	return "unknown"
}

// UnzipToDir 用 Go 标准库解压 zip 到目标目录。
func UnzipToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 zip 失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// CopyToContainer 将本地目录打包为 tar 并通过 SDK 复制到容器内。
func (c *Client) CopyToContainer(containerID, dstPath, srcDir string) error {
	tarReader, err := packDirToTar(srcDir)
	if err != nil {
		return fmt.Errorf("打包失败: %w", err)
	}
	defer tarReader.Close()

	ctx, cancel := c.withTimeout()
	defer cancel()

	return c.cli.CopyToContainer(ctx, containerID, dstPath, tarReader, container.CopyToContainerOptions{})
}

// packDirToTar 将目录打包为 tar 流（仅包含目录内文件，不含顶层目录名）。
func packDirToTar(srcDir string) (*io.PipeReader, error) {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		defer tw.Close()

		basePath := srcDir
		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(basePath, path)
			if relPath == "." {
				return nil
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		})
	}()
	return pr, nil
}
