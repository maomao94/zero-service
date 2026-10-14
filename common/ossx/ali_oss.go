package ossx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"zero-service/common/filex"
	"zero-service/common/tool"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// AliTemplate 阿里云 OSS 实现。Endpoint 为区域访问地址（如 oss-cn-hangzhou.aliyuncs.com），
// 对象地址通过将 bucket 拼接到协议头之后得到。
type AliTemplate struct {
	client        *oss.Client
	ossProperties OssProperties
	ossRule       OssRule
}

var _ OssTemplate = (*AliTemplate)(nil)

func (a AliTemplate) MakeBucket(ctx context.Context, tenantId, bucketName string) error {
	return a.client.CreateBucket(a.bucket(tenantId, bucketName))
}

func (a AliTemplate) RemoveBucket(ctx context.Context, tenantId, bucketName string) error {
	return a.client.DeleteBucket(a.bucket(tenantId, bucketName))
}

func (a AliTemplate) StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error) {
	bucket, err := a.bucketClient(tenantId, bucketName)
	if err != nil {
		return nil, err
	}
	header, err := bucket.GetObjectDetailedMeta(filename)
	if err != nil {
		return nil, err
	}
	size, _ := strconv.ParseInt(header.Get(oss.HTTPHeaderContentLength), 10, 64)
	var putTime time.Time
	if lastModified := header.Get(oss.HTTPHeaderLastModified); lastModified != "" {
		putTime, _ = http.ParseTime(lastModified)
	}
	return &OssFile{
		Link:        a.fileLink(tenantId, bucketName, filename),
		Name:        filename,
		Size:        size,
		FormatSize:  tool.DecimalBytes(size),
		PutTime:     putTime,
		ContentType: header.Get(oss.HTTPHeaderContentType),
	}, nil
}

func (a AliTemplate) BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error) {
	return a.client.IsBucketExist(a.bucket(tenantId, bucketName))
}

func (a AliTemplate) PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error) {
	f, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	objectName := a.ossRule.filename(fileHeader.Filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = a.ossProperties.BucketName
	}
	size, md5Hex, err := a.putObjectWithMD5(tenantId, bucketName, objectName, fileHeader.Header.Get("content-type"), f, fileHeader.Size)
	if err != nil {
		return nil, err
	}
	return a.buildFile(tenantId, bucketName, objectName, fileHeader.Filename, size, md5Hex), nil
}

func (a AliTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) {
	objectName := a.ossRule.filename(filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = a.ossProperties.BucketName
	}
	written, md5Hex, err := a.putObjectWithMD5(tenantId, bucketName, objectName, contentType, reader, size)
	if err != nil {
		return nil, err
	}
	return a.buildFile(tenantId, bucketName, objectName, filename, written, md5Hex), nil
}

func (a AliTemplate) SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	bucket, err := a.bucketClient(tenantId, bucketName)
	if err != nil {
		return "", err
	}
	return bucket.SignURL(filename, oss.HTTPGet, int64(expires.Seconds()))
}

func (a AliTemplate) RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error {
	bucket, err := a.bucketClient(tenantId, bucketName)
	if err != nil {
		return err
	}
	return bucket.DeleteObject(filename)
}

func (a AliTemplate) RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error) {
	results := make([]RemoveFileResult, len(filenames))
	for i, f := range filenames {
		results[i] = RemoveFileResult{Filename: f, Err: a.RemoveFile(ctx, tenantId, bucketName, f)}
	}
	return results, nil
}

// putObjectWithMD5 上传对象并同步计算内容 MD5，返回实际大小与 MD5（未知大小按实际读取字节数回填）。
func (a AliTemplate) putObjectWithMD5(tenantId, bucketName, objectName, contentType string, reader io.Reader, size int64) (int64, string, error) {
	bucket, err := a.bucketClient(tenantId, bucketName)
	if err != nil {
		return 0, "", err
	}
	var options []oss.Option
	if contentType != "" {
		options = append(options, oss.ContentType(contentType))
	}
	md5Hex, written, err := filex.UploadWithMD5Size(reader, func(r io.Reader) error {
		return bucket.PutObject(objectName, r, options...)
	})
	if err != nil {
		return 0, "", err
	}
	if size < 0 {
		size = written
	}
	return size, md5Hex, nil
}

// buildFile 统一构建 File 返回结构（与 MinioTemplate.buildFile 对齐）。
func (a AliTemplate) buildFile(tenantId, bucketName, objectName, originalName string, size int64, md5Hex string) *File {
	return &File{
		Link:         a.fileLink(tenantId, bucketName, objectName),
		Domain:       a.getOssHost(tenantId, bucketName),
		Name:         objectName,
		Size:         size,
		FormatSize:   tool.DecimalBytes(size),
		OriginalName: originalName,
		Md5:          md5Hex,
	}
}

func (a AliTemplate) bucket(tenantId, bucketName string) string {
	if len(bucketName) == 0 {
		bucketName = a.ossProperties.BucketName
	}
	return a.ossRule.fullBucketName(tenantId, bucketName)
}

func (a AliTemplate) bucketClient(tenantId, bucketName string) (*oss.Bucket, error) {
	return a.client.Bucket(a.bucket(tenantId, bucketName))
}

func (a AliTemplate) getOssHost(tenantId, bucketName string) string {
	return endpointWithBucket(a.ossProperties.Endpoint, a.bucket(tenantId, bucketName))
}

func (a AliTemplate) fileLink(tenantId, bucketName, filename string) string {
	return a.getOssHost(tenantId, bucketName) + "/" + filename
}

func NewAliTemplate(config *Config, ossRule OssRule) (*AliTemplate, error) {
	if config.Endpoint == "" {
		return nil, errors.New("ali oss: endpoint is required")
	}
	client, err := oss.New(ensureScheme(config.Endpoint), config.AccessKey, config.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client (endpoint=%s): %w", config.Endpoint, err)
	}
	return &AliTemplate{
		client:        client,
		ossProperties: ossPropertiesFromConfig(config),
		ossRule:       ossRule,
	}, nil
}
