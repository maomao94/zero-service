# File OSS 文件服务规范

## 适用范围

修改 `app/file` 的 OSS 管理、文件上传/签名/中继/转码操作，或公共库 `common/ossx`、`common/filex` 时读取。

## 服务总览

`app/file` 是 go-zero gRPC 服务，管理 OSS 配置并提供文件操作能力。当前仅 MinIO 后端实现。

| 属性 | 值 |
| --- | --- |
| 端口 | 21003 |
| RPC 数量 | 15 |
| Nacos | 可选（`NacosConfig.IsRegister`） |
| Interceptor | `interceptor.LoggerInterceptor` |
| 持久化 | GORM（`oss` 表） |
| 并发 | `threading.TaskRunner`（缩略图）、`antsx.Reactor`（中继上传） |

依据：`app/file/file.go`、`app/file/file.proto`。

## 入口与 ServiceContext

```go
type ServiceContext struct {
    Config              config.Config
    DB                  *gormx.DB
    Validate            *validator.Validate
    ThumbTaskRunner     *threading.TaskRunner
    Httpc               httpc.Service
    NetClient           *netx.Client
    OssTemplateResolver ossx.TemplateResolver
}
```

- `DB`：通过 `gormx.MustOpenWithConf` 打开，Dev/Test 模式自动 migrate `Oss` 表。
- `OssTemplateResolver`：通过 `ossx.NewTemplateResolver(tenantMode, loadOssConfig)` 创建，支持 DI 注入，`loadOssConfig` 按 tenant_id + oss_code 查 DB。
- `ThumbTaskRunner`：异步缩略图生成，并发数由 `ThumbTaskConcurrency` 配置（默认 2）。
- `NetClient`：通过 `netx.NewClient(netx.WithEngine(netx.NewHTTPEngine(httpcSvc)))` 创建，提供可复用的 HTTP 客户端。

依据：`app/file/internal/svc/servicecontext.go`。

## OSS 配置管理

`oss` 表字段定义（`model/gormmodel/oss.go`）：

```go
type Oss struct {
    gormx.LegacyStringBaseModel  // Id (UUID), CreateTime, UpdateTime, DeleteTime
    gormx.VersionMixin           // Version (乐观锁)
    TenantId   string            // uniqueIndex: uk_oss_tenant_id_oss_code
    Category   int               // 1=Minio 2=Qiniu 3=Ali 4=Tencent
    OssCode    string
    Endpoint, AccessKey, SecretKey, BucketName, AppId, Region, Remark string
    Status     int               // 1=open 2=closed
}
```

- Unique 约束：`(tenant_id, oss_code)`，不同租户可使用相同 oss_code。
- 更新/删除 OSS 配置后必须调用 `ossx.CacheInvalidate(tenantId)` 清除模板缓存。

依据：`model/gormmodel/oss.go`、`app/file/internal/logic/updateosslogic.go`。

## OssTemplate 模板池

`common/ossx/ossx.go` 中的 `Template()` 实现双重检查锁定（DCL）缓存：

```go
func Template(ctx context.Context, TenantId, Code string, tenantMode bool, getConfig GetConfigFn) (OssTemplate, error) {
    // 1. RLock: 读取缓存
    // 2. 缓存有效 → 直接返回
    // 3. 缓存无效 → Lock → 二次检查 → 调用 getConfig + NewTemplate → 写入缓存
}
```

- 缓存 key 为 `tenantId`。
- 重建条件：Endpoint、AccessKey、SecretKey 任意变化。
- `CacheInvalidate(tenantId)` 直接删除缓存项。
- `NewTemplate` 按 `Category` 创建对应实现（当前仅 Minio）。

新增 OSS 后端：
1. 实现 `OssTemplate` 接口全部方法。
2. 在 `NewTemplate` 的 switch 中添加分支。
3. 确保 `needRebuild` 的比较逻辑适配新后端的配置差异。

依据：`common/ossx/ossx.go`、`common/ossx/minio_oss.go`。

## 租户模式

`OssRule.fullBucketName(tenantId, bucketName)`:
- `tenantMode=true`：`{tenantId}-{bucketName}`
- `tenantMode=false`：`{bucketName}`

文件路径生成 `OssRule.filename(originalFilename, pathPrefix...)`:
- 第一个 prefix 参数：替换默认 "upload" 目录
- 第二个 prefix 参数：使用固定文件名
- 最终格式：`{prefix}/YYYYMMDD/{uuid}.{ext}`

依据：`common/ossx/ossx.go`。

## 文件操作约定

### SignUrl

- 调用前使用 `validator.Validate.Struct` 校验请求字段。
- `expires` 从请求的 `Expires` 参数计算，有上限值（`calcExpires` 在 `helper.go`）。

### PutFile

- 从本地文件路径上传，使用 `ossx.UploadWithMD5` 获取 MD5。
- 通过 `OssTemplateResolver` 获取 template → 调用 `PutObject`。

### PutStreamFile（流式上传）

- gRPC 流式接口，用 `stream.Context()` 构建 context。
- 内部调用 `ossx.UploadStream`，使用 `filex.Capture` 同时完成：
  1. 捕获头部字节（MIME 检测）
  2. 可选 tee 到临时文件（缩略图源）
  3. 流式上传到 OSS

```go
// stream_upload_helper.go 中的处理链
func resolveContentType(builder *ContentTypeBuilder)    // 调用方指定 > 头部检测
func buildCaptureOptions(tempDir, pattern...)            // 构建 Capture 选项
func processUploadResult(file, headContent...)           // 提取 ContentType + Size
func scheduleThumbGeneration(runner, file, options...)   // 投递到 TaskRunner
```

### RelayFile（中继上传）

- 先下载源文件 → 并发上传到多个目标 OSS。
- 使用 `antsx.Reactor` goroutine 池，并发数由 `RelayUploadConcurrency` 配置（默认 4）。
- 单目标失败不阻塞其他目标。
- 文件大小限制 200MB。
- 源文件写入 temp 目录，`KeepTempFiles=false` 时上传后清理。

### CaptureVideoStream

- 从视频流抓取帧 → 保存为图片 → 上传到 OSS。
- 使用 `ffmpeg` 命令行工具（需部署环境具备）。

依据：`app/file/internal/logic/putstreamfilelogic.go`、`app/file/internal/logic/relayfilelogic.go`、`app/file/internal/logic/stream_upload_helper.go`。

## common/filex — 文件工具

- **`Capture`**：Tee 读取器，同时捕获头部字节和可选写入临时文件。
- **`HeadCaptureWriter`**：捕获前 `MaxHeadRead` 字节，用于 `http.DetectContentType`。
- **`MD5Digest` / `NewMD5TeeReader`**：流式 MD5 计算。
- **`CaptureOptions`**：TempDir、TempPattern、NeedTemp、MaxHeadRead。

使用 `Capture` 后，调用方必须：
1. 检查 `CaptureOptions.NeedTemp` 决定是否处理临时文件。
2. 若 `NeedTemp=true` 且不再需要，调用 `RemoveTempFile` 清理。
3. 不可在 `Capture` 返回的 `io.Reader` 被完全读取前访问 `HeadCaptureWriter.Bytes()`。

依据：`common/filex/filex.go`。

## 异步缩略图

- 通过 `threading.TaskRunner.Schedule` 投递到 `ThumbTaskRunner`。
- 使用 `context.WithoutCancel(ctx)` 防止主请求取消影响后台任务。
- 缩略图尺寸由 `Image.Thumb.Width/Height` 配置，开关由 `Image.Thumb.Enabled` 控制。

## Scenario: OssTemplate 缓存与失效

### 1. Scope / Trigger

- 修改 `ossx.Template`、`ossx.CacheInvalidate`、或 app/file 的 CreateOss/UpdateOss/DeleteOss Logic 时适用。

### 2. Signatures

```go
func Template(ctx context.Context, TenantId, Code string, tenantMode bool, getConfig GetConfigFn) (OssTemplate, error)
func CacheInvalidate(tenantId string)
```

### 3. Contracts

- `Template` 在 `RLock` 内读取缓存；缓存有效时直接返回，不经 `getConfig`。
- 缓存无效时先获取写锁，在锁内二次检查（DCL），防止重复创建。
- `getConfig` 仅在没有有效缓存时调用；若 `getConfig` 返回 error，不写入缓存。
- 缓存重建条件：Endpoint、AccessKey、SecretKey 中任意变化。其他字段变更（如 Region、AppId）当前不触发重建。
- `CacheInvalidate` 必须持有写锁，删除 `templatePool` 和 `ossPool` 两个 map 中的条目。
- 调用方 CreateOss/UpdateOss/DeleteOss 在 DB 操作成功后必须调用 `CacheInvalidate`；写 DB 成功但清理缓存失败会导致永久脏缓存，应视为错误。

### 4. Validation & Error Matrix

- `getConfig` 返回 nil config 且无 error → DCL 内以 nil config 做 `needRebuild` 对比仍为 true，触发重建；重建时的 nil config 场景由 `NewTemplate` 处理。
- 并发两个请求无缓存时 → DCL 确保只创建一个 template。
- `CacheInvalidate` 后并发 Template 调用 → 下次调用重新加载 config。
- 更新 OSS 配置后未调用 `CacheInvalidate` → 缓存不更新，直到缓存因 key 对比变化自然失效或服务重启。

### 5. Good/Base/Bad Cases

- Good: CreateOss → DB 写入成功 → `CacheInvalidate` 删除缓存 → 下次 PutFile 正常加载新 template → 使用新配置上传。
- Base: 无 OSS 变更时直接上传 → 缓存命中 → 不查 DB。
- Bad: UpdateOss 只写 DB 不调 `CacheInvalidate` → 旧 template 仍在使用 → 上传到错误的 bucket。

### 6. Tests Required

- 断言首次 Template 调用查 DB 并缓存。
- 断言同 tenantId 二次 Template 调用不查 DB。
- 断言配置变更后需重建缓存。
- 断言并发 Template 调用只查一次 DB。
- 断言 `CacheInvalidate` 后 Template 重新查 DB。
- 对 `common/ossx` 运行 race test。

### 7. Wrong vs Correct

#### Wrong

```go
func (l *UpdateOssLogic) UpdateOss(in *file.UpdateOssReq) (*file.UpdateOssRes, error) {
    err := l.svcCtx.DB.Save(&ossModel).Error
    if err != nil { return nil, err }
    // OSS 配置更新成功，但未清除缓存
    return &file.UpdateOssRes{}, nil
}
```

#### Correct

```go
func (l *UpdateOssLogic) UpdateOss(in *file.UpdateOssReq) (*file.UpdateOssRes, error) {
    err := l.svcCtx.DB.Save(&ossModel).Error
    if err != nil { return nil, err }
    ossx.CacheInvalidate(in.TenantId) // 必须在 DB 写入成功后立即清除
    return &file.UpdateOssRes{}, nil
}
```

依据：`common/ossx/ossx.go`、`app/file/internal/logic/updateosslogic.go`、`app/file/internal/logic/createosslogic.go`、`app/file/internal/logic/deleteosslogic.go`。

## 反模式

- 修改 OSS 配置后不调用 `CacheInvalidate`。
- 在 Logic 中直接调用 `ossx.NewTemplate` 绕过 template pool 缓存。
- 在 `Capture` reader 读完前访问 `HeadCaptureWriter.Bytes()`。
- 在 RelayFile 中使用 `errgroup` 但未限制并发数。
- 在 PutStreamFile 逻辑中使用 `ctx` 而非 `stream.Context()`。
- 在 `ossx.Template` 外维护独立的 OssTemplate 缓存。
- 在多租户模式下忘记应用 `fullBucketName` 前缀。

## 验证

- Proto 变更执行 `app/file/gen.sh` 并测试所有直接调用方。
- OSS CRUD 测试覆盖创建/更新/删除后的缓存失效。
- PutFile/PutStreamFile 测试覆盖 ContentType 检测与 MD5 计算。
- RelayFile 测试覆盖单目标失败不阻塞其他目标、文件大小限制。
- 运行 `go test -race ./app/file/internal/logic ./common/ossx ./common/filex`。
