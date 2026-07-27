# 设计：按 Trellis 标准重建项目 Spec

## 设计目标

新 `.trellis/spec/` 采用 Trellis 官方模板的职责划分：技术层 Code-Spec 负责可执行编码契约，`guides/` 负责跨层思考路径，根 `README.md` 负责入口和边界说明。

`zero-service` 是单 Go module 的多服务后端仓库，没有前端代码，因此保留 `backend/`，不创建空 `frontend/`。源码中的 `app/`、`common/`、`model/` 等是代码所有权边界，不提升为 Trellis 顶层 layer。

## 最终目录

```text
.trellis/spec/
├── README.md
├── backend/
│   ├── index.md
│   ├── 基础规范
│   ├── 公共基础设施规范
│   └── 稳定领域契约
└── guides/
    ├── index.md
    ├── cross-layer-thinking-guide.md
    ├── code-reuse-thinking-guide.md
    └── documentation-guide.md
```

## 组织原则

### `backend/`

覆盖所有 Go 后端目录。文件名按开发触发点命名，而不是按源码目录一一镜像：

- 基础规范：仓库边界、编码、质量、go-zero、契约生成、服务生命周期、错误边界。
- 公共基础设施：公共包设计、GORM、并发、消息客户端、crontask。
- 稳定领域契约：Trigger、ISP、IEC 104、DJI、GIS、实时事件、AI/MCP。

每份 Code-Spec 至少说明适用范围、必须保持的契约、反模式、当前证据和针对性验证。高风险基础设施还应明确状态/身份/字段所有权、失败矩阵或可观察场景。

### `guides/`

只保存跨主题问题清单和路由：跨层影响、复用决策、文档归属。Guide 的结论必须回到 `backend/` 的具体 Code-Spec，不复制实现规则。

### 根 `README.md`

解释 Spec 的用途、标准目录、阅读顺序、内容归属和排除政策。它不是第三个规则层，也不复制索引正文。

## 排除策略

以下内容不建立独立 Spec：

- `common/gnetx`：当前源码明确是原型/简单场景能力；稳定 ISP 边界写入 ISP 规范。
- `app/xfusionmock`：Mock/Demo 服务。
- `1.7.1/`、`1.9.x/`：历史模型快照。
- 带生成标记的 Go 文件与 `third_party/`：分别由源契约/生成脚本和上游契约维护。
- 完整 API 表、依赖版本抄录、历史排障过程、发布步骤和 Trellis 自身模板策略。

排除项仍服从基础编码和质量规则。实验能力只有在形成正式调用方、清晰所有者、生命周期和测试后，才进入现有 Code-Spec 或新增规范。

## 关键契约保留

- `crontask`：lease claim/complete CAS、零值/SQL NULL 终止语义、`RunNow` 不改变周期状态、成功才更新 `LastRun`。
- `gormx`：连接配置入口、原子 mixin、租户作用域、乐观锁、分页、Upsert 与条件更新所有权。
- ISP/IEC 104/DJI：连接身份、关联响应顺序、旧 session/lease 隔离、协议层所有权和锁外回调。
- GIS：项目坐标 `lon,lat`，H3 边界显式转换 `lat,lng`，GEOS 安全边界，H3 粗筛后精判。
- Trigger/实时/AI：异步接受与业务完成分离，状态写入所有权明确，不从 client API 推断可靠性保证。

## 迁移方式

直接用标准 `backend/ + guides/ + README.md` 替换旧文件集，不保留兼容跳转文件。旧规范中的事实逐条由当前源码、测试、配置和契约源复核；与实现矛盾时记录当前可证明的契约，而不是延续历史结论。

## 验证策略

- `get_context.py --mode packages` 只识别 `backend` 技术层，Guide 由工作流按需注入。
- 根入口、两层索引、索引链接和磁盘文件集一致。
- 所有 Markdown 相对链接与明确引用的仓库路径存在。
- 无占位、空标题、实验模块独立 Spec、个人绝对路径或敏感配置。
- `git diff --check` 通过，Git diff 只包含 `.trellis/` 文档。
- 本任务不改产品源码，因此不以全仓 Go 测试代替 Spec 内容审查。
