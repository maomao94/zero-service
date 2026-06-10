# Implementation Plan: dtui 配置统一与 Tab 重构

## 执行清单

### Step 1: 更新 Tab 名称显示
- [ ] `views/layout.go`: RenderTabs 中 "发布" → "部署"
- [ ] `views/layout.go`: RenderFooter 中 case 3 帮助文本更新

### Step 2: 调整按键逻辑
- [ ] `keys.go`: handleKey 中 "c" 键逻辑调整
  - 移除 ComposeView 和 DeployView 的 config 编辑
  - 新增 ComposeView 的 compose 文件编辑

### Step 3: 新增编辑 compose 文件命令
- [ ] `commands.go`: 新增 openComposeFileCmd() 方法
- [ ] `commands.go`: 复用 openConfigCmd 的编辑器查找逻辑

### Step 4: 更新配置默认值
- [ ] `config/config.go`: InitDefault 中 "测试发布" → "测试部署"

### Step 5: 验证编译
- [ ] `go build ./cli/dtui/...` 编译通过
- [ ] `go vet ./cli/dtui/...` 无警告

## 验证命令

```bash
# 编译
cd cli/dtui && go build -o bin/dtui .

# 运行测试
go test ./cli/dtui/...

# 手动验证
./bin/dtui
# 1. Tab 切换确认名称：容器 | 镜像 | 编排 | 部署 | 设置
# 2. 编排页按 c 打开 compose 文件编辑器
# 3. 部署页按 d 执行部署（无 c 键）
# 4. 设置页按 c 打开 config.json 编辑器
```

## 回滚方案

如果需要回滚，恢复以下文件：
- `cli/dtui/internal/tui/keys.go`
- `cli/dtui/internal/tui/commands.go`
- `cli/dtui/internal/tui/views/layout.go`
- `cli/dtui/internal/config/config.go`

## 相关文件

- `cli/dtui/internal/tui/keys.go` - 按键处理
- `cli/dtui/internal/tui/commands.go` - 命令执行
- `cli/dtui/internal/tui/views/layout.go` - UI 渲染
- `cli/dtui/internal/config/config.go` - 配置结构
