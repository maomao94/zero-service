package skills

import (
	"context"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// resourceURI 格式: skill://{skill_name}/content
const resourceURIPrefix = "skill://"

// RegisterResources 注册 skills resources 到 MCP server
func RegisterResources(server *sdkmcp.Server, loader *Loader) {
	// 注册资源模板（支持动态 URI）
	server.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: resourceURIPrefix + "{skill_name}/content",
		Name:        "{skill_name}",
		Description: "读取指定 skill 的完整内容",
		MIMEType:    "text/markdown",
	}, listResourcesHandler(loader))

	// 注册单个资源的直接访问
	for _, skill := range loader.ListSkills() {
		registerSingleResource(server, loader, skill.Name)
	}
}

// registerSingleResource 注册单个 skill 为固定资源
func registerSingleResource(server *sdkmcp.Server, loader *Loader, skillName string) {
	resource := &sdkmcp.Resource{
		Name:        skillName,
		Description: fmt.Sprintf("读取 %s skill 的完整内容", skillName),
		MIMEType:    "text/markdown",
		URI:         resourceURIPrefix + skillName + "/content",
	}

	server.AddResource(resource, readResourceHandler(loader))
}

// listResourcesHandler 列出所有可用 resources
func listResourcesHandler(loader *Loader) sdkmcp.ResourceHandler {
	return func(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		// 解析 URI 模板参数
		name := ""
		if req.Params != nil && req.Params.URI != "" {
			name = extractSkillName(req.Params.URI)
		}

		skills := loader.ListSkills()

		contents := make([]*sdkmcp.ResourceContents, 0, len(skills))

		for _, skill := range skills {
			// 如果指定了 name，只返回对应的 skill
			if name != "" && name != "{skill_name}" && skill.Name != name {
				continue
			}

			contents = append(contents, &sdkmcp.ResourceContents{
				URI:  resourceURIPrefix + skill.Name + "/content",
				Text: buildResourceContent(skill),
			})
		}

		return &sdkmcp.ReadResourceResult{
			Contents: contents,
		}, nil
	}
}

// readResourceHandler 读取单个 resource 内容
func readResourceHandler(loader *Loader) sdkmcp.ResourceHandler {
	return func(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
		if req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("URI 不能为空")
		}

		skillName := extractSkillName(req.Params.URI)
		if skillName == "" {
			return nil, fmt.Errorf("无效的 skill URI: %s", req.Params.URI)
		}

		skill, ok := loader.GetSkill(skillName)
		if !ok {
			logx.Errorf("Skill not found: %s", skillName)
			return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
		}

		return &sdkmcp.ReadResourceResult{
			Contents: []*sdkmcp.ResourceContents{
				{
					URI:  req.Params.URI,
					Text: buildResourceContent(skill),
				},
			},
		}, nil
	}
}

// extractSkillName 从 URI 中提取 skill name
// skill://{skill_name}/content -> {skill_name}
func extractSkillName(uri string) string {
	// 去掉前缀
	uri = strings.TrimPrefix(uri, resourceURIPrefix)
	// 去掉后缀 /content
	uri = strings.TrimSuffix(uri, "/content")
	// 去掉可能的 /content/xxx
	if idx := strings.Index(uri, "/"); idx > 0 {
		uri = uri[:idx]
	}
	return uri
}

// buildResourceContent 构建资源内容（包含元数据和正文）
func buildResourceContent(skill *Skill) string {
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(skill.Name)
	sb.WriteString("\n\n")

	if skill.Description != "" {
		sb.WriteString(skill.Description)
		sb.WriteString("\n\n")
	}

	if len(skill.AllowedTools) > 0 {
		sb.WriteString("**Allowed Tools:** ")
		sb.WriteString(strings.Join(skill.AllowedTools, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString(skill.Content)

	return sb.String()
}
