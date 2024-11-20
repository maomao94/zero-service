package ossx

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/minio/minio-go"
	"io"
	"mime/multipart"
	"net/url"
	"time"
	"zero-service/model"
)

type MinioTemplate struct {
	client        *minio.Client // Minio客户端
	ossProperties OssProperties // 配置参数
	ossRule       OssRule
}

func (m MinioTemplate) MakeBucket(tenantId, bucketName string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	return m.client.MakeBucket(m.ossRule.bucketName(tenantId, bucketName), "")
}

func (m MinioTemplate) RemoveBucket(tenantId, bucketName string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	return m.client.RemoveBucket(m.ossRule.bucketName(tenantId, bucketName))
}

func (m MinioTemplate) StatFile(tenantId, bucketName, filename string) (*OssFile, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	object, err := m.client.StatObject(m.ossRule.bucketName(tenantId, bucketName), filename, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	} else {
		return &OssFile{
			Link:        m.fileLink(tenantId, bucketName, object.Key),
			Name:        object.Key,
			Length:      object.Size,
			PutTime:     object.LastModified,
			ContentType: object.ContentType,
		}, nil
	}
}

func (m MinioTemplate) BucketExists(tenantId, bucketName string) (bool, error) {
	if err := validateClient(m.client); err != nil {
		return false, err
	}
	return m.client.BucketExists(m.ossRule.bucketName(tenantId, bucketName))
}

func (m MinioTemplate) PutFile(tenantId, bucketName string, fileHeader *multipart.FileHeader) (*File, error) {
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
	_, err = m.client.PutObject(m.ossRule.bucketName(tenantId, bucketName),
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
			OriginalName: fileHeader.Filename,
		}, nil
	}
}

func (m MinioTemplate) PutStream(tenantId, bucketName, filename, contentType string, stream *[]byte) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	reader := bytes.NewReader(*stream)
	buffer := bufio.NewReader(reader)
	_, err := m.client.PutObject(m.ossRule.bucketName(tenantId, bucketName),
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
			OriginalName: filename,
		}, nil
	}
}

func (m MinioTemplate) PutObject(tenantId, bucketName, filename, contentType string, reader io.Reader, objectSize int64) (*File, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	objectName := m.ossRule.filename(filename)
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	_, err := m.client.PutObject(m.ossRule.bucketName(tenantId, bucketName),
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
			OriginalName: filename,
		}, nil
	}
}

func (m MinioTemplate) GetObject(tenantId, bucketName, filename string) (*minio.Object, error) {
	if err := validateClient(m.client); err != nil {
		return nil, err
	}
	return m.client.GetObject(m.ossRule.bucketName(tenantId, bucketName), filename, minio.GetObjectOptions{})
}

func (m MinioTemplate) SignUrl(tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	if err := validateClient(m.client); err != nil {
		return "", err
	}
	// 创建一个 URL 查询参数对象
	reqParams := url.Values{}
	reqParams.Set("version", "1.0") // 添加文件版本
	url, err := m.client.PresignedGetObject(m.ossRule.bucketName(tenantId, bucketName), filename, expires, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (m MinioTemplate) RemoveFile(tenantId, bucketName, filename string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	return m.client.RemoveObject(m.ossRule.bucketName(tenantId, bucketName), filename)
}

func (m MinioTemplate) RemoveFiles(tenantId string, bucketName string, filenames []string) error {
	if err := validateClient(m.client); err != nil {
		return err
	}
	if len(bucketName) == 0 {
		bucketName = m.ossProperties.BucketName
	}
	objectsCh := make(chan string)
	go func() {
		defer close(objectsCh)
		for _, f := range filenames {
			objectsCh <- f
		}
	}()
	errorCh := m.client.RemoveObjects(m.ossRule.bucketName(tenantId, bucketName), objectsCh)
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
	minioClient, _ := minio.New(Oss.Endpoint, Oss.AccessKey, Oss.SecretKey, false)
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
