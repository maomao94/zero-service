package ossx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"mime/multipart"
	"net/url"
	"time"
	"zero-service/common/tool"
	"zero-service/model"
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
	return m.client.MakeBucket(ctx, m.ossRule.bucketName(tenantId, bucketName), minio.MakeBucketOptions{})
}

func (m MinioTemplate) RemoveBucket(ctx context.Context, tenantId, bucketName string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	return m.client.RemoveBucket(ctx, m.ossRule.bucketName(tenantId, bucketName))
}

func (m MinioTemplate) StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	object, err := m.client.StatObject(ctx, m.ossRule.bucketName(tenantId, bucketName), filename, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	} else {
		return &OssFile{
			Link:        m.fileLink(tenantId, bucketName, object.Key),
			Name:        object.Key,
			Size:        object.Size,
			PutTime:     object.LastModified,
			ContentType: object.ContentType,
		}, nil
	}
}

func (m MinioTemplate) BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error) {
	if err := validateClient(m.client); err != nil {
		return false, err
	}
	return m.client.BucketExists(ctx, m.ossRule.bucketName(tenantId, bucketName))
}

func (m MinioTemplate) PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	f, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	filename := m.ossRule.filename(fileHeader.Filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	info, err := m.client.PutObject(ctx, m.ossRule.bucketName(tenantId, bucketName),
		filename, f, fileHeader.Size, minio.PutObjectOptions{
			ContentType: fileHeader.Header.Get("content-type"),
		})
	if err != nil {
		return nil, err
	} else {
		return &File{
			Link:         m.fileLink(tenantId, bucketName, filename),
			Domain:       m.getOssHost(tenantId, bucketName),
			Name:         filename,
			Size:         info.Size,
			FormatSize:   tool.DecimalBytes(info.Size),
			OriginalName: fileHeader.Filename,
		}, nil
	}
}

func (m MinioTemplate) PutStream(ctx context.Context, tenantId, bucketName, filename, contentType string, stream *[]byte) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	reader := bytes.NewReader(*stream)
	buffer := bufio.NewReader(reader)
	info, err := m.client.PutObject(ctx, m.ossRule.bucketName(tenantId, bucketName),
		objectName, buffer, reader.Size(), minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return nil, err
	} else {
		return &File{
			Link:         m.fileLink(tenantId, bucketName, objectName),
			Domain:       m.getOssHost(tenantId, bucketName),
			Name:         objectName,
			Size:         info.Size,
			FormatSize:   tool.DecimalBytes(info.Size),
			OriginalName: filename,
		}, nil
	}
}

func (m MinioTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, objectSize int64) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	info, err := m.client.PutObject(ctx, m.ossRule.bucketName(tenantId, bucketName),
		objectName, reader, objectSize, minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return nil, err
	} else {
		return &File{
			Link:         m.fileLink(tenantId, bucketName, objectName),
			Domain:       m.getOssHost(tenantId, bucketName),
			Name:         objectName,
			Size:         info.Size,
			FormatSize:   tool.DecimalBytes(info.Size),
			OriginalName: filename,
		}, nil
	}
}

func (m MinioTemplate) SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	if err := validateClient(m.client); err != nil {
		return "", err
	}
	// 创建一个 URL 查询参数对象
	reqParams := url.Values{}
	reqParams.Set("version", "1.0.0") // 添加文件版本
	url, err := m.client.PresignedGetObject(ctx, m.ossRule.bucketName(tenantId, bucketName), filename, expires, reqParams)
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
	return m.client.RemoveObject(ctx, m.ossRule.bucketName(tenantId, bucketName), filename, minio.RemoveObjectOptions{})
}

func (m MinioTemplate) RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for _, f := range filenames {
			// 构造 ObjectInfo 对象
			objectInfo := minio.ObjectInfo{
				Key: f,
			}
			objectsCh <- objectInfo
		}
	}()
	errorCh := m.client.RemoveObjects(ctx, m.ossRule.bucketName(tenantId, bucketName), objectsCh, minio.RemoveObjectsOptions{})
	select {
	case err := <-errorCh:
		return err.Err
	}
	return nil
}

func (m MinioTemplate) getOssHost(tenantId, bucketName string) string {
	return m.ossProperties.Endpoint + "/" + m.ossRule.bucketName(tenantId, bucketName)
}

func (m MinioTemplate) fileLink(tenantId, bucketName, filename string) string {
	return m.ossProperties.Endpoint + "/" + m.ossRule.bucketName(tenantId, bucketName) + "/" + filename
}

func NewMinioTemplate(Oss *model.Oss, ossRule OssRule) *MinioTemplate {
	ossProperties := OssProperties{
		Endpoint:   Oss.Endpoint,
		AccessKey:  Oss.AccessKey,
		SecretKey:  Oss.SecretKey,
		BucketName: Oss.BucketName,
		Args:       nil,
	}
	// 初使化 minio client对象。
	minioClient, _ := minio.New(Oss.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(Oss.AccessKey, Oss.SecretKey, ""),
		Secure: false,
	})
	return &MinioTemplate{
		client:        minioClient,
		ossProperties: ossProperties,
		ossRule:       ossRule,
	}
}

func validateClient(client *minio.Client) error {
	if client == nil {
		return errors.New("client is nil")
	}
	return nil
}
