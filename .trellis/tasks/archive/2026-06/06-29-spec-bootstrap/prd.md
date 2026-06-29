# 补全 .trellis/spec/ 覆盖缺口

## Goal

整理现有 `.trellis/spec/`，确保内容一致、索引匹配、无占位文本。不做新增服务级 spec。

## Scope

- 验证所有 spec 文件引用的源文件路径存在
- 清除占位文本（"To be filled"、"TODO" 等）
- 确保 `backend/index.md` 与实际文件集一致
- 修复发现的不一致或过时内容

### Out of Scope

- 新增逐服务架构 spec
- 新增 common 包 spec
- 修改产品源代码

## Acceptance Criteria

- [ ] `grep -R "To be filled\|TODO:\|FIXME\|placeholder\|PLACEHOLDER" .trellis/spec/` 无命中（合法引用除外）
- [ ] `backend/index.md` 条目与 `backend/` 下文件一一对应
- [ ] 每个 spec 中的文件路径经抽样验证存在
- [ ] `guides/index.md` 路由表与 `guides/` 下文件一致
