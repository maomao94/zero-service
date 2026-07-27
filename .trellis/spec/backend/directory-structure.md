# 目录结构与仓库边界

## 适用范围

新增目录、移动代码、选择复用位置、判断文件是否可直接编辑或决定是否需要独立 Spec 时读取。

## 项目结构

- 仓库只有一个 `go.mod`，模块名为 `zero-service`；服务之间可以共享 `common/`，但仍应保持明确的服务所有权。
- `app/` 是核心与工业协议服务，`aiapp/` 是 AI 服务，`socketapp/` 是实时连接服务，`gtw/` 是 BFF/API 网关，`facade/` 是跨服务门面。
- 服务私有实现放在服务的 `internal/`；跨服务且已经证明稳定复用的能力才进入 `common/`。
- 业务数据模型集中在 `model/` 或服务自有模型目录；先沿用目标服务现有数据访问方式，不为统一形式跨服务搬迁。
- 部署、项目说明和外部协议副本分别位于 `deploy/`、`docs/`、`third_party/`，不与业务实现混放。

依据：`README.md`、`docs/architecture.md`、`go.mod`、`app/`、`aiapp/`、`socketapp/`、`gtw/`、`facade/`、`common/`。

## 手写与生成边界

- `.proto`、`.api` 和服务目录的 `gen.sh` 是生成链路入口；带 `Code generated` 或 `DO NOT EDIT` 标记的 Go 文件不得手工维护。
- 业务逻辑通常位于 `internal/logic`，依赖装配位于 `internal/svc`，配置结构位于 `internal/config`。
- 生成后的 server/handler/route/type/pb 只是契约产物，不应被提炼成公共编码模式。
- `third_party/` 中的 proto 是外部依赖契约；除非任务就是同步外部契约，否则不要按本项目风格改写。

依据：`app/trigger/gen.sh`、`app/bridgegtw/gen.sh`、`aiapp/aisolo/gen.sh` 及对应生成文件头。

## 不建立独立 Spec 的范围

- `common/gnetx` 当前包含明确的原型/简单场景定位，不进入必读规范。
- `app/xfusionmock` 是 Mock/Demo 服务，只服从项目和服务通用规则。
- `1.7.1/`、`1.9.x/` 是历史模型快照，不作为当前实现依据。
- 小型纯函数工具包由相邻代码、测试和 [公共包设计](./common-package-design.md) 覆盖，不机械创建“一包一文档”。

实验代码变成稳定基础设施后，必须先证明存在正式调用方、清晰所有者、生命周期和测试，再决定是否补充 Spec。

## 反模式

- 因多个服务看起来相似，就把带业务状态的代码抽入 `common/`。
- 手改生成文件绕过源契约或生成脚本。
- 从历史快照、Mock、示例或实验组件反推当前生产契约。
- 为临时实现创建长期全局规则。

## 验证

```bash
go list ./...
git diff --name-only
git diff --check
```

目录变化还要检查 `README.md`、`docs/README.md`、生成脚本和导入路径是否需要同步。
