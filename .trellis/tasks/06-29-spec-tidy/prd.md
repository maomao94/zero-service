# 整理现有 .trellis/spec/

## Goal

删除实验性 spec 文件，整理现有 spec 的索引一致性和内容质量。

## Scope

1. 删除 `backend/uix-framework.md`（实验性，AI 生成 TUI 框架）
2. 删除 `guides/bubble-tea-tui-guide.md`（实验性，AI 生成 TUI 指南）
3. 从 `backend/index.md` 和 `guides/index.md` 中移除对应条目
4. 全量检查占位文本（"To be filled"、"TODO" 等）
5. 验证 `index.md` 与实际文件集一致

## Acceptance Criteria

- [ ] `uix-framework.md` 和 `bubble-tea-tui-guide.md` 已删除
- [ ] `backend/index.md` 无悬挂条目
- [ ] `guides/index.md` 无悬挂条目
- [ ] 占位文本清理完成
- [ ] `index.md` 文件与实际 spec 文件集一一对应
