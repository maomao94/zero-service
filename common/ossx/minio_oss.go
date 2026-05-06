package ossx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"time"
	"zero-service/common/tool"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioTemplate struct {
	client        *minio.Client // Minio客户端
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
	info, md5Hex, err := m.putObjectWithMD5(ctx, tenantId, bucketName, filename, fileHeader.Header.Get("content-type"), f, fileHeader.Size)
	if err != nil {
		return nil, err
	}
	return m.buildFile(tenantId, bucketName, filename, fileHeader.Filename, info, md5Hex), nil
}

func (m MinioTemplate) PutStream(ctx context.Context, tenantId, bucketName, filename, contentType string, stream io.Reader, streamSize int64) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	info, md5Hex, err := m.putObjectWithMD5(ctx, tenantId, bucketName, objectName, contentType, stream, streamSize)
	if err != nil {
		return nil, err
	}
	return m.buildFile(tenantId, bucketName, objectName, filename, info, md5Hex), nil
}

func (m MinioTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, objectSize int64, pathPrefix ...string) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	info, md5Hex, err := m.putObjectWithMD5(ctx, tenantId, bucketName, objectName, contentType, reader, objectSize)
	if err != nil {
		return nil, err
	}
	return m.buildFile(tenantId, bucketName, objectName, filename, info, md5Hex), nil
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

// buildFile 统一构建 File 返回结构（消除 PutStream/PutObject 中的重复代码）
func (m MinioTemplate) buildFile(tenantId, bucketName, objectName, originalName string, info minio.UploadInfo, md5Hex string) *File {
	return &File{
		Link:         m.fileLink(tenantId, bucketName, objectName),
		Domain:       m.getOssHost(tenantId, bucketName),
		Name:         objectName,
		Size:         info.Size,
		FormatSize:   tool.DecimalBytes(info.Size),
		OriginalName: originalName,
		Md5:          md5Hex,
	}
}

// putObjectWithMD5 在上传对象时同步计算内容 MD5，避免重复读取流。
func (m MinioTemplate) putObjectWithMD5(ctx context.Context, tenantId, bucketName, objectName, contentType string, reader io.Reader, objectSize int64) (minio.UploadInfo, string, error) {
	var (
		info minio.UploadInfo
		err  error
	)
	md5Hex, err := UploadWithMD5(reader, func(md5Reader io.Reader) error {
		info, err = m.client.PutObject(ctx, m.ossRule.fullBucketName(tenantId, bucketName),
			objectName, md5Reader, objectSize, minio.PutObjectOptions{
				ContentType: contentType,
			})
		return err
	})
	if err != nil {
		return minio.UploadInfo{}, "", err
	}
	return info, md5Hex, nil
}

func (m MinioTemplate) getOssHost(tenantId, bucketName string) string {
	return m.ossProperties.Endpoint + "/" + m.ossRule.fullBucketName(tenantId, bucketName)
}

func (m MinioTemplate) fileLink(tenantId, bucketName, filename string) string {
	return m.ossProperties.Endpoint + "/" + m.ossRule.fullBucketName(tenantId, bucketName) + "/" + filename
}

func NewMinioTemplate(config *Config, ossRule OssRule) (*MinioTemplate, error) {
	ossProperties := OssProperties{
		Endpoint:   config.Endpoint,
		AccessKey:  config.AccessKey,
		SecretKey:  config.SecretKey,
		BucketName: config.BucketName,
		AppId:      config.AppId,
		Region:     config.Region,
		Args:       nil,
	}
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client (endpoint=%s): %w", config.Endpoint, err)
	}
	return &MinioTemplate{
		client:        minioClient,
		ossProperties: ossProperties,
		ossRule:       ossRule,
	}, nil
}

func validateClient(client *minio.Client) error {
	if client == nil {
		return errors.New("client is nil")
	}
	return nil
}
