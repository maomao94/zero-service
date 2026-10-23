package svc

import (
	"context"
	"time"

	"zero-service/common/gormx"
	"zero-service/common/grpcx"
	"zero-service/facade/openapi/openapi"
	"zero-service/model/gormmodel"
	"zero-service/openapp/opengtw/internal/config"
	"zero-service/openapp/opengtw/internal/middleware"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config          config.Config
	DB              *gormx.DB
	AppStore        *gormmodel.OpenAppStore
	SecretStore     *gormmodel.OpenAppSecretStore
	CredentialCache *middleware.CredentialCache
	OpenApiCli      openapi.OpenApiRpcClient
	UserAuth        rest.Middleware
	OpenAuth        rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := gormx.MustOpenWithConf(c.DB)
	if c.Mode == service.DevMode || c.Mode == service.TestMode {
		db.MustAutoMigrate(&gormmodel.OpenApp{}, &gormmodel.OpenAppSecret{})
	}

	appStore := gormmodel.NewOpenAppStore(db)
	secretStore := gormmodel.NewOpenAppSecretStore(db)
	credentialCache := middleware.NewCredentialCache(appStore, secretStore, 60*time.Second)
	credentialCache.MustRefresh(context.Background())
	go credentialCache.Run(context.Background())

	openApiCli := openapi.NewOpenApiRpcClient(zrpc.MustNewClient(c.OpenApiConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor),
		zrpc.WithUnaryClientInterceptor(grpcx.AppIDMetadataInterceptor)).Conn())

	return &ServiceContext{
		Config:          c,
		DB:              db,
		AppStore:        appStore,
		SecretStore:     secretStore,
		CredentialCache: credentialCache,
		OpenApiCli:      openApiCli,
		UserAuth:        middleware.NewUserAuthMiddleware(c.JwtAuth.ClaimMapping).Handle,
		OpenAuth:        middleware.NewOpenAuthMiddleware(credentialCache).Handle,
	}
}

// RefreshCredentials 管理面变更后立即刷新凭据，避免等待 60s 周期。
func (s *ServiceContext) RefreshCredentials(ctx context.Context) {
	if s.CredentialCache == nil {
		return
	}
	if err := s.CredentialCache.Refresh(ctx); err != nil {
		logx.WithContext(ctx).Errorf("refresh credentials after admin change failed: %v", err)
	}
}
