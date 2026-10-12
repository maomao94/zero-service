package ossx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zero-service/common/filex"
	"zero-service/common/tool"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// TencentTemplate 腾讯云 COS 实现。Region 指定存储区域（如 ap-guangzhou），
// 访问域名为 https://{bucket}-{appId}.cos.{region}.myqcloud.com；
// 未配置 Region 时回退到 Endpoint 并拼接 bucket。
type TencentTemplate struct {
	ossProperties OssProperties
	ossRule       OssRule
}

var _ OssTemplate = (*TencentTemplate)(nil)

func (t TencentTemplate) MakeBucket(ctx context.Context, tenantId, bucketName string) error {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return err
	}
	_, err = client.Bucket.Put(ctx, nil)
	return err
}

func (t TencentTemplate) RemoveBucket(ctx context.Context, tenantId, bucketName string) error {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return err
	}
	_, err = client.Bucket.Delete(ctx)
	return err
}

func (t TencentTemplate) BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error) {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return false, err
	}
	if _, err = client.Bucket.Head(ctx); err != nil {
		if cos.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t TencentTemplate) StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error) {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return nil, err
	}
	resp, err := client.Object.Head(ctx, filename, nil)
	if err != nil {
		return nil, err
	}
	var putTime time.Time
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		putTime, _ = http.ParseTime(lastModified)
	}
	return &OssFile{
		Link:        t.fileLink(tenantId, bucketName, filename),
		Name:        filename,
		Size:        resp.ContentLength,
		FormatSize:  tool.DecimalBytes(resp.ContentLength),
		PutTime:     putTime,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (t TencentTemplate) PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error) {
	f, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	objectName := t.ossRule.filename(fileHeader.Filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = t.ossProperties.BucketName
	}
	size, md5Hex, err := t.putObjectWithMD5(ctx, tenantId, bucketName, objectName, fileHeader.Header.Get("content-type"), f, fileHeader.Size)
	if err != nil {
		return nil, err
	}
	return t.buildFile(tenantId, bucketName, objectName, fileHeader.Filename, size, md5Hex), nil
}

func (t TencentTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) {
	objectName := t.ossRule.filename(filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = t.ossProperties.BucketName
	}
	written, md5Hex, err := t.putObjectWithMD5(ctx, tenantId, bucketName, objectName, contentType, reader, size)
	if err != nil {
		return nil, err
	}
	return t.buildFile(tenantId, bucketName, objectName, filename, written, md5Hex), nil
}

func (t TencentTemplate) SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return "", err
	}
	signed, err := client.Object.GetPresignedURL(ctx, http.MethodGet, filename,
		t.ossProperties.AccessKey, t.ossProperties.SecretKey, expires, nil)
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

func (t TencentTemplate) RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return err
	}
	_, err = client.Object.Delete(ctx, filename)
	return err
}

func (t TencentTemplate) RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error) {
	results := make([]RemoveFileResult, len(filenames))
	for i, f := range filenames {
		results[i] = RemoveFileResult{Filename: f, Err: t.RemoveFile(ctx, tenantId, bucketName, f)}
	}
	return results, nil
}

// putObjectWithMD5 上传对象并同步计算内容 MD5，返回实际大小与 MD5（未知大小按实际读取字节数回填）。
func (t TencentTemplate) putObjectWithMD5(ctx context.Context, tenantId, bucketName, objectName, contentType string, reader io.Reader, size int64) (int64, string, error) {
	client, err := t.client(tenantId, bucketName)
	if err != nil {
		return 0, "", err
	}
	headerOptions := &cos.ObjectPutHeaderOptions{}
	if contentType != "" {
		headerOptions.ContentType = contentType
	}
	if size >= 0 {
		headerOptions.ContentLength = size
	}
	md5Hex, written, err := filex.UploadWithMD5Size(reader, func(r io.Reader) error {
		_, err := client.Object.Put(ctx, objectName, r, &cos.ObjectPutOptions{ObjectPutHeaderOptions: headerOptions})
		return err
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
func (t TencentTemplate) buildFile(tenantId, bucketName, objectName, originalName string, size int64, md5Hex string) *File {
	return &File{
		Link:         t.fileLink(tenantId, bucketName, objectName),
		Domain:       t.getOssHost(tenantId, bucketName),
		Name:         objectName,
		Size:         size,
		FormatSize:   tool.DecimalBytes(size),
		OriginalName: originalName,
		Md5:          md5Hex,
	}
}

func (t TencentTemplate) bucket(tenantId, bucketName string) string {
	if len(bucketName) == 0 {
		bucketName = t.ossProperties.BucketName
	}
	return t.ossRule.fullBucketName(tenantId, bucketName)
}

// withAppId 为腾讯云 bucket 追加应用 ID 后缀（bucket 形如 name-1250000000）。
func (t TencentTemplate) withAppId(bucket string) string {
	appId := t.ossProperties.AppId
	if appId == "" || strings.HasSuffix(bucket, "-"+appId) {
		return bucket
	}
	return bucket + "-" + appId
}

func (t TencentTemplate) getOssHost(tenantId, bucketName string) string {
	bucket := t.withAppId(t.bucket(tenantId, bucketName))
	if t.ossProperties.Region != "" {
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, t.ossProperties.Region)
	}
	if t.ossProperties.Endpoint != "" {
		return endpointWithBucket(t.ossProperties.Endpoint, bucket)
	}
	return ""
}

func (t TencentTemplate) fileLink(tenantId, bucketName, filename string) string {
	return t.getOssHost(tenantId, bucketName) + "/" + filename
}

func (t TencentTemplate) client(tenantId, bucketName string) (*cos.Client, error) {
	host := t.getOssHost(tenantId, bucketName)
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("tencent cos: invalid endpoint %q", host)
	}
	return cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  t.ossProperties.AccessKey,
			SecretKey: t.ossProperties.SecretKey,
		},
	}), nil
}

func NewTencentTemplate(config *Config, ossRule OssRule) (*TencentTemplate, error) {
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, errors.New("tencent cos: secret id and secret key are required")
	}
	if config.Region == "" && config.Endpoint == "" {
		return nil, errors.New("tencent cos: region or endpoint is required")
	}
	return &TencentTemplate{
		ossProperties: ossPropertiesFromConfig(config),
		ossRule:       ossRule,
	}, nil
}
