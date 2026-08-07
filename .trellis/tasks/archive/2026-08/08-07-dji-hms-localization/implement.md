# 实施计划

## 实施顺序

1. 新增严格设备三元组解析、domain 枚举、当前产品注册表与产品名称查询。
2. 将 HMS Args 调整为开放 map 并提供安全读取 helper，新增嵌入字典的 `HmsResolver`。
3. 为官方 dock/fpv key 选择、语言回退、alarmid、索引方向、未知 code 和缺失参数补单元测试。
4. 在 `ServiceContext` 构造 resolver，并经 hooks 注册边界注入 HMS handler。
5. 扩展 `DjiHmsAlert`，调整 HMS hook 保存产品名、三元组、args 和 message，并更新 hook 测试。
6. 在 `djicloud.proto` 追加新增 HMS 字段，运行 `app/djicloud/gen.sh`。
7. 更新 `ListHmsAlertsLogic` 和相关测试/调用点。
8. 更新示例配置中的 `Dji.Hms.Language`。
9. 审查生成 diff，确保无插件版本造成的无关重排。

## 验证命令

```bash
gofmt -w <changed-go-files>
go test ./common/djisdk
go test ./app/djicloud/internal/hooks ./app/djicloud/internal/logic ./app/djicloud/model/gormmodel
cd app/djicloud && ./gen.sh
go test ./app/djicloud/...
git diff --check
git status --short
```

如目标测试暴露直接消费者，再扩大到相应包；本任务不涉及共享可变状态或 goroutine，不默认运行 race test。

## 审查点

- `HmsResolver` 不导入 `app/djicloud` 类型。
- 生成文件仅由 `gen.sh` 更新。
- 新字段只追加、不改变已有 proto 字段号和 JSON 名。
- hook 继续逐条 `Create`，不得改为 Upsert。
- 缺失模板或参数不丢弃 HMS 历史记录。
- 最终 diff 不包含无关格式化、依赖变更或凭据。
