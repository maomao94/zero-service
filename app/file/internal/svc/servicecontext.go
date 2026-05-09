package svc

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpc"

	"zero-service/app/file/internal/config"
	"zero-service/common/gormx"
	"zero-service/common/netx"
	"zero-service/common/ossx"
	"zero-service/model/gormmodel"
)

type ServiceContext struct {
	Config          config.Config
	DB              *gormx.DB
	Validate        *validator.Validate
	ThumbTaskRunner *threading.TaskRunner
	// Httpc 提供 go-zero httpc.Service，供 netx 客户端复用熔断/中间件能力。
	Httpc httpc.Service
	// NetClient 复用全局 netx 客户端，转推等场景直接使用，避免重复构造。
	NetClient *netx.Client
	// OssTemplateResolver 由 NewServiceContext 注入；单测可替换为 fake。
	OssTemplateResolver ossx.TemplateResolver
}

// loadOssConfig 按租户与 oss_code 从数据库加载 OSS 配置。
func (s *ServiceContext) loadOssConfig(ctx context.Context, tenantId, code string) (*ossx.Config, error) {
	var oss gormmodel.Oss
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND oss_code = ?", tenantId, code).First(&oss).Error; err != nil {
		return nil, err
	}
	return toOssConfig(&oss), nil
}

// GetOssTemplate 获取 OSS 模板（消除 Logic 层重复调用）
func (s *ServiceContext) GetOssTemplate(ctx context.Context, tenantId, code string) (ossx.OssTemplate, error) {
	if s.OssTemplateResolver == nil {
		return nil, errors.New("file svc: OssTemplateResolver not configured")
	}
	return s.OssTemplateResolver(ctx, tenantId, code)
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	db := gormx.MustOpenWithConf(c.DB)
	if isDevOrTest(c.Mode) {
		db.MustAutoMigrate(&gormmodel.Oss{})
	}

	httpcSvc := netx.NewHTTPCService("file-httpc")
	svc := &ServiceContext{
		Config:          c,
		DB:              db,
		Validate:        validator.New(),
		ThumbTaskRunner: threading.NewTaskRunner(c.ThumbTaskConcurrency),
		Httpc:           httpcSvc,
		NetClient: netx.NewClient(
			netx.WithEngine(netx.NewHTTPEngine(httpcSvc)),
		),
	}
	svc.OssTemplateResolver = ossx.NewTemplateResolver(c.Oss.TenantMode, svc.loadOssConfig)
	return svc
}

func isDevOrTest(mode string) bool {
	return mode == service.DevMode || mode == service.TestMode
}

func toOssConfig(oss *gormmodel.Oss) *ossx.Config {
	if oss == nil {
		return nil
	}
	return &ossx.Config{
		Category:   oss.Category,
		Endpoint:   oss.Endpoint,
		AccessKey:  oss.AccessKey,
		SecretKey:  oss.SecretKey,
		BucketName: oss.BucketName,
		AppId:      oss.AppId,
		Region:     oss.Region,
	}
}
