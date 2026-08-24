# Research: Oryx HTTP OpenAPI 各接口请求字段核对（vs zero-service oryxserver SDK/proto）

- **Query**: 对照 Docker SRS Platform（现 ossrs/oryx）源码，核对 zero-service common/oryxx SDK + app/oryxserver proto 的 13 个 proxy 接口的请求字段是否齐全，找出缺少/被忽略的请求参数
- **Scope**: external（Oryx 上游源码）+ internal（zero-service SDK/proto）
- **Date**: 2026-08-25

## 源码定位

- GitHub 无法直接 clone（网络阻断），改用 raw.githubusercontent.com 验证。
- **仓库已改名**：`ossrs/docker-srs-platform` → **`ossrs/oryx`**（SRS Stack → Oryx）。平台 Go 源码在 `platform/` 目录（`main.go`、`service.go`、`callback.go`、`dvr-local-disk.go`、`dvr-tencent-cos.go`、`dvr-tencent-vod.go`、`srs-hooks.go`、`utils.go`、`forward.go`、`version.go` 等）。
- 本地已有提取副本：`/var/folders/0g/rl3htjrs1jdd9p9jb0sfz9fc0000gn/T/opencode/oryx-src/*.go`（上一会话产物，扁平化命名：`dvr-local-disk.go`=record，`dvr-tencent-cos.go`=dvr，`callback.go`=hooks，`service.go`=versions/mgmt）。
- **校验**：webfetch `raw.githubusercontent.com/ossrs/oryx/main/platform/{dvr-local-disk.go, callback.go}` 与本地提取副本逐行一致 → 本地提取副本 == 上游 main（Oryx 6.0 线，PRD 约定）。
- 目录清单（API contents，platform 下）：确认无 `record.go`/`dvr.go`/`hooks.go`/`routes.go`/`versions.go` 旧名文件；任务提示中的旧文件名（record.go/dvr.go/hooks.go/routes.go/versions.go）为 SRS Stack v5 之前的旧布局，**v6 main 不存在**这些文件。

## 关键机制（先读这些，再看字段表）

### 鉴权（utils.go）
- `envApiSecret()` 读 SRS_PLATFORM_SECRET 环境变量（platform/utils.go:366）。
- `Authenticate(ctx, apiSecret, token, header)`（utils.go:1337-1379）：
  - `apiSecret==""` → `errors.New("no api secret")`；
  - `Authorization: Bearer <secret>`（header）与 body 字段 `token`（JWT，用 apiSecret 签名）**二选一**；两者都缺 → `no Authorization or token`。
  - zero-service SDK（common/oryxx/oryx.go:38-41）用 `Authorization: Bearer Secret` → **合法**，无需发 body `token`。
- `ParseBody`（utils.go:1316-1332）：空 body 直接返回 nil（`len(b)==0` 不报错）→ SDK 对 query/files 类接口传 nil body 合法。
- 鉴权要求：**除 `/terraform/v1/mgmt/versions` 外，所有 13 个接口都要鉴权**（版本接口无鉴权，service.go:450-460 无 Authenticate 调用）。

## 每个接口的完整字段表

### 1. GET /terraform/v1/mgmt/versions（service.go:450-460）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| （无 body，无 query） | - | - | - | - |

- 响应：`{"version":"6.0.x"}`（`strings.TrimPrefix(version,"v")`，version 为编译时版本）。
- SDK `Versions()` GET + 解析 version → **齐全**。

### 2. POST /terraform/v1/hooks/record/query（dvr-local-disk.go:52-106）
| 字段 | 位置 | 必填 | 类型 | 默认/说明 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权二选一；*也可走 Bearer header，下同 |
| all / home / globs / processCpDir | 响应字段 | - | bool/string/[]string/string | **非请求字段！** 从 Redis 读后返回：`all`（"true"==true）、`home`（硬编码 `"/data/record"`，line 96）、`globs`（JSON 数组）、`processCpDir` |
| progress | - | - | - | 不存在于此接口（progress 是 record/files 的响应字段） |

- 请求体只有 `token`。**无 host/app/stream/vhost/expired 等过滤字段**。
- SDK：`RecordQuery()` body=nil + header 鉴权 → 合法；`RecordQueryData{All,Home,Globs,ProcessCpDir}` 与响应结构一致 → **齐全**。

### 3. POST /terraform/v1/hooks/record/apply（dvr-local-disk.go:108-138）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| all | body | 是（语义上） | bool | 缺省 false；写入 Redis SRS_RECORD_PATTERNS/all |

- **无 `?action=start/stop`**，也不读任何 query 参数（事件函数只 `ParseBody`）。
- SDK：`RecordApply(all bool)` 发送 `{"all":...}` → **齐全**。

### 4. POST /terraform/v1/hooks/record/end（dvr-local-disk.go:294-331）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| uuid | body | **是** | string | 空 → `no uuid`（line 313-315）；查不到活动任务 → `no record task for uuid=...`（line 317-320） |

- 没有 `?action=`；动作是 `task.Expired = true`（line 323），内部 worker 快速收尾。
- SDK：`RecordEnd(uuid)` 发送 `{"uuid":...}` → **齐全**。注意：仅对"录制中"的任务有效，已结束任务会报错。

### 5. POST /terraform/v1/hooks/record/remove（dvr-local-disk.go:224-292）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| uuid | body | **是** | string | 空 → `no uuid`；无该 uuid → `no record for uuid=...`（line 243-254） |

- 无 action/query 参数。删除 ts/m3u8/mp4 及 Redis 记录。
- SDK：`RecordRemove(uuid)` → **齐全**。

### 6. POST /terraform/v1/hooks/record/files（dvr-local-disk.go:333-389）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |

- **无任何其他请求字段/url 参数**。单次 HScan（count=100，只取一次 cursor，理论上大于 100 条会截断——响应侧注意点，非请求字段）。
- 响应数组元素（line 370-380）：`uuid/vhost/app/stream/progress/update/nn/duration/size`（progress=metadata.Processing，更新为 RFC3339）。
- SDK：`RecordFiles()` body=nil；`RecordFile` 九字段与响应一致 → **齐全**。

### 7. POST /terraform/v1/hooks/record/globs（dvr-local-disk.go:140-179）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| globs | body | 否 | []string | 空串被过滤掉（line 160-165）；结果为 [] 时 `SRS_RECORD_PATTERNS/globs="[]"` → 录制所有流 |

- 无 uuid/其他字段。SDK `RecordGlobs(globs)` 发送 `{"globs":[...]}` → **齐全**。

### 8. POST /terraform/v1/hooks/record/post-processing（dvr-local-disk.go:181-222）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| postProcess | body | **是（必须）** | string | **只接受 `"post-cp-file"`**，否则 `invalid post process %v`（line 202-204） |
| postCpDir | body | 否 | string | 非空则必须为本地已存在目录（os.Stat，line 205-209）；空 = 取消后处理 |

- 无 uuid。SDK `RecordPostProcessing(postProcess, postCpDir)` → **齐全**（注意 postProcess 必须传 `"post-cp-file"`）。

### 9. POST /terraform/v1/hooks/dvr/query（dvr-tencent-cos.go:58-98）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |

- 响应：`{all: bool, secret: bool}`（secret=腾讯云 appId/secretId/secretKey 均已配置，line 85-91）。
- SDK `DvrQuery()` + `DvrQueryData{All,Secret}` → **齐全**。

### 10. POST /terraform/v1/hooks/dvr/apply（dvr-tencent-cos.go:100-130）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| all | body | 是（语义上） | bool | 缺省 false；写入 SRS_DVR_PATTERNS/all |

- 无 action/query。SDK `DvrApply(all)` → **齐全**。

### 11. POST /terraform/v1/hooks/dvr/files（dvr-tencent-cos.go:132-191）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |

- 响应数组元素（line 169-182）：`uuid/vhost/app/stream/progress/update/nn/duration/size/bucket/region`（bucket/region 仅 DVR）。
- SDK `DvrFiles()` + `DvrFile` 11 字段 → **齐全**。

### 12. POST /terraform/v1/mgmt/hooks/query（callback.go:50-99）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| all | body | **被声明但被忽略** | bool | 解析结构里有 `All *bool`（line 57）但赋值时只给了 `Token`（line 59），**发送 `all` 无任何效果** |

- 响应（line 84-93）：`{req, res, target, opaque, all, host}`（req/res=最近一次回调的请求/响应原文，来自 Redis SRS_HOOKS/req|res）。
- SDK `HooksQuery()` body=nil；`HooksQueryData{Req,Res,Target,Opaque,All,Host}` → 字段定义与响应一致（`req/res/target/opaque/all/host` 全有）；SDK 未发 `all` 也无所谓（Oryx 反正忽略）。

### 13. POST /terraform/v1/mgmt/hooks/apply（callback.go:101-156）
| 字段 | 位置 | 必填 | 类型 | 默认 |
|---|---|---|---|---|
| token | body | 否* | string | 鉴权 |
| target | body | 是（语义上） | string | 空则写了空串（回调 worker 会因 `config.Target==""` 不回调） |
| opaque | body | 否 | string | 回调透传凭证 |
| all | body | 否 | bool | 缺省 false（仅 all=true 才回调） |
| host | body | 否 | string | **空时默认 `http://` + r.Host（HTTPS 时 `https://`），见 line 133-138** |

- 无 action/query 参数。SDK `HooksApply(HooksApplyReq{Target,Opaque,All,Host})` → **齐全**。

## 三个任务提示中"疑似缺字段"的核对结论

| 提示字段 | 结论 | 证据 |
|---|---|---|
| `app` / `stream` / `vhost` | **不是**这 13 个接口的请求字段。只出现在：record/files、dvr/files 的**响应**对象（读 artifact 元数据）；on_publish/on_record_* 等**回调**（callback.go:261-282、396-409）。不存在"按 app/stream 过滤查询"的请求参数 | dvr-local-disk.go:370-380；dvr-tencent-cos.go:169-182；callback.go OnStreamMessage/OnRecordMessage |
| `req`（request 对象） | 是 hooks/query 的**响应**字段 `req`/`res`（最近一次回调原文），SDK 已有（HooksQueryData.Req/Res） | callback.go:74-82、84-93 |
| `host` | hooks/apply 可选入参（空默认取请求 Host）+ hooks/query 出参，SDK 已有 | callback.go:132-141 |
| `expired` | **不是 API 字段**。是内部 `RecordM3u8Stream.Expired` 状态（record/end 置 true 加速收尾；过期判断在 `expired()`），不出现在任何请求/响应 JSON | dvr-local-disk.go:316-323、RecordM3u8Stream struct 与 `expired()` |
| `progress` | record/files、dvr/files **响应**字段（=artifact.Processing），SDK 已有（RecordFile.Progress / DvrFile.Progress） | dvr-local-disk.go:375；dvr-tencent-cos.go:174 |
| `?action=start/stop` | **不存在**。13 个接口均不读 query 参数（全文件 grep：`r.URL.Query()` 仅出现在 service.go /rtc proxy `?eip=`、/api proxy `?token=`，callback.go hooks/example `?fail=true`，均非本次接口）。Oryx 录制是"pattern 驱动"（apply 写 Redis all/globs），按流启停通过 on_record_begin/end 回调 + record/end | grep 结果；dvr-local-disk.go 全文 |

## 现有 SDK/proto 与 Oryx 的差异清单（核心结论）

**请求侧：无缺失字段。** 每个接口 Oryx 实际读取的请求参数（token/all/globs/postProcess/postCpDir/uuid/target/opaque/all/host）在 `common/oryxx/oryx.go` 都已发送，且 proto（app/oryxserver/oryxserver.proto）逐接口字段一致：

- VersionsReq{} ✓（GET 无参）；RecordQueryReq{} ✓；RecordApplyReq{all} ✓；RecordEndReq{uuid} ✓；RecordRemoveReq{uuid} ✓；RecordFilesReq{} ✓；RecordGlobsReq{globs} ✓；RecordPostProcessingReq{post_process,post_cp_dir} ✓；DvrQueryReq{} ✓；DvrApplyReq{all} ✓；DvrFilesReq{} ✓；HooksApplyReq{target,opaque,all,host} ✓；HooksQueryReq{} ✓。
- 唯一"被忽略"的字段：hooks/query 的 body `all` —— **Oryx 声明但忽略**（callback.go:57-59），SDK 未发，无需处理。

**需要注意的点（非字段缺失，但影响兼容性）：**
1. **鉴权**：SDK 走 `Authorization: Bearer <Secret>` 路径；Oryx 也支持 body `token`（JWT）。SDK 的 `Secret` 必须等于 Oryx 的 `SRS_PLATFORM_SECRET`，否则 `invalid bearer token`（utils.go:1365-1367）。mgmt/versions 无需鉴权（SDK 也会带 header，无影响）。
2. **record/post-processing**：`postProcess` 必须传字符串常量 `"post-cp-file"`；`postCpDir` 非空时必须为 Oryx 容器内已存在目录（SDK/proto 注释可补充）。
3. **record/end 与 record/remove**：uuid 必填且必须存在；record/end 只对"录制中"任务有效（查不到活动 task 报错）——SDK 调用方需容忍 OryxError。
4. **record/files / dvr/files**：只有 token 入参；响应为数组（data 为数组而非对象），SDK call() 解析数组正确。
5. 响应字段与 proto 一一对应：RecordFile 9 字段 / DvrFile 11 字段（extra bucket/region）/ HooksQuery 6 字段 / RecordQuery 4 字段 / DvrQuery 2 字段，全部对齐。

## 附带：Oryx 回调（on_record_begin/on_record_end）请求字段（或yxgtw 侧参考，callback.go:393-433）

- on_record_begin：`request_id / action / opaque / vhost / app / stream / uuid`（uuid=录制任务 UUID）
- on_record_end：上述 + `artifact_code(int, 0=成功) / artifact_path / artifact_url`（artifact_url 用 config.Host 拼接 `/terraform/v1/hooks/record/hls/<uuid>/index.mp4`）
- 与 oryxgtw 现有实现（.opencode/plans/oryx-hook-proxy.md 表格）一致。

## Caveats / 未确认项

- 未能在本地验证部署镜像的具体 Oryx 版本（未找到 docker-compose/k8s 清单引用 ossrs/oryx 镜像标签）；本文以 PRD 的 "6.0 (Stable)" 即 **ossrs/oryx main** 为准，且已与 upstream main 逐文件核对（记录/回调 handler 完全一致）。若部署的是 v5 线（release/5.x 分支，docker-srs-platform→oryx），上述 record/dvr/hooks/versions 端点与字段自 5.7+ 起未变（同样文件、同样 handler 结构，版权 2022-2024），但未逐分支 diff 验证。
- go.mod 无 go-oryx-lib/oryx 直接依赖（SDK 为手写 httpc），因此"依赖版本"线索不适用；已用 GitHub 原始文件替代验证。
