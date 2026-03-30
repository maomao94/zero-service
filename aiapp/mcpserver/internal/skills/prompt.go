package skills

import (
	"context"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// PromptTemplate prompt 模板
type PromptTemplate struct {
	Name        string
	Description string
	Arguments   []*sdkmcp.PromptArgument
	Content     string // 模板内容，支持 {{variable}} 替换
}

// RegisterPrompts 注册 skills prompts 到 MCP server
func RegisterPrompts(server *sdkmcp.Server, loader *Loader) {
	// 注册通用的 skill 提示词
	registerSkillPrompts(server, loader)
}

// registerSkillPrompts 注册所有 skill 相关的 prompts
func registerSkillPrompts(server *sdkmcp.Server, loader *Loader) {
	skills := loader.ListSkills()

	for _, skill := range skills {
		registerSinglePrompt(server, skill)
	}
}

// registerSinglePrompt 注册单个 skill 的 prompt
func registerSinglePrompt(server *sdkmcp.Server, skill *Skill) {
	// 构建 prompt argument
	args := []*sdkmcp.PromptArgument{
		{
			Name:        "task",
			Description: "用户想要完成的任务",
			Required:    true,
		},
		{
			Name:        "context",
			Description: "额外的上下文信息（可选）",
			Required:    false,
		},
	}

	prompt := &sdkmcp.Prompt{
		Name:        skill.Name + "-guide",
		Description: fmt.Sprintf("使用 %s 完成任务的引导提示词", skill.Name),
		Arguments:   args,
	}

	handler := getPromptHandler(skill)
	server.AddPrompt(prompt, handler)
}

// getPromptHandler 获取 prompt 处理函数
func getPromptHandler(skill *Skill) sdkmcp.PromptHandler {
	return func(ctx context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
		if req.Params == nil {
			return nil, fmt.Errorf("参数不能为空")
		}

		// 获取参数
		task := ""
		contextExtra := ""

		if req.Params.Arguments != nil {
			if t, ok := req.Params.Arguments["task"]; ok {
				task = t
			}
			if c, ok := req.Params.Arguments["context"]; ok {
				contextExtra = c
			}
		}

		// 构建 prompt 内容
		content := buildPromptContent(skill, task, contextExtra)

		return &sdkmcp.GetPromptResult{
			Messages: []*sdkmcp.PromptMessage{
				{
					Role: sdkmcp.Role("user"),
					Content: &sdkmcp.TextContent{
						Text: content,
					},
				},
			},
		}, nil
	}
}

// buildPromptContent 构建 prompt 内容
func buildPromptContent(skill *Skill, task, contextExtra string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("你是 %s 领域的专家。\n\n", skill.Name))

	if skill.Description != "" {
		sb.WriteString(fmt.Sprintf("## 领域描述\n%s\n\n", skill.Description))
	}

	if task != "" {
		sb.WriteString(fmt.Sprintf("## 用户任务\n%s\n\n", task))
	}

	if contextExtra != "" {
		sb.WriteString(fmt.Sprintf("## 额外上下文\n%s\n\n", contextExtra))
	}

	// 添加 skill 核心内容（取前 2000 字符作为引导）
	guideContent := skill.Content
	if len(guideContent) > 2000 {
		guideContent = guideContent[:2000] + "\n\n...(内容已截断，完整内容请通过 resources 读取)"
	}

	sb.WriteString("## 参考知识\n")
	sb.WriteString(guideContent)

	return sb.String()
}

// ListPromptTemplates 返回所有 prompt 模板的元数据（不包含处理器）
func ListPromptTemplates(loader *Loader) []*PromptTemplate {
	skills := loader.ListSkills()
	templates := make([]*PromptTemplate, 0, len(skills))

	for _, skill := range skills {
		templates = append(templates, &PromptTemplate{
			Name:        skill.Name + "-guide",
			Description: fmt.Sprintf("使用 %s 完成任务的引导提示词", skill.Name),
			Arguments: []*sdkmcp.PromptArgument{
				{
					Name:        "task",
					Description: "用户想要完成的任务",
					Required:    true,
				},
				{
					Name:        "context",
					Description: "额外的上下文信息（可选）",
					Required:    false,
				},
			},
			Content: skill.Content,
		})
	}

	return templates
}

// GetPromptContent 根据 name 和参数获取渲染后的 prompt
func GetPromptContent(loader *Loader, name string, args map[string]string) (string, error) {
	// 解析 prompt name: {skill_name}-guide
	if !strings.HasSuffix(name, "-guide") {
		return "", fmt.Errorf("无效的 prompt name: %s", name)
	}

	skillName := strings.TrimSuffix(name, "-guide")
	skill, ok := loader.GetSkill(skillName)
	if !ok {
		logx.Errorf("Skill not found for prompt: %s", skillName)
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	task := ""
	contextExtra := ""

	if args != nil {
		task = args["task"]
		contextExtra = args["context"]
	}

	return buildPromptContent(skill, task, contextExtra), nil
}
