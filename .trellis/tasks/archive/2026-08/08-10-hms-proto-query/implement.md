# 实施计划

- [ ] 修改 `djicloud.proto` 的 `HmsAlertInfo` 声明，删除 reserved，按语义顺序连续重编号并校对注释。
- [ ] 在 `app/djicloud` 执行 `./gen.sh` 并审查生成文件。
- [ ] 更新 `ListHmsAlerts` 字段映射和查询测试。
- [ ] 从 GORM model、Proto 和 Logic 中同步删除五个 args 平铺索引字段，不保留 `gorm:"-"` 兼容字段。
- [ ] 搜索删除字段的生产代码消费者。
- [ ] 运行 `go test ./app/djicloud/...` 和 `git diff --check`。
