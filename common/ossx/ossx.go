package ossx

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	Category_Minio   int64 = 1
	Category_Qiniu   int64 = 2
	Category_Ali     int64 = 3
	Category_Tencent int64 = 4

	templatePool = make(map[string]OssTemplate)
	ossPool      = make(map[string]*Config)
	poolLock     sync.RWMutex
)

type OssTemplate interface {
	MakeBucket(ctx context.Context, tenantId, bucketName string) error                                                                                          // 创建存储桶
	RemoveBucket(ctx context.Context, tenantId, bucketName string) error                                                                                        // 删除存储桶
	StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error)                                                                      // 获取文件信息
	BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error)                                                                                // 存储桶是否存在
	PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error)                            // 上传文件（HTTP multipart）
	PutStream(ctx context.Context, tenantId, bucketName, filename, contentType string, stream io.Reader, streamSize int64) (*File, error)                       // 上传文件（字节流）
	PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, objectSize int64, pathPrefix ...string) (*File, error) // 上传文件（通用 Reader）
	SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error)                                                  // 生成文件url
	RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error                                                                                // 删除文件
	RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error)                                        // 批量删除文件
}

var _ OssTemplate = (*MinioTemplate)(nil)

type OssRule struct {
	tenantMode bool
}

func (o *OssRule) fullBucketName(tenantId, bucketName string) string {
	prefix := ""
	if o.tenantMode {
		prefix = tenantId + "-"
	}
	return prefix + bucketName
}

func (o *OssRule) filename(originalFilename string, pathPrefix ...string) string {
	if len(pathPrefix) > 1 && pathPrefix[1] != "" { // 第二个参数是固定 filename
		return pathPrefix[1]
	}

	u, _ := uuid.NewUUID()
	prefix := "upload"
	if len(pathPrefix) > 0 && pathPrefix[0] != "" {
		prefix = pathPrefix[0] // 使用调用者传入的路径
	}
	return prefix + "/" + time.Now().Format("20060102") + "/" +
		strings.ReplaceAll(u.String(), "-", "") +
		path.Ext(originalFilename)
}

type File struct {
	Link         string // 文件地址
	Domain       string // 域名地址
	Name         string // 文件名
	Size         int64  // 文件大小
	FormatSize   string // 格式化文件大小
	OriginalName string // 初始文件名
	AttachId     string // 附件表ID
	Md5          string // 文件内容 MD5
}

type RemoveFileResult struct {
	Filename string
	Err      error
}

type OssFile struct {
	Link        string    // 文件地址
	Name        string    // 文件名
	Size        int64     // 文件大小
	FormatSize  string    // 格式化文件大小
	PutTime     time.Time // 文件上传时间
	ContentType string    // 文件contentType
}

type OssProperties struct {
	Enabled    bool           // 是否启用
	TenantMode bool           // 是否开启租户模式
	Endpoint   string         // 对象存储服务的URL
	AppId      string         // 应用ID TencentCOS需要
	Region     string         // 区域简称 TencentCOS需要
	AccessKey  string         // Access key就像用户ID，可以唯一标识你的账户
	SecretKey  string         // Secret key是你账户的密码
	BucketName string         // 默认的存储桶名称
	Args       map[string]any // 自定义属性
}

type Config struct {
	Category   int64
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	AppId      string
	Region     string
}

// GetConfigFn 按租户与业务编码加载 OSS 配置；ctx 用于取消与超时（如数据库查询）。
type GetConfigFn func(ctx context.Context, tenantId, code string) (config *Config, err error)

func Template(ctx context.Context, TenantId, Code string, tenantMode bool, getConfig GetConfigFn) (ossTemplate OssTemplate, err error) {
	config, err := getConfig(ctx, TenantId, Code)
	if err != nil {
		return nil, err
	}

	poolLock.RLock()
	configCached := ossPool[TenantId]
	ossTemplate = templatePool[TenantId]
	poolLock.RUnlock()

	if needRebuild(configCached, ossTemplate, config) {
		poolLock.Lock()
		defer poolLock.Unlock()
		configCached = ossPool[TenantId]
		ossTemplate = templatePool[TenantId]
		if needRebuild(configCached, ossTemplate, config) {
			ossRule := OssRule{tenantMode: tenantMode}
			ossTemplate, err = NewTemplate(config, ossRule)
			if err != nil {
				return nil, err
			}
			templatePool[TenantId] = ossTemplate
			ossPool[TenantId] = config
		}
	}
	return
}

// NewTemplate 根据 Config.Category 创建对应的 OssTemplate 实例。
func NewTemplate(config *Config, ossRule OssRule) (OssTemplate, error) {
	switch config.Category {
	case Category_Minio:
		return NewMinioTemplate(config, ossRule)
	default:
		return nil, fmt.Errorf("unsupported oss category: %d", config.Category)
	}
}

// MustNewTemplate 调用 NewTemplate，失败时 panic（go-zero Must* 风格）。
func MustNewTemplate(config *Config, ossRule OssRule) OssTemplate {
	t, err := NewTemplate(config, ossRule)
	logx.Must(err)
	return t
}

func needRebuild(cached *Config, ossTemplate OssTemplate, current *Config) bool {
	if cached == nil || current == nil || ossTemplate == nil {
		return true
	}
	return current.Endpoint != cached.Endpoint ||
		current.AccessKey != cached.AccessKey ||
		current.SecretKey != cached.SecretKey
}

// CacheInvalidate 清除指定租户的 OSS 缓存，在更新/删除 OSS 配置后调用
func CacheInvalidate(tenantId string) {
	poolLock.Lock()
	defer poolLock.Unlock()
	delete(templatePool, tenantId)
	delete(ossPool, tenantId)
}
