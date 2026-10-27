# OpenAPI 测试控制台

`web/openapi` 是 `app/opengtw` 的配套调试控制台，用于验证开放 API、管理应用凭据和查阅错误码。仅限开发/测试环境使用。

## 功能

- **连接设置**：Base URL 支持当前站点同源、本地直连和自定义环境；实时验证 `/v1/health`。
- **API 调试**：19 个 method 的示例与字段说明、method 分组搜索、写操作提示、请求 ID 生成、JSON 校验/格式化/压缩、Ctrl/⌘ + Enter 发送。
- **响应面板**：HTTP 状态、业务错误码、耗时、大小、`X-Request-Id`；数据/原始响应/错误说明/请求回显四个视图，JSON 语法高亮。
- **请求历史**：最近 50 条，支持搜索、method 与状态过滤、回填重放；只存 method、data、状态等，不存 secret。
- **应用管理**：JWT 鉴权，创建应用、启停、重置密钥、分页查询；secret 仅在创建/重置后展示一次。
- **错误码速查**：BC=07 与常用 extproto 错误码、语义 HTTP 映射、处理建议和基础设施折叠规则。

当前开放 19 个 method：Podman 4 个、Modbus I/O 15 个。Modbus 字段、示例和写操作注意事项见 [`docs/open-api/modbus.md`](../../docs/open-api/modbus.md)。

## 技术栈与端口

- React 18 + TypeScript + Vite 6
- 开发端口：`5181`
- 生产路径：`/openapi/`

## 本地开发

```bash
cd web/openapi
npm install
npm run dev
```

打开 http://127.0.0.1:5181 。

开发态 Vite 已代理：

| 前缀 | 目标 |
| --- | --- |
| `/v1` | `http://127.0.0.1:11005` |
| `/open-admin` | `http://127.0.0.1:11005` |

只要 opengtw 已启动，Base URL 保持「当前站点」即可，不需要处理浏览器跨域。

### 本地依赖

```bash
# 终端 1：Mock 模式的转发服务
cd app/openapi && go run . -f etc/openapi.yaml

# 终端 2：HTTP 网关（本地默认金仓 zero 库；dev/test 自动建表）
cd app/opengtw && go run . -f etc/opengtw.yaml
```

管理面 JWT 需使用与 `etc/opengtw.yaml` 中 `JwtAuth.AccessSecret` 相同的密钥签发。

## 构建与部署

```bash
cd web/openapi
npm run build

# 或在仓库根目录统一构建所有前端
bash deploy/nginx/build-web.sh openapi
```

构建时 Vite `base` 固定为 `/openapi/`，产物复制到 `deploy/nginx/dist/openapi`，由 nginx 静态托管：

- 控制台：`/openapi/`
- 健康检查：`/health/openapi` → opengtw `/v1/health`
- 开放 API：`/v1/` → opengtw `11005`
- 管理面：`/open-admin/` 默认返回 404，不对外转发

## 安全约束

- App ID / Secret / 管理 JWT 只保存在当前浏览器 `localStorage`，不会发送到其他存储。
- 页面明确标记「仅限测试环境」，**不要部署到生产环境**。
- 请求历史不保存 secret，也不保存完整响应体。
- 管理面能力依赖同源 nginx 代理；如果生产不转发 `/open-admin/`，控制台的应用管理页不可用。

## 开发约束

1. `src/lib/methods.ts` 的 method 清单必须与 `app/openapi/internal/dispatch/registry.go` 人工同步。
2. `src/lib/errors.ts` 必须与 `third_party/extproto.proto`、`common/errx/codec.go` 同步。
3. 路径型站点不能按端口推导地址，生产访问必须使用 `/openapi/`；工作台通过 `VITE_OPENAPI_URL` 覆盖开发地址。
4. 跨域直连需要 opengtw 开启 CORS；本地开发优先使用 Vite 同源代理。

## 目录

```text
web/openapi/
├── package.json
├── vite.config.ts          # 5181、/v1 与 /open-admin 代理、构建 base
├── index.html
└── src/
    ├── App.tsx             # 三个页签与调试主流程
    ├── components/         # 连接、方法、请求、响应、历史、弹窗等
    ├── pages/              # 应用管理、错误码速查
    ├── lib/
    │   ├── http.ts         # Basic/Bearer 请求、信封解析、健康检查
    │   ├── methods.ts      # method 清单与示例（需与后端同步）
    │   ├── errors.ts       # 错误码表
    │   ├── storage.ts      # localStorage 设置与历史
    │   └── format.ts       # 时间/大小/JSON 辅助
    ├── types.ts
    └── styles.css
```
