# 2026-03-18 Silent Drop Reject Traffic

## Stage 1 - Brainstorming

### 需求边界

- 目标范围：仅处理 PipeScope 已判定为“应拒绝”的入口流量。在当前源码里，明确可见的拒绝来源是 `geo_policy` 命中 `deny`，以及 `require_allow_hit=true` 且未命中 `allow`。
- 行为目标：命中拒绝后，不向客户端写入任何应用层拒绝说明内容；在转发前结束连接处理，同时保留内部 `blocked` 事件记录。
- 保留项：`status=blocked`、`blocked_reason`、geo 字段入库、admin sessions/analytics 可见性、`blocked` 统计口径不变。
- 非目标：不改 geo 规则匹配逻辑，不新增配置开关，不把 geo lookup 失败从当前 fail-open 改成 fail-close。
- 现状判断：源码审查显示当前 blocked 分支是 `MarkBlockedGeo -> emit -> return -> defer client.Close()`，未发现主动向对端写拒绝 payload 的代码。下一阶段应先复现实测问题；如果问题真实存在，优先排查是否来自别的 reject 路径、旧运行产物，或对连接关闭语义的误判。
- 术语约束：在当前用户态 TCP proxy 架构里，可落地的 “silent drop” 更准确是“不返回应用层说明内容，直接关闭/终止连接”；如果需求实际指网络层黑洞式丢包，不属于当前进程内改动可以完成的范围。

### 涉及模块候选

- `internal/gateway/proxy/runner.go`：reject 后的连接处置主路径，决定是否向客户端写回、是否继续拨号 upstream。
- `internal/gateway/session/session.go`：`blocked` 状态、原因与 geo 信息的会话建模，需要保证观测数据不回退。
- `internal/gateway/geo/policy.go`：判定“应拒绝”的直接来源，决定 Stage 2 默认覆盖哪些 blocked 场景。
- `internal/store/sqlite/writer.go` 与 `internal/store/sqlite/schema.sql`：blocked 事件持久化链路，作为回归验证对象，应保持行为不变。
- `internal/admin/service/query.go` 与 admin 查询/UI 展示链路：作为回归验证对象，确认 silent drop 后仍能查到 `blocked_reason`。
- `internal/gateway/proxy/runner_test.go`：优先补/改 socket 行为回归测试，断言 blocked 后无 upstream dial、无应用层返回 payload。
- `tests/e2e/pipescope_e2e_test.go`：如需端到端证据，可补“被拒绝连接无响应内容但事件仍可查询”的验证。
- `README.md`、`docs/runbook.md`、`assets/config.example.yaml`：若现有措辞容易被理解为“返回拒绝说明”，需要同步为 silent close/silent drop 的表述。

### 风险点

- `drop` 一词歧义较大：FIN close、RST、超时黑洞是三种不同观测结果；如果不先定验收口径，下一阶段容易“修了但验不掉”。
- 当前源码未见“拒绝说明回包”逻辑，需求可能针对未提交代码、旧运行产物，或对 EOF/连接关闭的误判；若直接改实现，容易做出无效变更。
- 跨平台 socket 表现不稳定：客户端在 blocked 后可能读到 EOF、connection reset 或 timeout；测试不应把某个具体错误字符串写死，应断言“没有业务 payload，且未拨号 upstream”。
- 不能丢失观测能力：silent drop 不能破坏 `blocked_reason`、geo 字段、实时会话页和统计查询。
- 作用域膨胀风险：当前明确的 reject 路径只有 geo policy；若把任务泛化到 `dial_fail`、`timeout` 等所有失败场景，会把需求从“拒绝流量”扩成“所有异常连接”，不应默认扩大。
- geo lookup 当前是 fail-open；如果下一阶段顺手改成 lookup 失败也拒绝，会引入额外行为回归。

### 下一阶段输入（给 writing-plans）

- 实施范围默认限定为 geo policy blocked 路径：`geo_denied` 与 `geo_not_in_allowlist`。除非先找到其他主动 reject 路径，否则不要扩 scope。
- 第一步先建立证据：
  1. 用单测或 e2e 复现“对端收到拒绝说明内容”的现象；如果复现不了，记录为“源码与现象不一致，先核实部署产物或复现脚本”。
  2. 把 blocked 连接的验收口径固定为：无应用层 payload 返回、无 upstream dial、`blocked` 事件仍可落库查询。
- 测试优先级：
  1. 在 `internal/gateway/proxy/runner_test.go` 增加 blocked 连接读侧断言：客户端读取不到任何说明内容；允许 EOF/closed/reset 任一结果，但 payload 必须为空。
  2. 保留并复用现有 no-dial 与 `blocked_reason` 测试。
  3. 如有必要，再补一条 e2e，验证 admin `/api/sessions` 仍能看到 blocked 事件。
- 代码变更预期：首选只动 `internal/gateway/proxy/runner.go` 及其测试；只有在产品术语和文档表述不一致时，再补 `README.md` / `docs/runbook.md`。
- 完成标准：reject 流量不返回说明内容，upstream 不被拨号，`status=blocked` 与 `blocked_reason` 保持，相关测试覆盖通过，文档用词与实现一致。
