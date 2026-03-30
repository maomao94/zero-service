package main

import (
	"flag"
	"fmt"

	"zero-service/aiapp/mcpserver/internal/config"
	"zero-service/aiapp/mcpserver/internal/skills"
	"zero-service/aiapp/mcpserver/internal/svc"
	"zero-service/aiapp/mcpserver/internal/tools"
	"zero-service/common/mcpx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/mcpserver.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()
	logx.DisableStat()

	// 创建带鉴权的 MCP 服务器（与 go-zero mcp.NewMcpServer 对齐）
	server := mcpx.NewMcpServer(c.McpServerConf)
	defer server.Stop()

	// 创建 ServiceContext（包含 skills loader）
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		logx.Errorf("创建 ServiceContext 失败: %v", err)
		return
	}

	// 注册 Skills Resources 和 Prompts
	if svcCtx.SkillsLoader != nil {
		skills.RegisterResources(server.Server(), svcCtx.SkillsLoader)
		skills.RegisterPrompts(server.Server(), svcCtx.SkillsLoader)
		logx.Infof("已注册 %d 个 skills", len(svcCtx.SkillsLoader.ListSkills()))

		// 启动热加载
		if err := svcCtx.SkillsLoader.StartWatcher(); err != nil {
			logx.Errorf("启动 skills 热加载失败: %v", err)
		}
		defer svcCtx.SkillsLoader.Stop()
	}

	// 注册所有工具
	tools.RegisterAll(server.Server(), svcCtx)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting MCP server at %s:%d ...\n", c.Host, c.Port)
	server.Start()
}
