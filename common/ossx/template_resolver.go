package ossx

import "context"

// TemplateResolver 按 tenantId + 业务 code 解析出可操作的 OssTemplate（通常由配置加载 + Template 模板池实现）。
type TemplateResolver func(ctx context.Context, tenantId, code string) (OssTemplate, error)

// NewTemplateResolver 基于租户模式与 GetConfigFn 构造解析器，等价于对 Template 的固定参数封装。
func NewTemplateResolver(tenantMode bool, getConfig GetConfigFn) TemplateResolver {
	return func(ctx context.Context, tenantId, code string) (OssTemplate, error) {
		return Template(ctx, tenantId, code, tenantMode, getConfig)
	}
}
