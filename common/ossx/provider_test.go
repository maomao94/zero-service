package ossx

import "testing"

func TestEnsureScheme(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "oss-cn-hangzhou.aliyuncs.com", want: "https://oss-cn-hangzhou.aliyuncs.com"},
		{in: "http://cdn.example.com", want: "http://cdn.example.com"},
		{in: "https://cdn.example.com", want: "https://cdn.example.com"},
	}
	for _, tt := range tests {
		if got := ensureScheme(tt.in); got != tt.want {
			t.Errorf("ensureScheme(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEndpointWithBucket(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		bucket   string
		want     string
	}{
		{name: "https endpoint", endpoint: "https://oss-cn-hangzhou.aliyuncs.com", bucket: "demo", want: "https://demo.oss-cn-hangzhou.aliyuncs.com"},
		{name: "http endpoint", endpoint: "http://cos.ap-guangzhou.myqcloud.com", bucket: "demo", want: "http://demo.cos.ap-guangzhou.myqcloud.com"},
		{name: "no scheme", endpoint: "oss-cn-hangzhou.aliyuncs.com", bucket: "demo", want: "https://demo.oss-cn-hangzhou.aliyuncs.com"},
		{name: "empty bucket", endpoint: "https://oss-cn-hangzhou.aliyuncs.com", bucket: "", want: "https://oss-cn-hangzhou.aliyuncs.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endpointWithBucket(tt.endpoint, tt.bucket); got != tt.want {
				t.Errorf("endpointWithBucket(%q, %q) = %q, want %q", tt.endpoint, tt.bucket, got, tt.want)
			}
		})
	}
}

func TestNewTemplateDispatch(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:   "minio",
			config: &Config{Category: Category_Minio, Endpoint: "http://minio.example.com:9000", AccessKey: "ak", SecretKey: "sk", BucketName: "demo"},
		},
		{
			name:   "qiniu",
			config: &Config{Category: Category_Qiniu, Endpoint: "https://cdn.example.com", AccessKey: "ak", SecretKey: "sk", BucketName: "demo", Region: "z0"},
		},
		{
			name:   "ali",
			config: &Config{Category: Category_Ali, Endpoint: "oss-cn-hangzhou.aliyuncs.com", AccessKey: "ak", SecretKey: "sk", BucketName: "demo"},
		},
		{
			name:   "tencent",
			config: &Config{Category: Category_Tencent, AccessKey: "ak", SecretKey: "sk", BucketName: "demo", AppId: "1250000000", Region: "ap-guangzhou"},
		},
		{
			name:   "s3",
			config: &Config{Category: Category_S3, Endpoint: "https://s3.us-east-1.amazonaws.com", AccessKey: "ak", SecretKey: "sk", BucketName: "demo", Region: "us-east-1"},
		},
		{
			name:    "unsupported",
			config:  &Config{Category: 99},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplate(tt.config, OssRule{})
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewTemplate() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewTemplate() error = %v", err)
			}
			if got == nil {
				t.Fatal("NewTemplate() = nil, want template")
			}
		})
	}
}

func TestAliOssHostAndLink(t *testing.T) {
	template := AliTemplate{ossProperties: OssProperties{Endpoint: "oss-cn-hangzhou.aliyuncs.com"}}
	if got, want := template.getOssHost("t1", "demo"), "https://demo.oss-cn-hangzhou.aliyuncs.com"; got != want {
		t.Errorf("getOssHost() = %q, want %q", got, want)
	}
	if got, want := template.fileLink("t1", "demo", "upload/a.txt"), "https://demo.oss-cn-hangzhou.aliyuncs.com/upload/a.txt"; got != want {
		t.Errorf("fileLink() = %q, want %q", got, want)
	}
}

func TestTencentOssHost(t *testing.T) {
	template := TencentTemplate{ossProperties: OssProperties{BucketName: "demo", AppId: "1250000000", Region: "ap-guangzhou"}}
	if host, want := template.getOssHost("t1", ""), "https://demo-1250000000.cos.ap-guangzhou.myqcloud.com"; host != want {
		t.Errorf("getOssHost() = %q, want %q", host, want)
	}

	// 已带 appId 后缀时不再重复追加
	if got := template.withAppId("demo-1250000000"); got != "demo-1250000000" {
		t.Errorf("withAppId() = %q, want unchanged", got)
	}
}

func TestTencentOssHostFallsBackToEndpoint(t *testing.T) {
	template := TencentTemplate{ossProperties: OssProperties{BucketName: "demo", Endpoint: "https://cos.ap-guangzhou.myqcloud.com"}}
	if host, want := template.getOssHost("t1", ""), "https://demo.cos.ap-guangzhou.myqcloud.com"; host != want {
		t.Errorf("getOssHost() = %q, want %q", host, want)
	}
}

func TestOssRuleResolveViaTemplateBucket(t *testing.T) {
	// tenantMode=true 时各后端均应应用租户前缀
	template := AliTemplate{
		ossProperties: OssProperties{BucketName: "demo"},
		ossRule:       OssRule{tenantMode: true},
	}
	if got, want := template.bucket("t1", ""), "t1-demo"; got != want {
		t.Errorf("bucket() = %q, want %q", got, want)
	}
	if got, want := template.bucket("t1", "other"), "t1-other"; got != want {
		t.Errorf("bucket() = %q, want %q", got, want)
	}
}

// TestProviderBuildFileKeepsMd5 校验各后端 buildFile 与 MinioTemplate.buildFile 行为一致（保留 MD5）。
func TestProviderBuildFileKeepsMd5(t *testing.T) {
	const md5Hex = "5eb63bbbe01eeed093cb22bb8f5acdc3"

	ali := AliTemplate{ossProperties: OssProperties{Endpoint: "oss-cn-hangzhou.aliyuncs.com"}}
	if got := ali.buildFile("t1", "demo", "upload/a.txt", "a.txt", 11, md5Hex); got.Md5 != md5Hex {
		t.Errorf("ali buildFile().Md5 = %q, want %q", got.Md5, md5Hex)
	}

	qiniu := QiniuTemplate{ossProperties: OssProperties{Endpoint: "https://cdn.example.com"}}
	if got := qiniu.buildFile("t1", "demo", "upload/a.txt", "a.txt", 11, md5Hex); got.Md5 != md5Hex {
		t.Errorf("qiniu buildFile().Md5 = %q, want %q", got.Md5, md5Hex)
	}

	tencent := TencentTemplate{ossProperties: OssProperties{BucketName: "demo", Region: "ap-guangzhou"}}
	if got := tencent.buildFile("t1", "demo", "upload/a.txt", "a.txt", 11, md5Hex); got.Md5 != md5Hex {
		t.Errorf("tencent buildFile().Md5 = %q, want %q", got.Md5, md5Hex)
	}
}

func TestResolveMinioEndpoint(t *testing.T) {
	tests := []struct {
		in            string
		defaultSecure bool
		host          string
		secure        bool
	}{
		{in: "https://s3.us-east-1.amazonaws.com", defaultSecure: true, host: "s3.us-east-1.amazonaws.com", secure: true},
		{in: "http://minio:9000", defaultSecure: true, host: "minio:9000", secure: false},
		{in: "minio:9000", defaultSecure: false, host: "minio:9000", secure: false},
		{in: "s3.us-east-1.amazonaws.com", defaultSecure: true, host: "s3.us-east-1.amazonaws.com", secure: true},
	}
	for _, tt := range tests {
		host, secure := resolveMinioEndpoint(tt.in, tt.defaultSecure)
		if host != tt.host || secure != tt.secure {
			t.Errorf("resolveMinioEndpoint(%q, %v) = (%q, %v), want (%q, %v)",
				tt.in, tt.defaultSecure, host, secure, tt.host, tt.secure)
		}
	}
}

// TestS3TemplateVirtualHostLink：AWS 端点走虚拟主机风格，且链接与客户端寻址一致。
func TestS3TemplateVirtualHostLink(t *testing.T) {
	template, err := NewS3Template(&Config{
		Category: Category_S3, Endpoint: "https://s3.us-east-1.amazonaws.com",
		AccessKey: "ak", SecretKey: "sk", BucketName: "demo", Region: "us-east-1",
	}, OssRule{})
	if err != nil {
		t.Fatalf("NewS3Template() error = %v", err)
	}
	if got, want := template.getOssHost("t1", "demo"), "https://demo.s3.us-east-1.amazonaws.com"; got != want {
		t.Errorf("getOssHost() = %q, want %q", got, want)
	}
	if got, want := template.fileLink("t1", "demo", "upload/a.txt"), "https://demo.s3.us-east-1.amazonaws.com/upload/a.txt"; got != want {
		t.Errorf("fileLink() = %q, want %q", got, want)
	}
}

// TestS3TemplatePathLinkForNonCloudEndpoint：R2/Ceph/SeaweedFS 等非云厂商端点走路径风格。
func TestS3TemplatePathLinkForNonCloudEndpoint(t *testing.T) {
	template, err := NewS3Template(&Config{
		Category: Category_S3, Endpoint: "https://accountid.r2.cloudflarestorage.com",
		AccessKey: "ak", SecretKey: "sk", BucketName: "demo",
	}, OssRule{})
	if err != nil {
		t.Fatalf("NewS3Template() error = %v", err)
	}
	if got, want := template.getOssHost("t1", "demo"), "https://accountid.r2.cloudflarestorage.com/demo"; got != want {
		t.Errorf("getOssHost() = %q, want %q", got, want)
	}
}

func TestS3TemplateNormalizesScheme(t *testing.T) {
	// endpoint 省略协议时默认 https，链接应补全协议
	template, err := NewS3Template(&Config{
		Category: Category_S3, Endpoint: "s3.us-east-1.amazonaws.com",
		AccessKey: "ak", SecretKey: "sk", BucketName: "demo",
	}, OssRule{})
	if err != nil {
		t.Fatalf("NewS3Template() error = %v", err)
	}
	if got, want := template.getOssHost("t1", "demo"), "https://demo.s3.us-east-1.amazonaws.com"; got != want {
		t.Errorf("getOssHost() = %q, want %q", got, want)
	}
}

// TestMinioTemplatePathLink：MinIO 固定路径风格；scheme-less endpoint 默认 http（保持兼容）。
func TestMinioTemplatePathLink(t *testing.T) {
	template, err := NewMinioTemplate(&Config{
		Category: Category_Minio, Endpoint: "127.0.0.1:9000",
		AccessKey: "ak", SecretKey: "sk", BucketName: "demo",
	}, OssRule{})
	if err != nil {
		t.Fatalf("NewMinioTemplate() error = %v", err)
	}
	if got, want := template.getOssHost("t1", "demo"), "http://127.0.0.1:9000/demo"; got != want {
		t.Errorf("getOssHost() = %q, want %q", got, want)
	}
}
