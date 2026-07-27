# 验证记录

## Trellis 结构

- `python3 ./.trellis/scripts/get_context.py --mode packages` 输出 `Spec layers: backend`。
- `.trellis/spec/` 只包含根 `README.md`、`backend/` 和 `guides/`；不存在旧 `project/`、`service/`、`common/`、`domain/` 空 layer。
- `backend/index.md` 覆盖 19/19 份 Code-Spec，无遗漏或陈旧链接。
- `guides/index.md` 覆盖 3/3 份 Guide，无遗漏或陈旧链接。

## 内容与引用

- 所有 Markdown 相对链接的目标均存在。
- 所有非索引正文均包含“适用范围”“依据”“反模式”“验证”。
- `依据` 行引用的确定源码/测试/文档路径均存在；通配测试路径至少匹配一个文件。
- 未发现 `TBD`、模板占位文本、旧 layer 路径、`app/geofence` 旧路径、个人绝对路径或常见明文凭据模式。
- 实验/原型、Mock/Demo、历史快照、生成代码和第三方副本均没有独立 Spec。

## Trellis 与 Git

- `python3 ./.trellis/scripts/task.py validate 07-27-rebuild-project-specs` 通过，`implement.jsonl` 与 `check.jsonl` 均有效。
- `git diff --check` 通过。
- `git status --short` 仅包含 `.trellis/` 下的 Spec 和本任务记录变更，没有产品源码改动。

## 范围说明

本任务只修改文档，没有运行全仓 Go 测试。仓库分析阶段已通过 `go list ./...` 确认 module/包结构；具体契约通过定向读取当前实现和现有测试复核。
