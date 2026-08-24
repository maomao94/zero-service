# Research: Oryx Recording artifact_code Error Codes

- **Query**: Oryx 录制场景中 artifact_code 不为 0 的情况
- **Scope**: Mixed (internal source code + external documentation)
- **Date**: 2026-08-25

## Summary

Oryx 的 `on_record_end` 回调中 `artifact_code` 字段表示录制结果的错误码。当前实现中，只有两种取值：`0`（成功）和 `100`（录制仍在处理中）。

---

## Source Code Analysis

### 1. artifact_code 赋值逻辑

**文件**: `platform/callback.go` - `OnRecordMessage` 方法

```go
if action == SrsActionOnRecordEnd {
    code := 0
    if artifact.Processing {
        code = int(SrsStackErrorCallbackRecord)
    }
    req.ArtifactCode = &code
    req.ArtifactPath = fmt.Sprintf("%v/record/%v/index.mp4", serverDataDirectory, artifact.UUID)
    req.ArtifactURL = fmt.Sprintf("%v/terraform/v1/hooks/record/hls/%v/index.mp4", config.Host, artifact.UUID)
}
```

**关键逻辑**:
- 默认 `code = 0`
- 当 `artifact.Processing == true` 时，设置 `code = 100`

### 2. 错误码定义

**文件**: `platform/srs-errors.go`

```go
// General error code for Oryx, it uses 100~999 for all errors.
type SrsStackError int

// Error code for callback, 100 ~ 200.
const (
    // Error for callback module, about the record events.
    SrsStackErrorCallbackRecord SrsStackError = 100
)
```

### 3. artifact_code 取值清单

| 值 | 常量名 | 含义 | 触发条件 |
|---|---|---|---|
| `0` | - | 录制成功完成 | `artifact.Processing == false`，MP4 文件已生成完毕 |
| `100` | `SrsStackErrorCallbackRecord` | 录制仍在处理中 | `artifact.Processing == true`，HLS 切片尚未转换为 MP4 |

---

## artifact_code = 100 的触发场景

当 `artifact.Processing == true` 时，表示录制的 MP4 文件尚未完成处理。这通常发生在以下情况：

### 流中断场景
- 推流端意外断开（网络故障、推流软件崩溃等）
- SRS 服务器检测到推流结束后，立即触发 `on_record_end` 回调
- 此时 HLS 临时切片文件已收集，但 FFmpeg 尚未完成将切片合并为 MP4 的操作

### 正常结束但处理未完成
- 推流端正常停止推流
- SRS 触发 `on_record_end` 回调
- MP4 后处理（合并 HLS 切片）仍在进行中

---

## Oryx 文档说明

**来源**: https://ossrs.net/lts/en-us/docs/v7/doc/getting-started-oryx

文档中关于 `on_record_end` 回调的说明：

```
Request:
{
  "request_id": "d13a0e60-e2fe-42cd-a8d8-f04c7e71b5f5",
  "action": "on_record_end",
  "opaque": "mytoken",
  "vhost": "__defaultVhost__",
  "app": "live",
  "stream": "livestream",
  "uuid": "824b96f9-8d51-4046-ba1e-a9aec7d57c95",
  "artifact_code": 0,
  "artifact_path": "/data/record/824b96f9-8d51-4046-ba1e-a9aec7d57c95/index.mp4",
  "artifact_url": "http://localhost/terraform/v1/hooks/record/hls/824b96f9-8d51-4046-ba1e-a9aec7d57c95/index.mp4"
}
```

文档说明：
- `uuid` 是录制任务的 UUID
- `artifact_code` 表示错误码，如果没有错误则为 0
- `artifact_path` 是录制 MP4 文件在容器中的路径
- `artifact_url` 是访问录制 MP4 文件的 URL 路径
- 忽略任何响应错误（Ignore any response error）

---

## SRS 内部错误码（参考）

SRS 服务器本身定义了丰富的错误码体系（`srs_kernel_error.hpp`），但这些错误码不会直接传递给 Oryx 的 `artifact_code`。SRS 的录制相关错误码包括：

### HLS 相关错误码（3000-3100 范围）
| 错误码 | 名称 | 含义 |
|---|---|---|
| 3000 | `HlsMetadata` | HLS metadata 无效 |
| 3001 | `HlsDecode` | HLS 解码音视频流失败 |
| 3002 | `HlsCreateDir` | HLS 创建目录失败 |
| 3003 | `HlsOpenFile` | HLS 打开 m3u8 文件失败 |
| 3004 | `HlsWriteFile` | HLS 写入 m3u8 文件失败 |
| 3062 | `HlsNoStream` | 没有配置 HLS 流 |

### 文件系统相关错误码（1000-1100 范围）
| 错误码 | 名称 | 含义 |
|---|---|---|
| 1041 | `FileOpened` | 文件已打开 |
| 1042 | `FileOpen` | 打开文件失败 |
| 1043 | `FileClose` | 关闭文件失败 |
| 1044 | `FileRead` | 读取文件失败 |
| 1045 | `FileWrite` | 写入文件失败（磁盘满、权限问题等） |
| 1046 | `FileEof` | 文件到达末尾 |
| 1047 | `FileRename` | 重命名文件失败 |
| 1056 | `DirExists` | 目录已存在 |
| 1057 | `DirCreate` | 创建目录失败 |
| 1064 | `FileNotFound` | 请求的文件不存在 |
| 1067 | `FileRemove` | 删除或解除链接文件失败 |
| 1068 | `FileRename` | 重命名文件段失败 |
| 1095 | `FileNotOpen` | 文件未打开 |
| 1101 | `FileUnlink` | 解除文件链接失败 |

### DVR 相关错误码
| 错误码 | 名称 | 含义 |
|---|---|---|
| 3050 | `DvrDisabled` | DVR 被禁用 |
| 3051 | `DvrRequest` | DVR 请求失败 |
| 3053 | `DvrCreate` | DVR 创建请求失败 |
| 3054 | `DvrNoTarget` | DVR 没有目标 |
| 3082 | `DvrAppend` | DVR 追加数据到文件失败 |
| 3083 | `DvrPlan` | DVR 计划无效 |

### MP4 相关错误码
| 错误码 | 名称 | 含义 |
|---|---|---|
| 3069 | `Mp4BoxOverflow` | MP4 box 仅支持 32 位 |
| 3070 | `Mp4BoxNoSpace` | 解码 MP4 box 时缓冲区空间不足 |
| 3071 | `Mp4BoxType` | MP4 box 类型无效 |
| 3072 | `Mp4BoxNoFtyp` | MP4 缺少 FTYP box |
| 3076 | `Mp4BoxMoov` | MP4 MOOV box 无效 |
| 3085 | `Mp4AvccChange` | MP4 不支持视频 AVCC 变化 |
| 3086 | `Mp4AscChange` | MP4 不支持音频 ASC 变化 |

### Stream Caster（TS 相关）错误码
| 错误码 | 名称 | 含义 |
|---|---|---|
| 4012 | `CasterTsHeader` | TS header 无效 |
| 4013 | `CasterTsSync` | TS 同步字节无效 |
| 4014 | `CasterTsAdaption` | TS 自适应字段无效 |
| 4016 | `CasterTsPsi` | TS PSI 载荷无效 |
| 4017 | `CasterTsPat` | TS PAT 程序无效 |
| 4018 | `CasterTsPmt` | TS PMT 信息无效 |
| 4019 | `CasterTsPse` | TS PSE 载荷无效 |
| 4020 | `CasterTsEs` | TS ES 流无效 |

**重要**: 这些 SRS 内部错误码存在于 SRS C++ 代码中，但 Oryx 的 `on_record_end` 回调的 `artifact_code` 并不直接传递这些错误码。Oryx 层面仅使用 `0` 和 `100` 两个值。

---

## 实际可能的录制失败原因（不通过 artifact_code 传递）

虽然 `artifact_code` 只有 0 和 100 两个值，但以下情况会导致录制失败或 MP4 文件异常：

### 1. 磁盘空间不足
- SRS 写入 HLS TS 切片失败
- FFmpeg 转换 MP4 时失败
- 日志中会出现 `FileWrite` 相关错误

### 2. 权限问题
- Docker 容器内的录制目录没有写权限
- `/data/record` 目录权限不正确

### 3. 流中断导致的录制不完整
- 推流端异常断开
- `artifact_code = 100`，MP4 文件可能已生成但内容不完整

### 4. FFmpeg 转换失败
- TS 切片数据损坏
- 编码格式不兼容
- FFmpeg 进程异常退出

### 5. 内存不足
- 大量并发录制任务
- Redis 内存溢出（参考 Issue #197）

---

## 对 zero-service 的启示

在处理 Oryx 的 `on_record_end` 回调时：

1. **检查 `artifact_code`**: 如果不为 0，表示录制可能未正常完成
2. **检查 `artifact_path` 和 `artifact_url`**: 确认 MP4 文件是否存在且可访问
3. **`artifact_code = 100` 时**: MP4 文件可能仍在生成中，需要延迟检查或轮询确认
4. **忽略响应错误**: 根据 Oryx 文档，回调响应中的错误应被忽略

---

## Caveats / Not Found

1. **M3u8VoDArtifact.Processing 字段的完整赋值逻辑**: 由于 `platform/objs` 是符号链接到 `platform/containers/objs`，无法直接查看 `M3u8VoDArtifact` 结构体的定义和 `Processing` 字段的赋值逻辑。需要在 Oryx 源码仓库中进一步查找。

2. **是否有更多错误码**: 当前源码中只定义了 `SrsStackErrorCallbackRecord = 100`，但 `SrsStackError` 类型的范围是 100~999，可能存在其他未定义的错误码。

3. **SRS 服务器内部错误是否会传递到 artifact_code**: 需要进一步确认 SRS 内部的录制错误是否会通过某种机制映射到 Oryx 的 `artifact_code`。
