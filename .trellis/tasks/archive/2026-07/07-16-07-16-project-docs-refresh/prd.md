# 整理项目文档

## Goal

梳理并更新项目级 README、docs 索引、服务端口清单和相关项目说明文档，使其反映当前服务布局、ISP/gnetx 能力与文档导航。

## Requirements

- 更新根 `README.md`，让项目能力、架构入口和服务清单反映当前代码库。
- 更新 `docs/README.md` 文档索引，补齐/修正服务文档导航。
- 更新 `docs/service-ports.md`，校准当前服务端口、目录和协议用途。
- 必要时补充项目相关文档，但避免大范围重写已有专题文档。
- 文档内容必须来自现有文件和代码配置，不写模板化占位内容。

## Acceptance Criteria

- [ ] 根 README 的服务和能力描述与当前项目一致。
- [ ] docs 索引能正确导航到主要用户、核心服务和开发者文档。
- [ ] 服务端口清单与当前 `etc/*.yaml` / 服务配置保持一致。
- [ ] 文档无明显过期路径、缺失链接或占位内容。

## Notes

- 用户指定重点文件：`README.md`、`docs/README.md`、`docs/service-ports.md`。
