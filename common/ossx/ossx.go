package ossx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/carbonx"
)

var (
	Category_Minio   int64 = 1
	Category_Qiniu   int64 = 2
	Category_Ali     int64 = 3
	Category_Tencent int64 = 4
	Category_S3      int64 = 5

	templatePool = make(map[templateCacheKey]templateCacheItem)
	poolLock     sync.RWMutex
)

type OssTemplate interface {
	MakeBucket(ctx context.Context, tenantId, bucketName string) error                                                                                    // 创建存储桶
	RemoveBucket(ctx context.Context, tenantId, bucketName string) error                                                                                  // 删除存储桶
	StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error)                                                                // 获取文件信息
	BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error)                                                                          // 存储桶是否存在
	PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error)                      // 上传文件（HTTP multipart）
	PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) // 上传文件（通用 Reader）
	SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error)                                            // 生成文件url
	RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error                                                                          // 删除文件
	RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error)                                  // 批量删除文件
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
	return prefix + "/" + carbonx.NowDateCompact() + "/" +
		strings.ReplaceAll(u.String(), "-", "") +
		path.Ext(originalFilename)
}

// GenOssFilename 生成 OSS 对象路径 {prefix}/YYYYMMDD/{uuid}{ext}。
// 供需要预先确定对象名（如缩略图）再上传的调用方使用，等价于 OssRule.filename 的单前缀形式。
func GenOssFilename(originalFilename, pathPrefix string) string {
	rule := OssRule{}
	return rule.filename(originalFilename, pathPrefix)
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
	UpdateTime time.Time // 配置行更新时间，用于判断客户端缓存是否需要刷新
}

// GetConfigFn 按租户与业务编码加载 OSS 配置；ctx 用于取消与超时（如数据库查询）。
type GetConfigFn func(ctx context.Context, tenantId, code string) (config *Config, err error)

// templateCacheKey 缓存键：同一租户可配置多个 oss_code（不同后端/桶），客户端必须按 (tenantId, code) 唯一。
type templateCacheKey struct {
	tenantId string
	code     string
}

// templateCacheItem 缓存项：SDK 客户端模板 + 构建时所用配置的 update_time 快照。
type templateCacheItem struct {
	template   OssTemplate
	updateTime time.Time
}

// Template 按 (tenantId, code) 返回缓存的 OssTemplate（本质是对应云厂商的 SDK 客户端）。
// 每次调用读取配置（按唯一索引的单行查询），并以其 update_time 判断缓存是否需要刷新：
// update_time 发生变化（配置被编辑，含绕过服务直接改库）即重建客户端，否则复用缓存。
// 缓存不含桶名等业务配置：bucketName 由调用方逐次传入，缺省时各实现回落到 config.BucketName。
// 显式失效仍可通过 CacheInvalidate 触发。
func Template(ctx context.Context, TenantId, Code string, tenantMode bool, getConfig GetConfigFn) (ossTemplate OssTemplate, err error) {
	config, err := getConfig(ctx, TenantId, Code)
	if err != nil {
		return nil, err
	}
	key := templateCacheKey{tenantId: TenantId, code: Code}

	poolLock.RLock()
	item, ok := templatePool[key]
	poolLock.RUnlock()
	if ok && config.UpdateTime.Equal(item.updateTime) {
		return item.template, nil
	}

	poolLock.Lock()
	defer poolLock.Unlock()
	// 双重检查：并发刷新时只重建一次
	if item, ok = templatePool[key]; ok && config.UpdateTime.Equal(item.updateTime) {
		return item.template, nil
	}
	ossTemplate, err = NewTemplate(config, OssRule{tenantMode: tenantMode})
	if err != nil {
		return nil, err
	}
	templatePool[key] = templateCacheItem{template: ossTemplate, updateTime: config.UpdateTime}
	return ossTemplate, nil
}

// NewTemplate 根据 Config.Category 创建对应的 OssTemplate 实例。
func NewTemplate(config *Config, ossRule OssRule) (OssTemplate, error) {
	if config == nil {
		return nil, errors.New("oss config is nil")
	}
	switch config.Category {
	case Category_Minio:
		return NewMinioTemplate(config, ossRule)
	case Category_Qiniu:
		return NewQiniuTemplate(config, ossRule)
	case Category_Ali:
		return NewAliTemplate(config, ossRule)
	case Category_Tencent:
		return NewTencentTemplate(config, ossRule)
	case Category_S3:
		return NewS3Template(config, ossRule)
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

// CacheInvalidate 清除指定租户下所有 oss_code 的 OSS 客户端缓存。
// 在新增/更新/删除 OSS 配置后调用；同一租户多 oss_code 时全部失效，保证下次加载最新配置。
func CacheInvalidate(tenantId string) {
	poolLock.Lock()
	defer poolLock.Unlock()
	for k := range templatePool {
		if k.tenantId == tenantId {
			delete(templatePool, k)
		}
	}
}
