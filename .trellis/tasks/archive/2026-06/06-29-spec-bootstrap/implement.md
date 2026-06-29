# 执行计划

## 步骤

1. **索引一致性** — 对比 `backend/index.md` 与 `backend/` 实际文件、`guides/index.md` 与 `guides/` 实际文件
2. **占位文本扫描** — 全量 grep 检查
3. **文件路径验证** — 对每个 spec 中的引用的源文件路径抽样验证
4. **修复发现的问题** — 索引缺失条目、过时引用、不一致描述

## 验证

```bash
# 占位文本检查
grep -Rn "To be filled\|TODO:\|FIXME\|placeholder\|PLACEHOLDER" .trellis/spec/ | grep -v "trellis-template-policy\|bubble-tea-tui-guide"
```
