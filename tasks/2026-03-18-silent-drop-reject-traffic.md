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

## Stage 2 - Writing Plans

I'm using the writing-plans skill to create the implementation plan.

### 当前事实

- `internal/gateway/proxy/runner.go` 的 blocked 路径当前只有 `sess.MarkBlockedGeo(...) -> r.emit(...) -> return -> defer client.Close()`，未发现任何向客户端写拒绝 payload 的代码。
- `internal/gateway/proxy/runner_test.go` 已覆盖 `blocked_reason`、geo 字段和 no-dial，但没有客户端读侧断言，尚未形成“silent drop”证据。
- `README.md` / `docs/runbook.md` 当前只描述“拒绝/blocked/关闭连接”与查询方式，未发现“返回拒绝说明内容”的文档表述。

### Silent Drop Reject Traffic Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用最小改动确认并锁定 geo blocked 连接的 silent drop 语义：不返回应用层 payload、不拨号 upstream、仍然产生可查询的 `blocked` 事件。

**Architecture:** 先在 `internal/gateway/proxy/runner_test.go` 增加 blocked 连接读侧回归测试，直接从客户端视角验证 `geo_denied` 与 `geo_not_in_allowlist` 两条路径的 socket 行为。如果新断言已经通过，则 Stage 3 只保留测试与任务留痕更新；只有在测试暴露出 payload、额外拨号或文档歧义时，才最小化调整 `runner.go` 或 README 表述。

**Tech Stack:** Go、标准库 `net`/`io`、现有 proxy runner 单测、Markdown 任务留痕。

---

### Task 1: Build socket-level evidence for blocked connections

**Files:**
- Modify: `internal/gateway/proxy/runner_test.go`
- Modify: `tasks/2026-03-18-silent-drop-reject-traffic.md`

- [ ] **Step 1: Add blocked connection read-side assertions**

  在 `runner_test.go` 为 geo blocked 场景补一个客户端辅助断言：连接到被拦截规则后写入探测 payload，再设置短读超时并执行读取。验收要求是读取到的 payload 必须为空；读取结果允许是 `io.EOF`、`net.ErrClosed`、connection reset 或 timeout 任一，不把具体错误字符串写死。

- [ ] **Step 2: Run focused proxy tests to establish evidence**

  Run: `go test ./internal/gateway/proxy -run 'TestGeoPolicy(AllowModeWithRequireAllowHit|BlockedConnectionDoesNotDial|RecordsGeoInfoInBlockedEvent|.*SilentDrop.*)' -count=1`

  Expected:
  - 若新断言首次即通过，记录“当前实现已符合 silent drop 语义”，Stage 3 不改 production code。
  - 若失败且读取到非空 payload、发生 upstream dial、或 blocked 事件缺失，记录具体现象并进入最小实现修复。

- [ ] **Step 3: Preserve existing blocked observability assertions**

  在新增读侧断言的同时，保留或复用现有 `blocked_reason`、geo 字段与 no-dial 断言，确保 silent drop 不以牺牲观测能力为代价。

- [ ] **Step 4: Re-run the proxy package after any code/test adjustments**

  Run: `go test ./internal/gateway/proxy -count=1`

  Expected: PASS

- [ ] **Step 5: Commit Stage 3 implementation**

  ```bash
  git add internal/gateway/proxy/runner_test.go internal/gateway/proxy/runner.go README.md tasks/2026-03-18-silent-drop-reject-traffic.md
  git commit -m "test: verify blocked geo connections close silently"
  ```

### Task 2: Clarify docs only if evidence shows wording risk

**Files:**
- Modify: `README.md`
- Modify: `tasks/2026-03-18-silent-drop-reject-traffic.md`

- [ ] **Step 1: Re-check README wording after test evidence**

  仅当 Stage 3 发现当前文档容易被理解为“会返回拒绝内容”时，补一句明确表述：连接会记 `blocked` 事件并关闭，不返回应用层说明 payload。

- [ ] **Step 2: Keep doc scope minimal**

  不扩展到新的 reject 场景，不修改 geo 匹配逻辑说明；只澄清 geo blocked 连接的关闭语义。

- [ ] **Step 3: Fold any doc clarification into the same Stage 3 commit**

  如果无需文档澄清，则该任务整体跳过，并在 Stage 3 留痕里注明“未发现文档歧义，无文档改动”。

## Stage 3 - Executing Plans

I'm using the executing-plans skill to implement this plan.

### 执行摘要

- 按 `test-driven-development` 先补 `internal/gateway/proxy/runner_test.go` 的 blocked 读侧断言，再执行验证。
- 初次直接运行现有集成式 blocked 用例时，`go test` 先后暴露了两个环境限制：
  - 默认 Go build cache 不可写，需要改用 `GOCACHE=/tmp/pipescope-gocache` 与 `GOTMPDIR=/tmp/pipescope-gotmp`
  - 当前沙箱禁止 `listen tcp 127.0.0.1:0`，导致基于 `runner.Start + net.Dial` 的 blocked 用例无法在本环境执行
- 为了继续在 `runner_test.go` 里验证同一条业务路径，我把目标 blocked 用例改为 `net.Pipe + runner.proxyConn(...)` 的单元测试形态，仍然覆盖 geo policy blocked 分支，但不依赖本地监听端口。

### 实际改动

- 新增 `assertReadReturnsNoPayload`：对客户端读侧进行统一断言，要求返回 payload 长度为 0；允许 EOF/closed/reset/timeout 等无 payload 结果。
- 新增 `startProxyConnWithPipe`：用 `net.Pipe` 驱动 `runner.proxyConn(...)`，并补齐 `activeConns/connWG` 的测试前置状态。
- 将以下三条 geo blocked 相关测试切换到上述单元测试路径，并保留原有 blocked/no-dial/geo 字段验收：
  - `TestGeoPolicyAllowModeWithRequireAllowHit`
  - `TestGeoPolicyBlockedConnectionDoesNotDial`
  - `TestGeoPolicyRecordsGeoInfoInBlockedEvent`
- 未修改 `internal/gateway/proxy/runner.go`
- 未修改 `README.md` / `docs/runbook.md`，因为未发现“返回拒绝说明内容”的文档歧义

### 验证证据

Run:

```bash
mkdir -p /tmp/pipescope-gocache /tmp/pipescope-gotmp && \
GOCACHE=/tmp/pipescope-gocache GOTMPDIR=/tmp/pipescope-gotmp \
go test ./internal/gateway/proxy -run 'TestGeoPolicy(AllowModeWithRequireAllowHit|BlockedConnectionDoesNotDial|RecordsGeoInfoInBlockedEvent)$' -count=1
```

Result:

```text
ok  	pipescope/internal/gateway/proxy	0.762s
```

### 结论

- 新增读侧断言在首次可执行验证中直接通过，说明当前 geo blocked 路径已经符合 silent drop 语义：无应用层 payload、无 upstream dial、`blocked` 事件与 geo 字段保持。
- 本阶段属于“补证据与锁行为”，不是生产逻辑修复。
- 按用户要求应在本阶段提交，计划提交信息为 `test: verify blocked geo connections close silently`；但当前沙箱禁止写 `.git`，无法创建 `index.lock`，所以实际 commit 仍被环境阻止。

## Stage 4 - Requesting Code Review

I'm using the requesting-code-review skill to review the current diff with Codex's regular capabilities.

### Review scope

- Reviewed diff: `internal/gateway/proxy/runner_test.go` and this task log
- Review method: manual code review against Stage 1/2 acceptance criteria; no `codex review` subcommand

### Findings

#### Important

1. `internal/gateway/proxy/runner_test.go:195-209`

   `assertReadReturnsNoPayload` 现在允许 `net.Pipe` 读超时也算通过。对于已经改成 `net.Pipe + proxyConn` 的测试路径，这会把验收从“blocked 后直接关闭连接且无 payload”放宽成“200ms 内没收到 payload”，从而可能放过一个永远不关闭但也不写回内容的挂起实现。因为 `net.Pipe` 的对端 `Close()` 会稳定反映到读侧，这里应该把 timeout 视为失败，并明确要求 EOF/closed 类结果，才能真正锁住 silent close 语义。

### Review summary

- Critical: 0
- Important: 1
- Minor: 0

### 下一步

- Stage 5 按 `receiving-code-review` 先验证该问题是否成立；若成立，收紧 helper 断言并重跑受影响测试。
- 按用户要求应在本阶段提交，计划提交信息为 `docs: record silent drop review findings`；但当前沙箱仍禁止写 `.git`，实际 commit 预计继续受阻。

## Stage 5 - Receiving Code Review

I'm using the receiving-code-review skill to verify and address the Stage 4 finding.

### 对 review 项的技术核验

- Review 要求重述：既然 blocked 测试已经切到 `net.Pipe + proxyConn`，读侧断言就不该把 timeout 视为“silent close”通过条件，而应明确要求 close 类结果。
- 本地代码核验：检查了 `/usr/local/Cellar/go/1.26.1/libexec/src/net/pipe.go`
  - `remoteDone` 关闭时，`Read` 返回 `io.EOF`
  - `localDone` 关闭时，`Read` 返回 `io.ErrClosedPipe`
  - deadline 到期时，`Read` 返回 `os.ErrDeadlineExceeded`
- 结论：Stage 4 的 Important 问题成立；当前 helper 的确放宽了 silent close 验收。

### 修复动作

- 在 `internal/gateway/proxy/runner_test.go` 给 `assertReadReturnsNoPayload` 增加 `errors` 判断：
  - 仍然要求 payload 长度为 0
  - 不再接受 timeout
  - 仅接受 `io.EOF`、`io.ErrClosedPipe`、`net.ErrClosed` 这类 close-related 错误

### 修复后验证

Run:

```bash
mkdir -p /tmp/pipescope-gocache /tmp/pipescope-gotmp && \
GOCACHE=/tmp/pipescope-gocache GOTMPDIR=/tmp/pipescope-gotmp \
go test ./internal/gateway/proxy -run 'TestGeoPolicy(AllowModeWithRequireAllowHit|BlockedConnectionDoesNotDial|RecordsGeoInfoInBlockedEvent)$' -count=1
```

Result:

```text
ok  	pipescope/internal/gateway/proxy	0.876s
```

### Review round outcome

- Round 1 Important issue: fixed
- Round 2: no additional findings identified in the updated diff
- 累计 review 轮次：1/3

### 阶段状态

- 本阶段已经完成对 review 问题的验证和修复。
- 按用户要求应在本阶段提交，计划提交信息为 `test: require blocked pipe reads to close cleanly`；但当前沙箱仍禁止写 `.git`，实际 commit 仍无法执行。

## Stage 6 - Implement TCP Silent Drop Window

### 执行摘要

- 按用户已确认的“方案 1”修改 `internal/gateway/proxy/runner.go`，在 Runner 内部新增默认 `blockedDropDuration` 与 setter，不接配置文件/CLI。
- geo blocked 分支从“emit 后立即 return + close”改成 `MarkBlockedGeo -> emit -> silentDrop(client) -> return`。
- `silentDrop(client)` 在窗口内只读并吞掉客户端发来的字节，不向客户端写任何响应；窗口到期后返回，交给原有 `defer client.Close()` 收尾。
- `internal/gateway/proxy/runner_test.go` 的 blocked 单测同步升级为窗口语义：先验证短窗口内读不到 payload，再验证窗口结束后连接关闭，同时保留 `no-dial`、`blocked_reason` 与 geo 字段断言。

### 实际改动

- `internal/gateway/proxy/runner.go`
  - 新增 `blockedDropDuration time.Duration`
  - `NewRunner(...)` 默认初始化为 `2 * time.Second`
  - 新增 `SetBlockedDropDuration(d time.Duration)`，仅在 `d > 0` 时生效
  - 新增 `silentDrop(conn net.Conn)`，通过 `SetReadDeadline(now + blockedDropDuration)` + 循环 `Read` 吞读数据
  - 在 geo blocked 分支中改为 `emit` 后执行 `silentDrop(client)`
- `internal/gateway/proxy/runner_test.go`
  - 新增 `assertWriteCompletes`、`assertReadTimesOutWithoutPayload`、`assertReadClosesWithoutPayload`、`assertSilentDropWindow`
  - `TestGeoPolicyAllowModeWithRequireAllowHit` 现在证明：
    - blocked 后窗口内读不到 payload
    - 窗口结束后连接关闭
    - `blocked_reason=geo_not_in_allowlist` 和 geo 字段仍正确
  - `TestGeoPolicyBlockedConnectionDoesNotDial` 现在证明：
    - blocked silent drop 期间无 payload
    - 窗口结束后关闭
    - upstream dial 次数仍为 0
  - `TestGeoPolicyRecordsGeoInfoInBlockedEvent` 现在证明：
    - blocked silent drop 行为存在
    - `blocked_reason=geo_denied` 与 country/province/city/adcode 仍保留

### 验证证据

Red（先写失败测试，再验证缺实现）：

```bash
mkdir -p /tmp/pipescope-gocache /tmp/pipescope-gotmp && \
GOCACHE=/tmp/pipescope-gocache GOTMPDIR=/tmp/pipescope-gotmp \
go test ./internal/gateway/proxy -run 'TestGeoPolicy(AllowModeWithRequireAllowHit|BlockedConnectionDoesNotDial|RecordsGeoInfoInBlockedEvent)$' -count=1
```

Result:

```text
# pipescope/internal/gateway/proxy [pipescope/internal/gateway/proxy.test]
internal/gateway/proxy/runner_test.go:378:9: runner.SetBlockedDropDuration undefined (type *Runner has no field or method SetBlockedDropDuration)
internal/gateway/proxy/runner_test.go:437:9: runner.SetBlockedDropDuration undefined (type *Runner has no field or method SetBlockedDropDuration)
internal/gateway/proxy/runner_test.go:491:9: runner.SetBlockedDropDuration undefined (type *Runner has no field or method SetBlockedDropDuration)
FAIL	pipescope/internal/gateway/proxy [build failed]
```

Green（实现后目标用例通过）：

```bash
mkdir -p /tmp/pipescope-gocache /tmp/pipescope-gotmp && \
GOCACHE=/tmp/pipescope-gocache GOTMPDIR=/tmp/pipescope-gotmp \
go test ./internal/gateway/proxy -run 'TestGeoPolicy(AllowModeWithRequireAllowHit|BlockedConnectionDoesNotDial|RecordsGeoInfoInBlockedEvent)$' -count=1
```

Result:

```text
ok  	pipescope/internal/gateway/proxy	1.277s
```

用户要求的整包验证命令已执行：

```bash
GOCACHE=/tmp/pipescope-gocache GOTMPDIR=/tmp/pipescope-gotmp go test ./internal/gateway/proxy -count=1
```

Result:

```text
--- FAIL: TestProxyForwardsBytes (0.00s)
    runner_test.go:19: listen echo: listen tcp 127.0.0.1:0: bind: operation not permitted
--- FAIL: TestGeoPolicyDenyModeBlocksMatchingIP (0.00s)
    runner_test.go:292: listen echo: listen tcp 127.0.0.1:0: bind: operation not permitted
--- FAIL: TestDialTimeoutStatus (0.00s)
    timeout_test.go:38: start runner: listen tcp 127.0.0.1:0: bind: operation not permitted
FAIL
```

### 风险与回归结论

- 这次改动已覆盖本任务要求的 TCP silent drop 窗口语义，且 `no-dial`、`blocked_reason`、geo 字段未回退。
- 当前 `emit(sess.Finalize())` 仍发生在 `silentDrop` 之前，因此 blocked 事件的 `DurationMS` 不包含 silent drop 窗口；这是按用户给定调用顺序实现的，但如果后续要把窗口时间计入会话时长，需要另起改动。
- 包内仍有 3 条历史测试依赖 `listen tcp 127.0.0.1:0`，在当前受限环境下无法通过；这不是 silent drop 逻辑回归，而是现有测试形态与沙箱能力不兼容。

### 下一步

- 如果要求在当前沙箱里让 `go test ./internal/gateway/proxy -count=1` 全绿，需要把剩余 bind 依赖测试（至少 `TestProxyForwardsBytes`、`TestGeoPolicyDenyModeBlocksMatchingIP`、`TestDialTimeoutStatus`）也重构为不依赖本地监听端口的测试形态。
