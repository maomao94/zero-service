package ossx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"time"

	"zero-service/common/filex"
	"zero-service/common/tool"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

type MinioTemplate struct {
	client        *minio.Client // Minio客户端
	autoAddress   bool          // true: 链接按 minio-go 寻址规则生成（S3 场景）；false: 固定路径风格（MinIO）
	ossProperties OssProperties // 配置参数
	ossRule       OssRule
}

func (m MinioTemplate) MakeBucket(ctx context.Context, tenantId, bucketName string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	return m.client.MakeBucket(ctx, m.ossRule.fullBucketName(tenantId, bucketName), minio.MakeBucketOptions{})
}

func (m MinioTemplate) RemoveBucket(ctx context.Context, tenantId, bucketName string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	return m.client.RemoveBucket(ctx, m.ossRule.fullBucketName(tenantId, bucketName))
}

func (m MinioTemplate) StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	object, err := m.client.StatObject(ctx, m.ossRule.fullBucketName(tenantId, bucketName), filename, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	} else {
		return &OssFile{
			Link:        m.fileLink(tenantId, bucketName, object.Key),
			Name:        object.Key,
			Size:        object.Size,
			FormatSize:  tool.DecimalBytes(object.Size),
			PutTime:     object.LastModified,
			ContentType: object.ContentType,
		}, nil
	}
}

func (m MinioTemplate) BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error) {
	if err := validateClient(m.client); err != nil {
		return false, err
	}
	return m.client.BucketExists(ctx, m.ossRule.fullBucketName(tenantId, bucketName))
}

func (m MinioTemplate) PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	f, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	filename := m.ossRule.filename(fileHeader.Filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	size, md5Hex, err := m.putObjectWithMD5(ctx, tenantId, bucketName, filename, fileHeader.Header.Get("content-type"), f, fileHeader.Size)
	if err != nil {
		return nil, err
	}
	return m.buildFile(tenantId, bucketName, filename, fileHeader.Filename, size, md5Hex), nil
}

func (m MinioTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	written, md5Hex, err := m.putObjectWithMD5(ctx, tenantId, bucketName, objectName, contentType, reader, size)
	if err != nil {
		return nil, err
	}
	return m.buildFile(tenantId, bucketName, objectName, filename, written, md5Hex), nil
}

func (m MinioTemplate) SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	if err := validateClient(m.client); err != nil {
		return "", err
	}
	reqParams := url.Values{}
	// 添加文件版本号，用于 MinIO 版本控制场景
	reqParams.Set("version", "1.0.0")
	url, err := m.client.PresignedGetObject(ctx, m.ossRule.fullBucketName(tenantId, bucketName), filename, expires, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (m MinioTemplate) RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	return m.client.RemoveObject(ctx, m.ossRule.fullBucketName(tenantId, bucketName), filename, minio.RemoveObjectOptions{})
}

func (m MinioTemplate) RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, f := range filenames {
			objectsCh <- minio.ObjectInfo{Key: f}
		}
	}()
	errorCh := m.client.RemoveObjects(ctx, m.ossRule.fullBucketName(tenantId, bucketName), objectsCh, minio.RemoveObjectsOptions{})

	// 收集失败的文件
	errMap := make(map[string]error)
	for e := range errorCh {
		if e.Err != nil {
			errMap[e.ObjectName] = e.Err
		}
	}

	// 按输入顺序组装结果
	results := make([]RemoveFileResult, len(filenames))
	for i, f := range filenames {
		results[i] = RemoveFileResult{Filename: f, Err: errMap[f]}
	}
	return results, nil
}

// buildFile 统一构建 File 返回结构（消除 PutFile/PutObject 中的重复代码）
func (m MinioTemplate) buildFile(tenantId, bucketName, objectName, originalName string, size int64, md5Hex string) *File {
	return &File{
		Link:         m.fileLink(tenantId, bucketName, objectName),
		Domain:       m.getOssHost(tenantId, bucketName),
		Name:         objectName,
		Size:         size,
		FormatSize:   tool.DecimalBytes(size),
		OriginalName: originalName,
		Md5:          md5Hex,
	}
}

// streamUploadPartSize 是未知大小流式上传（size < 0）时使用的分片大小。
// 未知大小会走 minio-go 的 no-length multipart 分支：不指定 PartSize 时它按 5TiB 估算并分配
// 约 528MiB 缓冲；这里固定为 16MiB（minio-go minPartSize），单次上传内存 16MiB、对象上限 160GiB，
// 且与已知大小路径的分片基准一致。
const streamUploadPartSize = 16 << 20

// minioPutOptions 构建 PutObject 选项。未知大小（size < 0）显式指定 PartSize 以限制内存占用；
// 已知大小（含 0）交给 minio-go 按实际大小决策，避免对超大数据（>PartSize*10000）造成分片数超限。
func minioPutOptions(contentType string, size int64) minio.PutObjectOptions {
	opts := minio.PutObjectOptions{ContentType: contentType}
	if size < 0 {
		opts.PartSize = streamUploadPartSize
	}
	return opts
}

// putObjectWithMD5 在上传对象时同步计算内容 MD5，返回实际写入大小与内容 MD5。
func (m MinioTemplate) putObjectWithMD5(ctx context.Context, tenantId, bucketName, objectName, contentType string, reader io.Reader, size int64) (int64, string, error) {
	var (
		info minio.UploadInfo
		err  error
	)
	md5Hex, err := filex.UploadWithMD5(reader, func(md5Reader io.Reader) error {
		info, err = m.client.PutObject(ctx, m.ossRule.fullBucketName(tenantId, bucketName),
			objectName, md5Reader, size, minioPutOptions(contentType, size))
		return err
	})
	if err != nil {
		return 0, "", err
	}
	return info.Size, md5Hex, nil
}

func (m MinioTemplate) getOssHost(tenantId, bucketName string) string {
	return m.objectHost(m.ossRule.fullBucketName(tenantId, bucketName))
}

// objectHost 生成对象访问域名。autoAddress 为 true 时（S3 场景）与 minio-go 的 BucketLookupAuto
// 判定保持一致：仅当虚拟主机寻址受支持（AWS/GCS/阿里云 OSS）时用 {bucket}.{host}，否则路径风格。
func (m MinioTemplate) objectHost(bucket string) string {
	endpoint := m.ossProperties.Endpoint
	if m.autoAddress {
		if u, err := url.Parse(endpoint); err == nil && s3utils.IsVirtualHostSupported(*u, bucket) {
			return u.Scheme + "://" + bucket + "." + u.Host
		}
	}
	return endpoint + "/" + bucket
}

func (m MinioTemplate) fileLink(tenantId, bucketName, filename string) string {
	return m.getOssHost(tenantId, bucketName) + "/" + filename
}

func NewMinioTemplate(config *Config, ossRule OssRule) (*MinioTemplate, error) {
	endpoint, secure := resolveMinioEndpoint(config.Endpoint, false)
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client (endpoint=%s): %w", config.Endpoint, err)
	}
	properties := ossPropertiesFromConfig(config)
	// 链接使用带协议的地址，避免调用方配置省略 scheme 时生成相对 URL。
	properties.Endpoint = schemeOf(secure) + "://" + endpoint
	return &MinioTemplate{
		client:        minioClient,
		ossProperties: properties,
		ossRule:       ossRule,
	}, nil
}

func validateClient(client *minio.Client) error {
	if client == nil {
		return errors.New("client is nil")
	}
	return nil
}
