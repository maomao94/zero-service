# 补全 .trellis/spec/ 覆盖缺口 — Design

## Spec 风格约定

所有新增 spec 遵循现有风格：

1. **源文件驱动**：每条规则链接到真实文件路径 + 符号名
2. **简短 snippet**：只在辅助理解时贴代码，优先用文件名定位
3. **Good / Bad 对比**：复杂规则用正反例
4. **常见陷阱**：每个 spec 至少一个 Gotcha 小节
5. **中文**：标题和主体用中文

## 目录分层

新增 spec 文件的落点：

| Child Task | 目录 | 文件命名 |
|------------|------|---------|
| app-service-specs | `.trellis/spec/backend/` | `<service-name>-guidelines.md`，例如 `alarm-guidelines.md` |
| aiapp-service-specs | `.trellis/spec/backend/` | `aichat-guidelines.md` 等 |
| global-model-spec | `.trellis/spec/backend/` | 新增 `global-models.md` 或合并入 `database-guidelines.md` |
| common-infra-specs | `.trellis/spec/backend/` | `mcpx-guidelines.md`、`asynqx-guidelines.md`等 |
| remaining-gaps-specs | `.trellis/spec/backend/` | 按模块命名 |

## app/ 服务 spec 模板（最小集）

每个服务至少包含：

```markdown
# <服务名> 规范

## 服务职责
一句话说明。

## 入口
- proto/api 文件路径
- gen.sh 位置

## 关键 Logic
列出主要 logic 文件及其职能。

## 模型
- 涉及的表
- 写策略（Upsert/Insert-only）

## Gotcha
- 针对该服务的特有陷阱
```

## aiapp/ 服务 spec 模板

同上，增加：

- AI Provider/模型配置方式
- SSE 或流式响应的边界

## 更新索引

每个 child task 完成后，必须：
1. 在 `backend/index.md` 的对应小节添加条目
2. 按字母顺序插入
