# Implementation Plan

## Checklist

1. 检查现有 plan/batch terminate、cron claim、callback update 的事务与错误模式，确认可复用的 model API。
2. 在 model/store 增加按 plan/batch 作用域检查 running exec 的窄接口，明确返回数量/业务错误所需信息。
3. 在 `TerminatePlanLogic` 和 `TerminatePlanBatchLogic` 中加入事务内 running guard，成功路径不更新 exec。
4. 保持 cron claim 原有的父级 JOIN 查询和 ExecItem 乐观 CAS，不引入父级行锁、额外父级子查询或补偿状态。
5. 修复 callback 状态条件更新的竞争失败处理，避免零行更新仍写成功流水或触发收尾。
6. 补充终止成功/拒绝、各 exec 状态边界、父级 enabled、ExecItem CAS 竞争和 callback 条件更新测试；不增加依赖父级行锁的测试。
7. 运行 gofmt、目标包测试、race test、go vet、diff check，并检查无 proto 生成 diff。

## Validation

```bash
gofmt -w <changed-go-files>
go test ./app/trigger/...
go test -race ./app/trigger/...
go vet ./app/trigger/...
git diff --check
```

## Review Gates

- 终止拒绝时父级和通知必须保持不变。
- 成功终止不改变 exec 状态。
- 终止成功后 cron 不得将任何子项变为 running。
- 条件更新的 `RowsAffected == 0` 不得被当作成功。
- 修改只覆盖本任务相关文件。
