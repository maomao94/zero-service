# dtui 面板交互优化执行计划

## Phase 1: 配置扩展 + FormPanel 基础 (2-3h)

### 1.1 配置结构扩展
- [ ] `config/config.go` 新增 `DeployPackage` 结构体
- [ ] `config/config.go` 新增 `AddDeployPackage`, `RemoveDeployPackage`
- [ ] `config/config.go` `InitDefault` 增加示例 deploy_packages
- [ ] 验证配置加载/保存兼容性

### 1.2 FormPanel 实现
- [ ] `tui/panels_form.go` 新增 `FormPanelImpl`
- [ ] 实现 Panel 接口全部方法
- [ ] 字段渲染：标签 + 输入值 + 光标
- [ ] Tab 切换字段，字符输入，Backspace 删除
- [ ] Enter 提交回调，Esc 取消回调
- [ ] placeholder 显示（字段为空时）

### 1.3 消息类型
- [ ] `tui/messages.go` 新增 `FormSubmitMsg`, `FormCancelMsg`
- [ ] `tui/update.go` 处理 FormSubmitMsg/FormCancelMsg

**验证**：`go build ./cli/dtui/... && go test ./cli/dtui/...`

## Phase 2: 设置页表单化 (1-2h)

### 2.1 按键改造
- [ ] `tui/keys.go` 设置页 `a` 键 → 根据选中区域打开对应 FormPanel
- [ ] `tui/keys.go` 设置页 `e` 键 → 编辑模式，回填当前值
- [ ] 编排目录表单：字段（名称、路径）
- [ ] 前端发布目标表单：字段（名称、容器、HTML 路径、备份目录）
- [ ] 发布包表单：字段（名称、路径）

### 2.2 表单提交处理
- [ ] `tui/update.go` FormSubmitMsg 处理：调用 config.AddXxx
- [ ] 提交后刷新设置页数据
- [ ] 错误处理（路径不存在等）

**验证**：手动测试新增/编辑三种配置类型

## Phase 3: 发布包配置 (1-2h)

### 3.1 数据加载
- [ ] `tui/commands.go` 新增 `loadDeployPackagesCmd`
- [ ] `tui/model.go` Model 新增 `deployPackages []DeployPackage`
- [ ] `tui/messages.go` LoadedMsg 增加 Packages 字段

### 3.2 前端发布视图
- [ ] `tui/views/containers.go` RenderDeploy 增加发布包区
- [ ] 发布目标和发布包分两区显示
- [ ] 选中发布包高亮

### 3.3 部署流程
- [ ] `tui/keys.go` DeployView `d` 键逻辑：
  - 有选中发布包 → 用包路径部署
  - 无发布包 → 打开 exec 面板输入路径
- [ ] `tui/commands.go` 新增 `deployPackageCmd`

**验证**：手动测试发布包选择和部署

## Phase 4: 日志换行 (1-2h)

### 4.1 换行渲染
- [ ] `views/detail_logs.go` Render 中对每行调用 WrapLines
- [ ] 续行不显示行号，用空格占位
- [ ] 搜索高亮在换行后正确

### 4.2 滚动重算
- [ ] `views/detail_logs.go` visibleLines 改为按渲染行数计算
- [ ] scrollPos 逻辑调整（按渲染行滚动而非日志条目）
- [ ] followTail 在换行模式下正确

**验证**：宽屏/窄屏切换测试日志显示

## Phase 5: 操作记录优化 (1h)

### 5.1 布局优化
- [ ] `views/history.go` 减少行间距
- [ ] 详情区域放底部固定区（选中条目的完整 error/info）
- [ ] 列表区占用更多行

### 5.2 信息密度
- [ ] 时间列缩短
- [ ] action 列紧凑
- [ ] target 列自适应宽度

**验证**：80x24 和宽屏下查看操作记录

## Phase 6: exec 面板自适应 (1h)

### 6.1 动态布局
- [ ] `panels_exec.go` Render 根据 height 动态分配输入/输出区域
- [ ] 小终端：输入区 3 行，输出区占剩余
- [ ] 大终端：输入区 5 行，输出区占剩余

### 6.2 输出换行
- [ ] 输出内容根据宽度自动换行

**验证**：小终端/大终端下 exec 面板显示

## Phase 7: 集成验证 (30min)

- [ ] `go build ./cli/dtui/...`
- [ ] `go vet ./cli/dtui/...`
- [ ] `go test ./cli/dtui/...`
- [ ] 手动验证所有场景

## 依赖关系

Phase 1 → Phase 2, 3 (FormPanel 和配置扩展是后续基础)
Phase 4, 5, 6 可并行 (互不依赖)
Phase 7 最后执行

## Rollback

每个 Phase 独立，可单独回滚。Phase 1 改动最大，如出问题可回退。
