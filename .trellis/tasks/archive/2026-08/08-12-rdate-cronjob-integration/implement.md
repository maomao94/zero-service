# Implementation Plan

1. 给 CronJob 模型新增指定/排除时间 JSON 列，并扩展转换层平铺与重建。
2. 扩展内部 task data/helper，把两个列表传给 CronJob 编译器并写入 Extra。
3. Create/Update/Submit 传递列表，Get/List 通过 Proto 转换回显。
4. 更新 Store 配置白名单，支持替换/清空，同时保持在途更新原子拒绝。
5. 补模型转换、Store、Logic、首次 next_run、推进和预览测试。
6. 运行 CronJob/Logic/common 测试、race、Trigger 全量和 `git diff --check`。

## Rollback

- Logic、模型、转换、Store 和测试作为一个单元回滚。
- 新增可空列可以保留，旧代码忽略。
