package ossx

import (
	"strings"
)

// ossPropertiesFromConfig 将 Config 的公共字段映射为 OssProperties，供各后端实现复用。
func ossPropertiesFromConfig(config *Config) OssProperties {
	return OssProperties{
		Endpoint:   config.Endpoint,
		AccessKey:  config.AccessKey,
		SecretKey:  config.SecretKey,
		BucketName: config.BucketName,
		AppId:      config.AppId,
		Region:     config.Region,
	}
}

// ensureScheme 保证 endpoint 带有 http(s) 协议头，缺省补全为 https。
func ensureScheme(endpoint string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	return "https://" + endpoint
}

// resolveMinioEndpoint 解析 minio/S3 的 endpoint，返回 scheme-less 主机与 secure 标记。
// endpoint 显式带协议时以协议为准（必须与 minio-go 客户端的 Secure 选项一致）；
// 无协议时采用 defaultSecure（MinIO 本地多为 http，S3 云端多为 https）。
func resolveMinioEndpoint(endpoint string, defaultSecure bool) (host string, secure bool) {
	if rest, ok := strings.CutPrefix(endpoint, "https://"); ok {
		return strings.TrimSuffix(rest, "/"), true
	}
	if rest, ok := strings.CutPrefix(endpoint, "http://"); ok {
		return strings.TrimSuffix(rest, "/"), false
	}
	return strings.TrimSuffix(endpoint, "/"), defaultSecure
}

// schemeOf 返回 secure 对应的协议名。
func schemeOf(secure bool) string {
	if secure {
		return "https"
	}
	return "http"
}

// endpointWithBucket 在 endpoint 的协议头之后插入 bucket，形成带 bucket 的访问域名。
// 云厂商（阿里云 OSS、腾讯云 COS）的对象访问地址形如：
//
//	endpoint=https://oss-cn-hangzhou.aliyuncs.com, bucket=mybucket
//	→ https://mybucket.oss-cn-hangzhou.aliyuncs.com
func endpointWithBucket(endpoint, bucket string) string {
	if bucket == "" {
		return endpoint
	}
	endpoint = ensureScheme(endpoint)
	if rest, ok := strings.CutPrefix(endpoint, "https://"); ok {
		return "https://" + bucket + "." + rest
	}
	if rest, ok := strings.CutPrefix(endpoint, "http://"); ok {
		return "http://" + bucket + "." + rest
	}
	return "https://" + bucket + "." + endpoint
}
