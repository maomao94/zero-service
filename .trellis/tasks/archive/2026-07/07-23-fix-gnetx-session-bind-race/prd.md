# 修复 gnetx Session 绑定关闭竞态

## Goal

保证 gnetx Session 的身份绑定与关闭在并发下保持一致，并让 ISP 客户端注册成功后绑定本端业务身份。

## Requirements

- SessionManager 的按 SessionID/ClientID 索引不得稳定返回已关闭 Session。
- BindClientID 与 Close 之间不得出现“关闭后仍绑定成功”的竞态。
- ClientID 冲突淘汰旧 Session 时，旧 Session 不得并发重新写回 ClientID 索引。
- gnetx ClientConn 支持绑定客户端业务身份。
- ISP 客户端注册成功后，将配置的 SendCode 绑定到当前 Session；注册绑定失败不得标记为已注册。
- 保留现有用户对 `app/ispagent/etc/ispagent.yaml` 的未提交改动。

## Acceptance Criteria

- [x] gnetx SessionManager 关闭/绑定并发测试通过 race 检查。
- [x] ClientConn 暴露 ClientID/BindClientID，且 ISP 注册路径调用绑定。
- [x] 相关 gnetx、isp 测试和 go vet 通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
