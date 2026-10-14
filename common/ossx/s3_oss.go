package ossx

import (
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Template 通用 S3 协议对象存储实现，基于 minio-go（其本身就是 S3 兼容客户端）。
// 适用于 AWS S3、Cloudflare R2、Ceph RGW、SeaweedFS、Garage 等任意 S3 兼容存储：
// 只需在 oss 表配置 endpoint（可带协议）、region、accessKey/secretKey，即可直接使用，无需新增代码。
//
// 复用 MinioTemplate 的全部操作，仅在客户端构建时按 S3 场景适配：
//   - Secure 由 endpoint scheme 推导，省略协议时默认 https；
//   - 支持显式 Region；
//   - 链接与客户端寻址风格一致（BucketLookupAuto），AWS/GCS/阿里云 OSS 走虚拟主机，其余走路径风格。
type S3Template struct {
	MinioTemplate
}

var _ OssTemplate = (*S3Template)(nil)

func NewS3Template(config *Config, ossRule OssRule) (*S3Template, error) {
	if config.Endpoint == "" {
		return nil, errors.New("s3: endpoint is required")
	}
	endpoint, secure := resolveMinioEndpoint(config.Endpoint, true)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure:       secure,
		Region:       config.Region,
		BucketLookup: minio.BucketLookupAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client (endpoint=%s): %w", config.Endpoint, err)
	}
	properties := ossPropertiesFromConfig(config)
	// 链接使用带协议的地址，避免调用方配置省略 scheme 时生成相对 URL。
	properties.Endpoint = schemeOf(secure) + "://" + endpoint
	return &S3Template{MinioTemplate{
		client:        client,
		autoAddress:   true,
		ossProperties: properties,
		ossRule:       ossRule,
	}}, nil
}
