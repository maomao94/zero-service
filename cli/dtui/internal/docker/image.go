package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// Image 代表一个镜像记录。
type Image struct {
	Repository string
	Tag        string
	ID         string
	Created    string
	Size       string
}

// Ref 返回镜像的可引用名称，格式为 repository:tag。
func (i Image) Ref() string {
	if i.Tag == "" || i.Tag == "<none>" {
		return i.Repository
	}
	return fmt.Sprintf("%s:%s", i.Repository, i.Tag)
}

// DefaultSaveFile 根据镜像信息生成默认的 tar 文件名。
func (i Image) DefaultSaveFile() string {
	name := i.Repository
	if name == "" || name == "<none>" {
		name = i.ID
	}
	name = filepath.Base(strings.ReplaceAll(name, ":", "-"))
	tag := strings.ReplaceAll(i.Tag, "/", "-")
	if tag == "" || tag == "<none>" {
		return name + ".tar"
	}
	return fmt.Sprintf("%s-%s.tar", name, tag)
}

// ListImages 通过 SDK 的 ImageList 获取镜像列表。
func (c *Client) ListImages(filter string) ([]Image, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	images, err := c.cli.ImageList(ctx, image.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("查询镜像列表失败: %w", err)
	}

	var result []Image
	for _, img := range images {
		repo := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			parts := strings.SplitN(img.RepoTags[0], ":", 2)
			repo = parts[0]
			if len(parts) > 1 {
				tag = parts[1]
			}
		}

		id := img.ID
		if len(id) > 12 {
			id = id[7:19] // 去掉 "sha256:" 前缀取 12 位
		}

		createdTime := time.Unix(img.Created, 0).Format("2006-01-02 15:04")
		sizeStr := formatSize(img.Size)

		im := Image{
			Repository: repo,
			Tag:        tag,
			ID:         id,
			Created:    createdTime,
			Size:       sizeStr,
		}

		if filter == "" ||
			strings.Contains(strings.ToLower(im.Repository), strings.ToLower(filter)) ||
			strings.Contains(strings.ToLower(im.Tag), strings.ToLower(filter)) {
			result = append(result, im)
		}
	}
	return result, nil
}

// SaveImage 保存镜像为 tar 文件。
func (c *Client) SaveImage(ref string, outputPath string) error {
	ctx, cancel := c.withLongTimeout()
	defer cancel()

	reader, err := c.cli.ImageSave(ctx, []string{ref})
	if err != nil {
		return fmt.Errorf("保存镜像失败: %w", err)
	}
	defer reader.Close()

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err
}

// RemoveImage 删除镜像。
func (c *Client) RemoveImage(ref string, force bool) error {
	ctx, cancel := c.withTimeout()
	defer cancel()

	_, err := c.cli.ImageRemove(ctx, ref, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

// TagImage 给镜像打标签。
func (c *Client) TagImage(source, target string) error {
	ctx, cancel := c.withTimeout()
	defer cancel()
	return c.cli.ImageTag(ctx, source, target)
}

// PruneImages 清理悬空镜像。
func (c *Client) PruneImages() (uint64, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	report, err := c.cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// ---- 镜像层分析 ----

// ImageHistoryEntry 镜像层历史记录。
type ImageHistoryEntry struct {
	ID        string
	Created   int64
	CreatedBy string
	Size      int64
	Comment   string
}

// ImageHistory 获取镜像层历史。
func (c *Client) ImageHistory(ref string) ([]ImageHistoryEntry, error) {
	ctx, cancel := c.withTimeout()
	defer cancel()

	history, err := c.cli.ImageHistory(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("获取镜像历史失败: %w", err)
	}

	var entries []ImageHistoryEntry
	for _, h := range history {
		id := h.ID
		if len(id) > 12 {
			id = id[:12]
		}
		entries = append(entries, ImageHistoryEntry{
			ID:        id,
			Created:   h.Created,
			CreatedBy: h.CreatedBy,
			Size:      h.Size,
			Comment:   h.Comment,
		})
	}
	return entries, nil
}

// formatSize 格式化字节大小为人类可读格式。
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
