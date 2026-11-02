# 变更日志

本文件记录项目的所有重要变更，格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

分类说明：`新增` / `变更` / `弃用` / `移除` / `修复` / `文档` / `依赖`。

## [未发布]

### 新增

- 文档质量门禁：`scripts/check_docs.py` 校验相对链接、页内锚点与代码块闭合，并由 `.github/workflows/docs.yml` 在 CI 执行。
- 仓库治理文件：`SECURITY.md`、`CODE_OF_CONDUCT.md`、`SUPPORT.md`、Issue 表单与 PR 模板。
- 文档：`docs/antsx.md`（原语选型与项目实践）、`docs/ai-collaboration.md`（AI 协作与技能集来源）。

### 变更

- 错误码：网关折叠策略从 `errx` 下沉到 `gtwx`，默认不折叠、按 `Error.Fold.Codes` 折叠；`100603` 强制透传。
- 文档：按「规则/示例」「正文/附录」拆分 `trigger-rrule-api-guide.md` 与 `iec104-message.md`。

### 移除

- 文档：移除 `docs/superpowers.md`、`docs/matt-pocock-skills.md`（外部技能集目录，改为在 `docs/ai-collaboration.md` 记录来源）。
- 文档：移除 `docs/antsx-vs-reactive.md`，内容并入 `docs/antsx.md`。

### 修复

- 文档门禁：`scripts/check_docs.py` 补齐 HTML `<img>`/`<a href>` 校验、重复标题的 `-1` 锚点、`~~~` 与四反引号围栏、`.mmd`/`.png` 配对检查。
- 文档：`README.md` 仓库结构补上 `gtw/`、`zerorpc/`、`cli/`、`util/`、`swagger/`、`third_party/` 与 goctl 模板目录。
- 文档：Bug 报告模板的服务下拉去掉非服务的 `streamevent`，补齐 `iecagent`、`socketpush`、`file`/`gis`/`podengine`/`openapi` 等实际服务。
- 文档：修正 `CONTRIBUTING.md` 中 CI 触发分支说明（实际为 `master` 与 `develop`）。
- 文档：修正 `docs/isp/isp.md` 中 ispagent 包路径（`internal/ispclient`）。

## [0.1.14] - 2026-10-28

### 新增

- 开放平台：请求签名（`X-Sign` 头 + 原始 body 签名），配套签名密钥查看接口与本地 `keytool`。
- 前端：工作台与子系统界面重构，支持本地签发 Token。

### 修复

- 开放网关：收紧签名密钥开关并修正启停时序。
- 前端：修正签名选项位置与请求状态提示。

### 文档

- 重整文档索引与配图，修复失效引用。
- 校正开放平台签名协议并补充本地联调说明。
- 区分平台错误码与厂商错误码体系。

## [0.1.13] - 2026-10-16

### 新增

- 定时任务：调度组件投产前 P0 改造（开启算法统一 + misfire 策略）。
- 视频会议：参会人过滤 Egress 机器人，会议列表支持录屏展示与分页。

### 变更

- 日期时间格式化统一复用 `carbonx`，去除裸 `time.Format`。

### 文档

- 标注 DJI 服务为第一代上云 API 参考实现，并指引上云 API 2。

## [0.1.12] - 2026-10-14

### 新增

- 对象存储：新增七牛、阿里云、腾讯云后端实现；未知大小流式上传按后端收敛内存并统一 size 语义。
- CI：新增 GitHub Actions 依赖监控，补充 `go mod tidy` 校验。

### 依赖

- 各服务 Dockerfile 统一为 `golang:1.26-alpine3.22` 基础镜像。

## [0.1.11] - 2026-10-10

### 新增

- 视频会议：LiveKit Egress 会议录制（后端 / 网关 / 前端 / 部署）。

### 修复

- 录制开始时间同步改为「值未变即跳过」，避免轮询写库。
- 录制 Egress Webhook 按状态分流并同步开始时间。
- 显式通告宿主机 `node_ip`，修复浏览器 ICE 建连失败。
- 工作台在线检测适配 nginx 代理；录屏播放地址复制为绝对路径。

### 变更

- 录制迁移到统一 `StartEgress`（TemplateSource）。

## [0.1.10] - 2026-10-08

### 变更

- Dependabot：PR 目标分支改为 `develop`，限制同时打开的 PR 数量并添加标签。

## 更早版本

`v0.1.9` 及更早版本的变更记录见 [Releases](https://github.com/maomao94/zero-service/releases)。
