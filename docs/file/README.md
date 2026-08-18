# File 文件与对象存储服务

`file.rpc` 是文件与对象存储服务（默认端口 21003），提供 OSS 配置管理、文件上传/签名/中继和视频截帧能力。当前对象存储后端实现为 MinIO。

## 服务职责

- **OSS 配置管理**：以租户维度维护对象存储配置（`oss` 表），支持创建、更新、删除、列表和详情查询；`(tenant_id, oss_code)` 唯一，同一租户可配置多个存储资源。
- **文件上传**：本地路径上传（`PutFile`）与 gRPC 流式上传（`PutStreamFile`）；流式上传自动探测 ContentType（调用方指定优先），并可选异步生成缩略图。
- **中继上传**：从 OSS URL 或本地路径拉取源文件，并发中继到多个目标 OSS（单目标失败不阻塞其他目标，源文件上限 200MB）。
- **签名与查询**：生成带过期时间的签名 URL、获取文件信息、删除/批量删除文件。
- **视频截帧**：从视频流抓取帧保存为图片并上传（依赖 `ffmpeg` 命令）。
- **图片缩略图**：图片上传成功后通过异步任务生成缩略图，返回缩略图地址与文件名（不替换原图）。

## 配置

配置文件：`app/file/etc/file.yaml`。关键项：

| 配置项 | 说明 | 默认值 |
| --- | --- | --- |
| `ListenOn` | gRPC 监听地址 | `0.0.0.0:21003` |
| `Timeout` | 单次调用上限（毫秒），RelayFile 等长链路需留足余量 | `600000` |
| `DB.DataSource` | 数据库连接（gormx 按 DataSource 自动识别 MySQL/PostgreSQL/SQLite） | 必填 |
| `Oss.TenantMode` | 是否启用多租户模式，启用后桶名加 `{tenantId}-` 前缀 | `true` |
| `ThumbTaskConcurrency` | 缩略图异步任务并发数 | `2` |
| `Upload.TempDir` | 上传临时目录 | `/opt/data/temp` |
| `Upload.KeepTempFiles` | 是否保留临时文件（仅排障时开启） | `false` |
| `Upload.RelayUploadConcurrency` | RelayFile 多目标并发数 | `4` |
| `Upload.Image.MaxExifRead` | 图片 EXIF 解析缓存的文件头字节数 | `65536` |
| `Upload.Image.Thumb.Enabled / Width / Height` | 缩略图开关与尺寸 | `false / 300 / 300` |
| `NacosConfig.IsRegister` | 是否注册到 Nacos（可选） | `false` |

## 关键接口

完整 RPC 定义见 [`app/file/file.proto`](../../app/file/file.proto)（`service FileRpc`），字段与校验以 proto 为权威。

| 分组 | RPC | 说明 |
| --- | --- | --- |
| OSS 配置管理 | `OssDetail` / `OssList` / `CreateOss` / `UpdateOss` / `DeleteOss` | 存储配置的增删改查；更新/删除后自动失效模板缓存 |
| 存储桶 | `MakeBucket` / `RemoveBucket` | 创建/删除存储桶 |
| 文件上传 | `PutFile` / `PutStreamFile` / `RelayFile` | 本地路径上传、流式上传、多目标中继上传 |
| 文件查询 | `StatFile` / `SignUrl` | 文件信息与签名 URL |
| 文件删除 | `RemoveFile` / `RemoveFiles` | 删除/批量删除文件 |
| 视频截帧 | `CaptureVideoStream` | 视频流抓帧上传为图片 |

## 路径约定

- 文件路径格式：`{pathPrefix}/YYYYMMDD/{uuid}.{ext}`；`pathPrefix` 默认 `upload`（`PutFile` 默认 `default`）。
- 多租户模式下完整桶名为 `{tenantId}-{bucketName}`。

## 部署

- 标准 go-zero zRPC 服务，启动方式：

```bash
./file -f etc/file.yaml
```
- 需要可访问的数据库（`oss` 表存储配置）与对象存储服务（MinIO）。
- `CaptureVideoStream` 需要部署环境具备 `ffmpeg` 命令。
- 日志默认输出到 `/opt/logs/file.rpc`。

## 权威契约

- RPC 契约：[`app/file/file.proto`](../../app/file/file.proto)
- 服务配置：`app/file/etc/file.yaml`
- 公共库：[`common/ossx`](../../common/ossx/ossx.go)（模板缓存与 MinIO 实现）、[`common/filex`](../../common/filex/filex.go)（流式捕获与 MD5）
