package ossx

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"time"

	"zero-service/common/filex"
	"zero-service/common/tool"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
)

// QiniuTemplate 七牛云存储实现。Endpoint 为空间绑定的访问域名（CDN 域名），
// Region 用于空间创建（z0/z1/z2/na0/as0 等）。
type QiniuTemplate struct {
	uploader       *storage.FormUploader
	resumeUploader *storage.ResumeUploaderV2
	bucketManager  *storage.BucketManager
	mac            *auth.Credentials
	ossProperties  OssProperties
	ossRule        OssRule
}

var _ OssTemplate = (*QiniuTemplate)(nil)

func (q QiniuTemplate) MakeBucket(ctx context.Context, tenantId, bucketName string) error {
	return q.bucketManager.CreateBucket(q.bucket(tenantId, bucketName), storage.RegionID(q.ossProperties.Region))
}

func (q QiniuTemplate) RemoveBucket(ctx context.Context, tenantId, bucketName string) error {
	return q.bucketManager.DropBucket(q.bucket(tenantId, bucketName))
}

func (q QiniuTemplate) StatFile(ctx context.Context, tenantId, bucketName, filename string) (*OssFile, error) {
	info, err := q.bucketManager.Stat(q.bucket(tenantId, bucketName), filename)
	if err != nil {
		return nil, err
	}
	return &OssFile{
		Link:        q.fileLink(filename),
		Name:        filename,
		Size:        info.Fsize,
		FormatSize:  tool.DecimalBytes(info.Fsize),
		PutTime:     time.Unix(0, info.PutTime*100), // 七牛 PutTime 单位为 100 纳秒
		ContentType: info.MimeType,
	}, nil
}

func (q QiniuTemplate) BucketExists(ctx context.Context, tenantId, bucketName string) (bool, error) {
	buckets, err := q.bucketManager.Buckets(false)
	if err != nil {
		return false, err
	}
	target := q.bucket(tenantId, bucketName)
	for _, b := range buckets {
		if b == target {
			return true, nil
		}
	}
	return false, nil
}

func (q QiniuTemplate) PutFile(ctx context.Context, tenantId, bucketName string, fileHeader *multipart.FileHeader, pathPrefix ...string) (*File, error) {
	f, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	objectName := q.ossRule.filename(fileHeader.Filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = q.ossProperties.BucketName
	}
	size, md5Hex, err := q.putObjectWithMD5(ctx, tenantId, bucketName, objectName, fileHeader.Header.Get("content-type"), f, fileHeader.Size)
	if err != nil {
		return nil, err
	}
	return q.buildFile(tenantId, bucketName, objectName, fileHeader.Filename, size, md5Hex), nil
}

func (q QiniuTemplate) PutObject(ctx context.Context, tenantId, bucketName, filename, contentType string, reader io.Reader, size int64, pathPrefix ...string) (*File, error) {
	objectName := q.ossRule.filename(filename, pathPrefix...)
	if len(bucketName) == 0 {
		bucketName = q.ossProperties.BucketName
	}
	written, md5Hex, err := q.putObjectWithMD5(ctx, tenantId, bucketName, objectName, contentType, reader, size)
	if err != nil {
		return nil, err
	}
	return q.buildFile(tenantId, bucketName, objectName, filename, written, md5Hex), nil
}

func (q QiniuTemplate) SignUrl(ctx context.Context, tenantId, bucketName, filename string, expires time.Duration) (string, error) {
	deadline := time.Now().Add(expires).Unix()
	return storage.MakePrivateURL(q.mac, q.getOssHost(), filename, deadline), nil
}

func (q QiniuTemplate) RemoveFile(ctx context.Context, tenantId, bucketName, filename string) error {
	return q.bucketManager.Delete(q.bucket(tenantId, bucketName), filename)
}

func (q QiniuTemplate) RemoveFiles(ctx context.Context, tenantId string, bucketName string, filenames []string) ([]RemoveFileResult, error) {
	results := make([]RemoveFileResult, len(filenames))
	for i, f := range filenames {
		results[i] = RemoveFileResult{Filename: f, Err: q.RemoveFile(ctx, tenantId, bucketName, f)}
	}
	return results, nil
}

// putObjectWithMD5 上传对象并同步计算内容 MD5，返回实际大小与 MD5。
// 已知大小走表单上传；未知大小（size < 0）走可续传 v2 的分片流式上传，避免整文件读入内存。
func (q QiniuTemplate) putObjectWithMD5(ctx context.Context, tenantId, bucketName, objectName, contentType string, reader io.Reader, size int64) (int64, string, error) {
	if size < 0 {
		return q.putObjectUnknownSizeWithMD5(ctx, tenantId, bucketName, objectName, contentType, reader)
	}

	token := (&storage.PutPolicy{Scope: q.bucket(tenantId, bucketName)}).UploadToken(q.mac)
	var ret storage.PutRet
	md5Hex, err := filex.UploadWithMD5(reader, func(r io.Reader) error {
		return q.uploader.Put(ctx, &ret, token, objectName, r, size, &storage.PutExtra{MimeType: contentType})
	})
	if err != nil {
		return 0, "", err
	}
	return size, md5Hex, nil
}

// putObjectUnknownSizeWithMD5 以未知大小分片流式上传，用计数 Reader 回填实际写入大小，
// 同时经 UploadWithMD5 在流上同步计算内容 MD5。
func (q QiniuTemplate) putObjectUnknownSizeWithMD5(ctx context.Context, tenantId, bucketName, objectName, contentType string, reader io.Reader) (int64, string, error) {
	token := (&storage.PutPolicy{Scope: q.bucket(tenantId, bucketName)}).UploadToken(q.mac)
	counter := filex.NewCountingReader(reader)
	var ret storage.PutRet
	md5Hex, err := filex.UploadWithMD5(counter, func(r io.Reader) error {
		return q.resumeUploader.PutWithoutSize(ctx, &ret, token, objectName, r, &storage.RputV2Extra{MimeType: contentType})
	})
	if err != nil {
		return 0, "", err
	}
	return counter.N(), md5Hex, nil
}

// buildFile 统一构建 File 返回结构（与 MinioTemplate.buildFile 对齐）。
// 七牛访问地址由空间绑定域名直接拼接对象 key，不含 bucket 段，故租户与桶名不参与组装。
func (q QiniuTemplate) buildFile(_, _, objectName, originalName string, size int64, md5Hex string) *File {
	return &File{
		Link:         q.fileLink(objectName),
		Domain:       q.getOssHost(),
		Name:         objectName,
		Size:         size,
		FormatSize:   tool.DecimalBytes(size),
		OriginalName: originalName,
		Md5:          md5Hex,
	}
}

func (q QiniuTemplate) bucket(tenantId, bucketName string) string {
	if len(bucketName) == 0 {
		bucketName = q.ossProperties.BucketName
	}
	return q.ossRule.fullBucketName(tenantId, bucketName)
}

// getOssHost 七牛访问地址由空间绑定域名直接拼接对象 key，不含 bucket 段。
func (q QiniuTemplate) getOssHost() string {
	return ensureScheme(q.ossProperties.Endpoint)
}

func (q QiniuTemplate) fileLink(filename string) string {
	return q.getOssHost() + "/" + filename
}

func NewQiniuTemplate(config *Config, ossRule OssRule) (*QiniuTemplate, error) {
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, errors.New("qiniu: access key and secret key are required")
	}
	mac := auth.New(config.AccessKey, config.SecretKey)
	cfg := storage.NewConfig()
	cfg.UseHTTPS = true
	if config.Region != "" {
		if region, ok := storage.GetRegionByID(storage.RegionID(config.Region)); ok {
			cfg.Region = &region
		}
	}
	return &QiniuTemplate{
		uploader:       storage.NewFormUploader(cfg),
		resumeUploader: storage.NewResumeUploaderV2(cfg),
		bucketManager:  storage.NewBucketManager(mac, cfg),
		mac:            mac,
		ossProperties:  ossPropertiesFromConfig(config),
		ossRule:        ossRule,
	}, nil
}
