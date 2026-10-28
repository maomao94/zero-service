# Zero 工作台

Zero Service 的统一入口与运行状态面板：一屏看清各子系统是否可达，并直接跳转到对应工具。

## 功能

- **状态总览**：在线模块数、平均响应、下次检测倒计时，以及最近 40 次检测的聚合响应趋势。
- **服务模块**：Live 会议、SocketIO 网关、OpenAPI 控制台、GitLab 仓库四个入口，卡片展示实时延迟、最近检测时间和逐次检测条。
- **GitHub 入口**：跨满模块网格的横向卡片，直达 `maomao94/zero-service` 公开仓库，可复制仓库地址。
- **端点速查**：`/v1/`、`/live/v1/`、`/socket.io/`、`/recordings/` 等同源路径与上游端口，一键复制完整地址。
- **检测记录**：记录启动事件与上下线变化（不刷屏），可清空；无变化时给出空状态说明。
- **运行环境**：运行模式、站点、版本、构建时间与检测周期。
- **命令面板**：`⌘K` / `Ctrl K` 搜索模块、端点与操作，支持方向键与回车。
- **深浅主题**：跟随系统偏好，可手动切换并记住选择。
- **响应式**：≥1024px 固定侧栏；窄屏收起为抽屉，卡片单列。

## 子系统与端口

| 子系统 | 开发端口 | 访问位置 |
| --- | --- | --- |
| Live 视频会议 | 5178 | 站点根路径 `/` |
| SocketIO 网关测试 | 5179 | 同主机 `:8090` |
| OpenAPI 测试控制台 | 5181 | 路径 `/openapi/` |
| GitLab 代码仓库 | — | `https://gitlab.localhost/` |
| GitHub 代码仓库 | — | `https://github.com/maomao94/zero-service`（外部站点，不参与探活） |
| 工作台自身 | 5180 | — |

生产环境由 nginx 的 8091 站点托管工作台静态产物，并代理各健康检查地址（见仓库根目录 `deploy/nginx/conf.d/default.conf`）。

## 快速开始

```bash
cd web/workspace
npm install
npm run dev
```

打开 http://127.0.0.1:5180 。工作台只负责入口与探活，子系统可按需启动；未启动的模块显示「未启动」，不影响其他模块。

## 健康检测

每 15 秒并发探测一次**同源**健康检查地址（避免跨域与写死端口），页面不可见时暂停，回到前台立即补检。

| 模块 | 检查地址 | 开发代理目标 | 生产代理 |
| --- | --- | --- | --- |
| Live 视频会议 | `/health/live` | `127.0.0.1:5178/` | 容器内 80 `/healthz` |
| SocketIO 网关测试 | `/health/socketio` | `127.0.0.1:5179/` | 容器内 8090 `/` |
| OpenAPI 控制台 | `/health/openapi` | `127.0.0.1:5181/` | 宿主机 `11005/v1/health` |
| GitLab 代码仓库 | `/health/gitlab` | `127.0.0.1:80/users/sign_in`（Host: `gitlab.localhost`） | 容器内 80 同路径 |

本地未运行 nginx 时 GitLab 会显示「未启动」，属于预期；需要联调时可启动网关或设置 `VITE_GITLAB_URL` 指向可达实例。

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `VITE_LIVE_URL` | Live 会议访问地址；留空按「当前主机 + 80/443」推导 |
| `VITE_SOCKETIO_URL` | SocketIO 测试工具地址；留空按「当前主机 + 8090」推导 |
| `VITE_OPENAPI_URL` | 控制台地址；留空使用同源 `/openapi/` |
| `VITE_GITLAB_URL` | GitLab 地址；留空使用 `https://gitlab.localhost/` |
| `VITE_GITHUB_URL` | GitHub 仓库地址；留空使用 `https://github.com/maomao94/zero-service` |

开发态默认值见 `.env.development`。生产构建留空即可按站点自动推导。

## 目录结构

```
src/
├── App.tsx                 # 页面编排与命令面板数据
├── components/
│   ├── ActivityPanel.tsx   # 检测记录
│   ├── BrandMark.tsx       # 工作台标识
│   ├── CommandPalette.tsx  # ⌘K 命令面板
│   ├── EndpointTable.tsx   # 端点速查
│   ├── EnvironmentPanel.tsx
│   ├── ExternalCard.tsx    # GitHub 等外部仓库入口
│   ├── ModuleCard.tsx      # 模块卡片与逐次检测条
│   ├── Rail.tsx            # 侧栏：状态、运行概况、快捷键、主题
│   ├── Sparkline.tsx       # 聚合响应趋势
│   ├── StatusBoard.tsx     # 状态总览面板
│   ├── Toast.tsx           # 轻提示 live region
│   └── TopBar.tsx
├── data/modules.ts         # 模块、端点与检测周期配置
├── hooks/                  # useHealth / useTheme / useNow / useToast
├── lib/                    # 探活、剪贴板、格式化、平台判定
└── styles/
    ├── tokens.css          # 设计令牌与深浅主题
    └── app.css             # 布局与组件样式
```

新增子系统时，在 `src/data/modules.ts` 增加一条配置（名称、描述、图标、强调色、访问地址、健康检查地址），并在 nginx / vite 中补上对应的 `/health/<id>` 代理即可。

## 构建与部署

```bash
cd web/workspace
npm run build

# 或在仓库根目录统一构建全部前端
bash deploy/nginx/build-web.sh workspace
```

构建产物输出到 `web/workspace/dist`，再由脚本汇总到 `deploy/nginx/dist/workspace`。构建期通过 Vite `define` 注入 `__APP_VERSION__`（取自 `package.json`）与 `__BUILD_TIME__`，页脚与运行环境面板会展示这两项。

## 实现约定

- 健康检查只使用同源地址，跨域与端口推导问题都收敛在 nginx / vite 配置里。
- 剪贴板优先 `navigator.clipboard`，900ms 未返回或不可用时回退 `execCommand`，保证内网 http 源站也能复制。
- 上下线才写入检测记录；首次检测只写一条「工作台已就绪」事件，避免首屏噪音。
- 主题在 `index.html` 内联脚本中于首屏前确定，避免深色模式闪白。
- 动效遵循 `prefers-reduced-motion`，交互控件均有可见焦点样式。
