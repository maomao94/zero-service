# Implementation plan

1. 记录移动前 10 个文件的校验值或内容差异基线。
2. 创建 5 个服务目录，移动明确文件，并创建各目录 README。
3. 更新移动文档中的同目录、跨目录、图片和源码链接。
4. 全仓库搜索旧路径，更新根 README、docs 索引、Spec 引用和其他真实引用；归档任务中的历史路径不改写。
5. 运行确定性相对链接检查、旧路径搜索和 `git diff --check`。
6. 确认除导航和路径外没有语义改写，输出最终目录清单。

## Validation

```bash
git diff --check
git status --short
```

另检查全仓库当前文档中的旧路径、Markdown/HTML 本地资源目标和 `docs/images/` 可达性。GitHub 锚点与外部 URL 人工抽查。

## Rollback Point

目录移动与所有链接更新必须作为一个原子交付；任一链接检查失败时不允许进入后续子任务。
