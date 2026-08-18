# Write missing service docs

## Goal

为端口清单中缺少专门文档的核心服务补齐文档：file、gis、podengine 各一篇，bridge 系列一篇总览。

## Background

父任务 `08-18-organize-docs` 的一部分。`service-ports.md` 列出约 30 个服务，但只有 5 个服务有专门文档。file/gis/podengine 是 README 点名的核心业务服务，bridge 系列（bridgedump/bridgegtw/bridgekafka/bridgemodbus/bridgemqtt）共 5 个服务，用一篇总览避免碎片化。AI 服务（aiapp/）为实验性内容，明确排除。

## Requirements

1. 为 `app/file` 写 `docs/file/README.md`：服务职责、配置、关键接口、部署
2. 为 `app/gis` 写 `docs/gis/README.md`：服务职责、配置、关键能力（H3/GeoHash/围栏/坐标转换）
3. 为 `app/podengine` 写 `docs/podengine/README.md`：服务职责、配置、容器生命周期管理接口
4. 为 bridge 系列写 `docs/bridge/README.md` 总览：bridgegtw/bridgekafka/bridgemodbus/bridgemqtt/bridgedump 各自职责、端口、协议
5. 遵循 `.trellis/spec/guides/documentation-guide.md` 的内容归属规则
6. 依赖子任务 `08-18-docs-restructure` 完成后的目录结构放置文档

## Acceptance Criteria

- [ ] docs/file/README.md、docs/gis/README.md、docs/podengine/README.md、docs/bridge/README.md 已创建
- [ ] 每篇文档覆盖服务职责、配置、关键接口/能力
- [ ] 内容基于源码和配置事实，无编造
- [ ] 文档遵循 documentation-guide.md 的内容归属规则
- [ ] 新增入口由本任务追加到 docs/README.md；端口事实与各 `etc/*.yaml` 和 `service-ports.md` 一致，明确排除的服务不要求专项文档

## Out of Scope

- AI 服务文档（实验性内容，排除）
- alarm、logdump、xfusionmock、lalproxy 等内部服务文档
- 修改服务源码

## Technical Notes

- `common/gisx/geos`、`common/flowx` 已有 API 级 README，可作为素材
- 各服务配置文件位于 `app/<svc>/etc/`
- 接口、字段和校验以对应 `.proto`、`.api` 或 typed 源码为权威；端口以配置文件为证据
- 必须在 `08-18-docs-restructure` 通过检查并提供最终目录清单后开始

## Key Decisions

- 范围：file、gis、podengine 各一篇 + bridge 系列一篇总览
- bridge 系列不拆分单篇，用总览覆盖 5 个服务

## Open Questions

- 无
