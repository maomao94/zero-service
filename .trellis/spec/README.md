# zero-service Trellis Spec

本目录记录修改 `zero-service` 时必须遵守的稳定编码契约。目录遵循 Trellis 的标准骨架：代码规范放在技术层目录，跨层分析方法放在 `guides/`。

## 目录结构

```text
.trellis/spec/
├── README.md
├── backend/
│   ├── index.md
│   └── *.md
└── guides/
    ├── index.md
    └── *-guide.md
```

本仓库是单 Go module 的多服务后端项目，没有前端代码，因此不创建空的 `frontend/`。`app/`、`aiapp/`、`socketapp/`、`gtw/`、`facade/`、`common/` 和 `model/` 都由 `backend/` 覆盖；它们是源码所有权边界，不是 Trellis 顶层 spec layer。

## 如何使用

1. 开发前先读 [backend/index.md](./backend/index.md)，按改动范围选择具体 Code-Spec。
2. 改动跨目录、跨进程或准备抽公共能力时，再读 [guides/index.md](./guides/index.md) 选择思考指南。
3. 以当前源码、测试、契约源和配置验证规则；Spec 与实现冲突时先调查原因，不机械服从过时文本。
4. 开发完成后执行所选 Code-Spec 的验证项，并按 [质量规范](./backend/quality-guidelines.md) 扩大检查范围。

## 内容边界

- `backend/*.md` 是可执行的 Code-Spec：描述适用范围、项目契约、反模式、证据和验证。
- `guides/*-guide.md` 是思考路径：帮助追踪跨层数据流、复用和文档归属，不替代 Code-Spec。
- 完整 API、协议字段和生成物以 `.proto`、`.api`、typed 协议源码和 Go doc 为准。
- 使用说明、部署流程和架构介绍放在仓库 `README.md` 与 `docs/`。
- 一次性方案、研究和排障历史放在 `.trellis/tasks/`，任务完成后归档。

## 不纳入独立 Spec

实验/原型、Mock/Demo、历史快照、生成代码和第三方契约副本不建立独立 Spec。当前包括 `common/gnetx`、`app/xfusionmock`、`1.7.1/`、`1.9.x/`、带生成标记的 Go 文件和 `third_party/`。

排除不代表可以忽略通用质量规则。实验代码若成为正式依赖，应先确认稳定调用方、所有者、生命周期和测试，再决定合并进现有规范或新增 Code-Spec。
