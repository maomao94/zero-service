# Checklist

## A2UI协议层
- [x] /common/einox/a2ui/types.go 包含所有A2UI类型定义
- [x] 所有组件构建函数（NewTextComponent、NewCardComponent等）可用
- [x] 所有消息生成函数（NewInterruptRequest等）可用
- [x] 编解码函数（Encode、Decode）正常工作

## 编译验证
- [x] go build ./common/einox/... 编译通过
- [x] go build ./aiapp/aisolo/... 编译通过
- [x] go build ./aiapp/aigtw/... 编译通过
- [x] 所有接口API完全兼容

## 前端中断弹窗
- [x] 弹窗样式美观，卡片式设计，阴影圆角
- [x] 半透明遮罩层，背景模糊效果
- [x] 确认/取消按钮样式统一，hover效果
- [x] 弹窗打开/关闭有平滑过渡动画
- [x] 支持键盘快捷键：回车确认、ESC取消
- [x] 点击按钮后显示加载状态
- [x] 移动端响应式适配正常

## 日志格式
- [x] convert.go中所有日志使用logx格式
- [x] 日志包含时间戳、级别等标准字段

## Resume流式响应
- [ ] aisolo.proto中包含ResumeStream接口定义
- [ ] aisolo服务实现ResumeStream方法
- [ ] aigtw网关包含/resume/stream路由
- [ ] 前端resumeAction函数调用流式接口
- [ ] 中断恢复后能正确接收后续消息

## AgentMode扩展
- [ ] aisolo.proto中包含plan和spec模式
- [ ] 后端Agent路由支持新模式
- [ ] 前端模式选择器支持新模式
- [ ] plan模式任务规划逻辑正确
- [ ] spec模式规格制定逻辑正确

## 智能体选择界面
- [ ] 智能体选择卡片组件设计美观
- [ ] 智能体列表正常展示
- [ ] 搜索和筛选功能正常
- [ ] 智能体详情弹窗正常
- [ ] 选择智能体后正确切换

## 功能测试
- [ ] 页面正常加载，无JS错误
- [ ] 智能体选择功能正常
- [ ] 多模式切换功能正常
- [ ] 发送"查询当前时间"弹出确认弹窗
- [ ] 点击确认后正确调用工具返回时间
- [ ] 点击取消后正确返回取消提示
- [ ] 操作完成后弹窗正确关闭
- [ ] 后续消息正确显示
