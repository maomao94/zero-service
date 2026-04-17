# AISolo智能体平台优化Spec

## Why
当前AISolo智能体平台需要以下优化：
1. Agent模式不够丰富，需要支持plan（规划）和spec（执行）等专业模式
2. 前端页面交互不够完善，需要支持先选择智能体再聊天的流程
3. 整体功能和体验需要全面对标字节的Solo产品

## What Changes
- 扩展AgentMode枚举，增加plan（规划）、spec（规格）等专业模式
- 前端增加智能体选择界面，支持先选择再聊天
- 前端中断弹窗全新设计：卡片式弹窗、动画效果、键盘快捷键、移动端适配
- Resume接口支持SSE流式响应，前端能完整接收中断恢复后的消息
- 日志统一使用logx格式

## Impact
- Affected specs: AISolo智能体服务、前端solo.html页面、A2UI协议
- Affected code: /common/einox/a2ui/*, /aiapp/aisolo/*, /aiapp/aigtw/*

## ADDED Requirements

### Requirement: AgentMode扩展
系统应支持多种Agent运行模式，满足不同场景需求。

#### Scenario: 快速对话
- **WHEN** 用户选择fast模式
- **THEN** Agent快速响应，适合简单问答和日常对话
- **AND** 延迟低，资源消耗少

#### Scenario: 深度思考
- **WHEN** 用户选择deep模式
- **THEN** Agent进行深度推理，适合复杂问题分析
- **AND** 延迟较高，但回答质量更高

#### Scenario: 任务规划
- **WHEN** 用户选择plan模式
- **THEN** Agent进行任务分解和规划，适合复杂任务
- **AND** 自动生成任务清单和执行计划

#### Scenario: 规格制定
- **WHEN** 用户选择spec模式
- **THEN** Agent进行详细规格制定，适合技术文档编写
- **AND** 生成结构化的规格文档

### Requirement: 智能体选择界面
系统应提供智能体选择界面，用户可以先选择智能体再开始聊天。

#### Scenario: 选择智能体
- **WHEN** 用户打开页面或点击切换智能体
- **THEN** 显示智能体列表卡片
- **AND** 每个智能体显示名称、描述、能力标签
- **AND** 用户可以搜索和筛选智能体

#### Scenario: 查看智能体详情
- **WHEN** 用户点击某个智能体卡片
- **THEN** 显示智能体详情弹窗
- **AND** 显示智能体介绍、可用工具、能力列表
- **AND** 用户可以确认选择或取消

### Requirement: 前端美观弹窗
系统应提供美观、流畅的中断确认弹窗交互体验。

#### Scenario: 用户确认操作
- **WHEN** Agent触发需要用户确认的中断请求
- **THEN** 页面显示美观的卡片式弹窗，带半透明遮罩和模糊背景
- **AND** 用户可以通过点击按钮或键盘快捷键（回车/ESC）进行操作
- **AND** 操作过程中有加载状态反馈

### Requirement: Resume流式响应
系统应支持Resume接口的SSE流式响应，确保中断恢复后能继续接收AI消息。

#### Scenario: 中断恢复流程
- **WHEN** 用户在中断弹窗点击确认/取消
- **THEN** 前端调用ResumeStream接口
- **AND** 后端返回SSE流，前端继续渲染后续消息
- **AND** 完整的对话流程无中断

### Requirement: 统一日志格式
系统应使用统一的logx日志格式，便于日志收集和分析。

#### Scenario: 日志记录
- **WHEN** 系统发生各类事件（请求、响应、错误、中断等）
- **THEN** 使用logx.InfoS/logx.Errorf等统一格式记录日志
- **AND** 日志包含时间戳、级别、消息内容等标准字段

## MODIFIED Requirements

### Requirement: 中断弹窗样式
原有的简单alert/confirm弹窗应升级为美观的卡片式设计。

#### Scenario: 弹窗显示
- **WHEN** 收到interruptRequest事件
- **THEN** 显示居中的卡片式弹窗
- **AND** 背景有半透明遮罩和模糊效果
- **AND** 弹窗有圆角、阴影等现代设计元素
- **AND** 按钮有hover效果和点击反馈
- **AND** 弹窗打开/关闭有平滑过渡动画

### Requirement: 智能体选择交互
页面应支持选择智能体后再开始聊天的交互流程。

#### Scenario: 选择模式
- **WHEN** 用户打开聊天页面
- **THEN** 显示模式选择器（auto/fast/deep/plan/spec）
- **AND** 用户选择模式后，切换到对应的智能体
- **AND** 切换模式时保持会话上下文

## REMOVED Requirements
- 无

## 字节Solo产品功能对标清单

### 核心功能
- [x] 智能体选择和切换
- [x] 多模式支持（fast/deep/plan/spec）
- [x] 流式对话输出
- [x] 中断和恢复机制
- [x] 用户确认交互（弹窗）
- [x] 工具调用展示
- [x] 会话历史管理

### 交互体验
- [x] 卡片式UI设计
- [x] 键盘快捷键支持
- [x] 移动端适配
- [x] 加载状态反馈
- [x] 错误提示
- [x] Toast通知

### 高级功能
- [ ] 多智能体协作
- [ ] 任务规划视图
- [ ] 工具调用链展示
- [ ] 代码执行预览
- [ ] 文件上传下载
