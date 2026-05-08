package filex

import (
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type HeadCaptureWriter struct {
	limit int
	buf   []byte
}

func NewHeadCaptureWriter(limit int) *HeadCaptureWriter {
	return &HeadCaptureWriter{limit: limit}
}

func (w *HeadCaptureWriter) Write(p []byte) (int, error) {
	if w.limit > 0 && len(w.buf) < w.limit {
		remain := w.limit - len(w.buf)
		if len(p) > remain {
			w.buf = append(w.buf, p[:remain]...)
		} else {
			w.buf = append(w.buf, p...)
		}
	}
	return len(p), nil
}

func (w *HeadCaptureWriter) Bytes() []byte {
	if w == nil || len(w.buf) == 0 {
		return nil
	}
	b := make([]byte, len(w.buf))
	copy(b, w.buf)
	return b
}

type CaptureOptions struct {
	TempDir     string
	TempPattern string
	NeedTemp    bool
	MaxHeadRead int
}

type Capture struct {
	file   *os.File
	path   string
	head   *HeadCaptureWriter
	closed bool
}

func NewCapture(options CaptureOptions) (*Capture, error) {
	capture := &Capture{
		head: NewHeadCaptureWriter(options.MaxHeadRead),
	}
	if options.NeedTemp {
		pattern := options.TempPattern
		if pattern == "" {
			pattern = "upload-*"
		}
		if err := os.MkdirAll(options.TempDir, os.ModePerm); err != nil {
			return nil, err
		}
		f, err := os.CreateTemp(options.TempDir, pattern)
		if err != nil {
			return nil, err
		}
		capture.file = f
		capture.path = f.Name()
	}
	return capture, nil
}

func (c *Capture) Writers() []io.Writer {
	writers := []io.Writer{c.head}
	if c.file != nil && !c.closed {
		writers = append([]io.Writer{c.file}, writers...)
	}
	return writers
}

func (c *Capture) Head() []byte {
	return c.head.Bytes()
}

func (c *Capture) HasTempFile() bool {
	return c != nil && c.path != ""
}

func (c *Capture) TempFilePath() string {
	if c == nil {
		return ""
	}
	return c.path
}

func (c *Capture) Close() error {
	if c == nil || c.file == nil || c.closed {
		return nil
	}
	err := c.file.Close()
	c.closed = true
	c.file = nil
	return err
}

func (c *Capture) Release() error {
	if c == nil || c.path == "" {
		return nil
	}
	if c.file != nil && !c.closed {
		_ = c.file.Close()
		c.closed = true
		c.file = nil
	}
	err := os.Remove(c.path)
	if os.IsNotExist(err) {
		err = nil
	}
	c.path = ""
	return err
}

func CopyFile(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func ReadFileHead(path string, maxHead int) ([]byte, int64, error) {
	if maxHead < 0 {
		maxHead = 0
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if maxHead == 0 {
		return nil, info.Size(), nil
	}
	head := make([]byte, maxHead)
	n, err := f.Read(head)
	if err != nil && err != io.EOF {
		return nil, 0, err
	}
	return head[:n], info.Size(), nil
}

// IsImageContentType 判断 Content-Type 是否为图片类型。
// 先通过 mime.ParseMediaType 去除参数（如 "; charset=utf-8"），再以前缀 "image/" 做匹配。
func IsImageContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		contentType = mediaType
	}
	return strings.HasPrefix(strings.ToLower(contentType), "image/")
}

// RemoveTempFile 根据 keepTempFiles 标记条件性清理临时文件。
// 复用 Capture.Release() 错误处理模式，返回 error 供调用方决定是否忽略。
func RemoveTempFile(tempPath string, keepTempFiles bool) error {
	if tempPath == "" || keepTempFiles {
		return nil
	}
	return os.Remove(tempPath)
}

// ExtractFilenameFromURL 从 URL 或本地文件路径提取文件名。
// 优先解析 URL 路径的 base，回退到 filepath.Base。
func ExtractFilenameFromURL(source string) string {
	if source == "" {
		return ""
	}
	if u, err := url.Parse(source); err == nil && u.Path != "" {
		if name := path.Base(u.Path); name != "." && name != "/" {
			return name
		}
	}
	return filepath.Base(strings.TrimRight(source, string(os.PathSeparator)))
}
