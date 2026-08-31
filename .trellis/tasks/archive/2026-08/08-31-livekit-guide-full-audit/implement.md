# LiveKit 指南全面复核执行计划

## 文件映射

- 修改 `docs/livekit-integration-guide.md`：版本声明、学习路径、所有 API/配置/行为说明、源码与官方链接。
- 修改 `.trellis/spec/backend/livekit-guidelines.md`：后续实现必须遵守的版本、API、授权、Webhook、部署和验证契约。
- 修改 `docs/README.md` 与 `.trellis/spec/backend/index.md`：保持导航入口有效。
- 创建 `.trellis/tasks/08-31-livekit-guide-full-audit/research/livekit-guide-full-audit.md`：保存逐项核验矩阵、证据、修复和环境限制。
- 保留本目录的 `prd.md`、`design.md`、`implement.md` 作为任务依据，完成后归档。

## 执行步骤

1. 固化最新稳定版本和本地源码 commit，记录 GitHub release/tag、Go module 版本、Protocol 依赖和查询日期。
2. 建立全文章节清单，逐段提取 API 名称、字段、枚举、配置 key、URL 和行为断言，不采用抽样。
3. 对 Server SDK 入口、Room/Participant、Egress、Ingress、SIP、Agent、实时参与者、错误类型逐项读取源码并编译对应片段。
4. 对 Protocol auth、Webhook receiver、事件常量、生成 API client、oneof 和请求/响应字段逐项读取源码并编译对应片段。
5. 对 LiveKit Server service 实现、配置样例、部署文档和 release notes 逐项核对行为、权限、端口、独立服务与 Cloud/自托管边界。
6. 修订指南和 backend spec，删除错误或无法证明的固定说法，补充版本条件、来源链接、验证命令和明确限制。
7. 运行独立 Go 示例的 `go mod tidy`、`go test ./...`，并对所有可执行片段做编译检查；记录原生依赖、凭据和 Server 缺失导致的失败。
8. 运行 Markdown/URL/版本/API/secret/个人路径/占位符扫描，检查导航和源码链接，执行 `git diff --check`。
9. 由质量检查流程复核全文 diff、验收标准和未验证项，修复发现的问题后再次运行全部检查。
10. 提交完成的文档变更，归档 Trellis 任务并记录会话；最终报告必须区分稳定版本、源码验证版本、编译验证和未完成的端到端验证。

## 验证命令

```bash
git diff --check
git status --short
python3 ./.trellis/scripts/get_context.py --mode packages
```

核心示例使用临时 module，并以本地或稳定 tag SDK 作为明确依赖来源，至少运行：

```bash
go mod tidy
go test ./...
```

禁止以网络超时、缺少原生库或没有运行 LiveKit Server 为由声称全量验证通过。
