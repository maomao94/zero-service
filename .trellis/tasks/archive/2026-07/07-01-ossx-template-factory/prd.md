# ossx 提供 NewTemplate/MustNewTemplate 工厂函数，支持按 Category 自动创建 OSS 实例

## Goal

提取公共工厂函数 `NewTemplate` / `MustNewTemplate`，按 `Config.Category` 自动选择 OSS 实现，消除调用方硬编码 `NewMinioTemplate` 的问题。`Template()` 内部也复用 `NewTemplate`，新增 OSS 类型只需改一处。

## Requirements

- `NewTemplate(config *Config, ossRule OssRule) (OssTemplate, error)`：根据 `config.Category` 分支创建对应的 OssTemplate 实例（当前仅 Minio，后续可扩展 Qiniu/Ali/Tencent）
- `MustNewTemplate(config *Config, ossRule OssRule) OssTemplate`：调用 `NewTemplate`，失败时 panic，符合 go-zero `Must*` 风格
- `Template()`（DB 路径）的 `config.Category == Category_Minio` 分支改为调用 `NewTemplate`，消除重复
- `app/djicloud/internal/svc/servicecontext.go` 的 `NewMinioTemplate` 调用改为 `MustNewTemplate`

## Acceptance Criteria

- [x] `NewTemplate` 按 `Category` 正确分发到对应实现，不支持的 Category 返回 error
- [x] `MustNewTemplate` 正常时返回 OssTemplate，异常时 panic
- [x] `Template()` 内部复用 `NewTemplate`，行为不变
- [x] djicloud 改用 `MustNewTemplate`，行为不变
- [x] 现有测试全部通过

## Notes

- 轻量任务，PRD-only
