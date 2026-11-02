# 获取帮助

请先查文档，再提 Issue：多数问题在 `docs/` 已有答案。

## 先看这些

| 我想… | 去哪里 |
| --- | --- |
| 跑起来看看 | [快速开始](./docs/quick-start.md) |
| 了解系统结构 | [架构概览](./docs/architecture.md) |
| 查服务端口 | [服务端口清单](./docs/service-ports.md) |
| 对接某个协议或服务 | [文档索引](./docs/README.md) |
| 新增网关或接口 | [网关标准开发流程](./docs/gateway-standard.md) |
| 处理错误码 | [错误码规范](./docs/error-codes.md) |
| 本地依赖怎么起 | [部署模板总览](./deploy/README.md) |

## 提问渠道

| 类型 | 渠道 |
| --- | --- |
| 可复现的缺陷 | [提交 Bug 报告](https://github.com/maomao94/zero-service/issues/new?template=bug_report.yml) |
| 新功能或改进建议 | [提交功能请求](https://github.com/maomao94/zero-service/issues/new?template=feature_request.yml) |
| 文档错误、失效链接 | 直接提 Issue 或按 [贡献指南](./CONTRIBUTING.md) 提 PR |
| 安全漏洞 | **不要公开提 Issue**，见 [安全策略](./SECURITY.md) |

## 提 Issue 前请确认

- 已查阅对应服务的文档与 `etc/*.yaml` 默认配置
- 已确认问题可以在最新 `develop` 分支复现
- 附上了版本（tag 或 commit）、操作系统、Go 版本与完整报错

## 响应预期

本项目由维护者利用业余时间维护，不提供商业 SLA。Issue 通常在一周内得到首次回复；带最小复现的 Issue 处理更快。

需要定制开发、私有部署支持或长期维护承诺，请在 Issue 中说明场景，维护者会单独沟通。
