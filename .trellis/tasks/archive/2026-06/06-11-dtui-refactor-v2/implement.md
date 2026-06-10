# dtui 重构执行计划

## Phase 1: 基础设施（父任务）

### 1.1 Context 系统
- [x] `context/context.go`: Context 接口 + ContextType 常量
- [x] `context/manager.go`: ContextManager (Push/Pop/SwitchPage)
- [x] `context/tree.go`: ContextTree (所有 Context 实例持有)

### 1.2 按键系统
- [x] `keybinding/binding.go`: Binding 结构体 + Scope 常量
- [x] `keybinding/registry.go`: AllBindings() 集中注册
- [x] `keybinding/help.go`: 帮助文本生成（供 bubbles/help 使用）

### 1.3 布局系统
- [x] `layout/layout.go`: 双栏布局计算 + 全屏模式
- [x] `layout/resizer.go`: 窗口尺寸变化处理

### 1.4 App 顶层结构
- [x] `app.go`: App struct（持有 ContextManager + Docker Client + Config）
- [x] `app_update.go`: App.Update() 全局消息路由
- [x] `app_view.go`: App.View() 布局拼接

### 1.5 公共组件
- [x] `components/statusbar.go`: 状态栏
- [x] `components/tabbar.go`: Tab 栏
- [x] `components/confirm.go`: 确认弹窗
- [x] `components/helpbar.go`: 快捷键栏（bubbles/help 包装）

### 1.6 Action 层
- [x] `actions/history.go`: 历史记录中间件
- [x] `actions/types.go`: ActionMsg 类型

### 1.7 文件清理
- [x] 创建新目录结构
- [x] 删除旧文件（model.go, commands.go, keys.go, update.go, panel.go, panels_*.go, views/ 旧文件）
- [x] 保留 styles/

**验证**：`go build ./cli/dtui/...` → 应能编译（功能不完整但结构就绪）

---

## Phase 2: Container View（子任务 1）

### 2.1 列表页
- [ ] `pages/containers/page.go`: ContainerPage Model + Update + View
- [ ] `pages/containers/table.go`: bubbles/table 容器列表
- [ ] `pages/containers/detail.go`: 右侧详情面板（Stats/Info/Logs 摘要）
- [ ] 注册按键：s/R/l/e/i/x/H/tab

### 2.2 操作
- [ ] `actions/container.go`: Start/Stop/Restart Actions
- [ ] 集成历史记录

### 2.3 面板（全屏）
- [ ] `pages/containers/panels/log.go`: 日志面板（bubbles/viewport + 实时流）
- [ ] `pages/containers/panels/stats.go`: Stats 面板（实时 CPU/Mem）
- [ ] `pages/containers/panels/inspect.go`: 详情面板（多 Tab：概览/网络/挂载/环境变量）
- [ ] `pages/containers/panels/exec.go`: Exec 面板（bubbles/textinput + 输出）

**验证**：容器列表显示、启停/重启正常、日志/Stats/详情面板正常、Exec 正常

---

## Phase 3: Image View（子任务 2）

### 3.1 列表页
- [ ] `pages/images/page.go`: ImagePage Model + Update + View
- [ ] `pages/images/table.go`: bubbles/table 镜像列表
- [ ] `pages/images/detail.go`: 右侧详情（大小/标签/创建时间）
- [ ] 注册按键：s/d/t/p/h

### 3.2 操作
- [ ] `actions/image.go`: Save/Delete/Tag/Prune Actions

### 3.3 面板
- [ ] `pages/images/panels/history.go`: 镜像历史层面板
- [ ] `pages/images/panels/save.go`: 镜像保存面板（bubbles/filepicker）

**验证**：镜像列表显示、保存/删除/标签/清理正常、历史层显示正常

---

## Phase 4: Compose View（子任务 3）

### 4.1 列表页
- [ ] `pages/compose/page.go`: ComposePage Model + Update + View
- [ ] `pages/compose/table.go`: bubbles/table 服务列表
- [ ] 注册按键：s/u/i/c

### 4.2 操作
- [ ] `actions/compose.go`: ComposeUp Action
- [ ] 外部编辑器打开 compose 文件（保留 tea.ExecProcess）

**验证**：服务列表显示、compose up 正常、编辑 compose 文件正常

---

## Phase 5: Deploy View（子任务 4）

### 5.1 列表页
- [ ] `pages/deploy/page.go`: DeployPage Model + Update + View
- [ ] `pages/deploy/table.go`: 发布目标 + 发布包列表
- [ ] 注册按键：d

### 5.2 操作
- [ ] `actions/deploy.go`: 部署流程（备份 → 解压 → 清空 → 部署）
- [ ] `pages/deploy/deploy.go`: 部署确认弹窗 + 进度

### 5.3 FilePicker
- [ ] `pages/deploy/picker.go`: bubbles/filepicker 路径选择

**验证**：发布目标/包列表、文件夹部署正常、zip 部署正常、备份正常

---

## Phase 6: Settings View（子任务 5）

### 6.1 列表页
- [ ] `pages/settings/page.go`: SettingsPage Model + Update + View
- [ ] `pages/settings/table.go`: 配置项列表（编排目录/发布目标/发布包）
- [ ] 注册按键：a/d/c

### 6.2 操作
- [ ] `actions/settings.go`: 编排目录/发布目标/发布包 CRUD
- [ ] 编辑配置文件（外部编辑器，复用现有逻辑）

### 6.3 历史面板
- [ ] `pages/settings/history.go`: 操作历史面板
- [ ] 注册按键：H

**验证**：配置项显示、CRUD 正常、历史记录显示正常

---

## Phase 7: 集成验证

### 7.1 编译与静态检查
- [ ] `go build ./cli/dtui/...`
- [ ] `go vet ./cli/dtui/...`
- [ ] 无 lsp diagnostics 错误

### 7.2 测试
- [ ] 更新 `smoke_test.go` 适配新架构
- [ ] 新增 context/manager_test.go
- [ ] 新增 keybinding/registry_test.go
- [ ] `go test ./cli/dtui/...`

### 7.3 手动验证
- [ ] 所有 Tab 切换正常
- [ ] 所有快捷键正常
- [ ] 双栏布局正常（大终端）
- [ ] 单栏布局正常（80x24 终端）
- [ ] 鼠标点击正常
- [ ] Docker daemon 断连提示正常

---

## Rollback

每个 Phase 独立，可单独回滚。Phase 1 必须最先完成（提供共享基础设施）。Phase 2-6 可并行。Phase 7 必须最后执行。

回滚策略：如果 Phase N 出问题，可用 `git checkout` 恢复该 Phase 的文件，不影响其他 Phase。
