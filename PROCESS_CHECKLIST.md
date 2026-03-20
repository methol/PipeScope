# PROCESS_CHECKLIST

## Task

页面顶部提示文案精简，并将“统计/分析”统一改为“统计”。

## Scope

- 目标仓库：`/Users/methol/code/github.com/methol/PipeScope`
- 目标页面：
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/AnalyticsPage.vue`
  - `web/admin/src/App.vue`
- 相关测试：
  - `web/admin/src/pages/MapPage.test.ts`
  - `web/admin/src/pages/AnalyticsPage.test.ts`
  - `web/admin/src/pages/App.test.ts`

## Branch

- Base branch sync:
  - Command: `git fetch origin main && git pull --ff-only origin main`
  - Result: PASS (`Already up to date.`)
- Start branch: `main`
- Start HEAD: `e2bc7cd788994db8745f21f6e09b7e694939a4d2`
- Feature branch command:
  - `git checkout -b feat/map-copy-cleanup-20260320`
- Feature branch result:
  - PASS
- Effective branch: `feat/map-copy-cleanup-20260320`

## Stage Status

- use superpower:brainstorming skill
  - Status: DONE
  - Artifact target: `docs/superpowers/specs/2026-03-20-admin-copy-simplification-design.md`
  - Context checked:
    - `web/admin/src/pages/MapPage.vue`
    - `web/admin/src/pages/AnalyticsPage.vue`
    - `web/admin/src/App.vue`
    - `web/admin/src/pages/MapPage.test.ts`
    - `web/admin/src/pages/AnalyticsPage.test.ts`
    - `web/admin/src/pages/App.test.ts`
    - recent commits on `main`
  - Approval note:
    - 用户需求已明确指定要删除/缩减的提示文案和统一命名方向，本次将其视为已给定的最小设计约束并直接固化到 spec。
  - Review rounds:
    - Spec review round 1: FAIL
      - Issue summary:
        - `统计/分析` 统一范围不够明确
        - 待删除/保留的文案清单不够机械化
        - 保留状态文案的测试要求不够明确
      - Action:
        - 补充 scope boundary、acceptance checklist、保留状态测试要求
    - Spec review round 2: PASS
      - Result: `✅ Approved`
  - Commit intent:
    - `git add PROCESS_CHECKLIST.md docs/superpowers/specs/2026-03-20-admin-copy-simplification-design.md`
    - `git commit -m "docs(ui): capture copy simplification design"`
  - Commit result:
    - PASS: `a318fcf docs(ui): capture copy simplification design`

- use superpower:writing-plans skill
  - Status: DONE
  - Artifact target: `docs/superpowers/plans/2026-03-20-admin-copy-simplification.md`
  - Review rounds:
    - Plan review round 1: FAIL
      - Issue summary:
        - 保留的 loading/error/empty-state 测试要求不完整
        - `npm test` / `npm run build` 未显式指向 `web/admin`
        - spec 对“产品范围”与“流程留痕产物”的边界表述不一致
      - Action:
        - 在 spec 补充 process artifacts 说明
        - 在 plan 补充保留状态测试
        - 所有前端命令统一改为 `npm --prefix web/admin ...`
    - Plan review round 2: PENDING
      - Issue summary:
        - “禁止 subagent-driven implementation” 与 “reviewer subagent” 的用途边界还不够明确
      - Action:
        - 明确 reviewer subagent 仅允许用于 spec/plan/code review，不用于实现
    - Plan review round 3: PENDING
      - Issue summary:
        - 计划未显式要求确认当前 feature 分支基于最新 `main`
      - Action:
        - 在 Chunk 1 增加 branch preflight 校验步骤与命令
    - Plan review round 4: PENDING
      - Issue summary:
        - branch preflight 用了固定 SHA，时间一久会失效
      - Action:
        - 改为 `git fetch origin main` + `git merge-base --is-ancestor origin/main HEAD`
    - Plan review round 5: PENDING
      - Result: `Approved`
  - Commit intent:
    - `git add PROCESS_CHECKLIST.md docs/superpowers/plans/2026-03-20-admin-copy-simplification.md`
    - `git commit -m "docs(ui): capture copy simplification plan"`
  - Commit result:
    - PASS: `bc941a1 docs(ui): capture copy simplification plan`

- use superpower:executing-plans skill
  - Status: DONE
  - TDD requirement: RED -> GREEN -> regression verification
  - Preflight:
    - Command: `git fetch origin main && git branch --show-current && git merge-base --is-ancestor origin/main HEAD`
    - Result: PASS (`feat/map-copy-cleanup-20260320`, merge-base exit `0`)
  - RED:
    - Command: `npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts -t "renders statistics tab label as 统计|renders heading as 统计 and omits the redundant manual-refresh hint while preserving loading and error states|omits redundant map meta hints while preserving loading error and empty-state messages"`
    - Result: FAIL
    - Failure summary:
      - `App.test.ts`: received `统计/分析`, expected `统计`
      - `AnalyticsPage.test.ts`: still rendered `统计/分析` and `分析型页面：不自动刷新（手动检索）`
      - `MapPage.test.ts`: still rendered `城市连接热度（市级边界） · 分析型页面（不自动刷新）`
  - GREEN:
    - Command: `npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts -t "renders statistics tab label as 统计|renders heading as 统计 and omits the redundant manual-refresh hint while preserving loading and error states|omits redundant map meta hints while preserving loading error and empty-state messages"`
    - Result: PASS (`3` tests passed)
  - Regression:
    - Command: `npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts`
    - Result: PASS (`31` tests passed)
  - Build:
    - Command: `npm --prefix web/admin run build`
    - Result: PASS (existing Vite chunk-size warning only)
  - Commit intent:
    - `git add PROCESS_CHECKLIST.md web/admin/src/App.vue web/admin/src/pages/App.test.ts web/admin/src/pages/AnalyticsPage.vue web/admin/src/pages/AnalyticsPage.test.ts web/admin/src/pages/MapPage.vue web/admin/src/pages/MapPage.test.ts`
    - `git commit -m "fix(ui): simplify admin copy and rename statistics tab"`
  - Commit result:
    - PASS: `135524f fix(ui): simplify admin copy and rename statistics tab`

- use superpower:requesting-code-review skill
  - Status: DONE
  - Constraint: review stage must not use `codex review`
  - Review range:
    - Base: `bc941a14dbbbc56d2dc6c6fb724c32ea9a74e69c`
    - Head: `135524ff04c064ca4e5bae354f3987a5b596824f`
  - Review artifact target:
    - `docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`
  - Review result summary:
    - Critical: none
    - Important: none
    - Minor: none
    - Assessment: `Ready to merge? Yes`
  - Commit intent:
    - `git add PROCESS_CHECKLIST.md docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`
    - `git commit -m "docs(ui): request review for admin copy simplification"`
  - Commit result:
    - PASS: `08bb42e docs(ui): request review for admin copy simplification`

- use superpower:receiving-code-review skill
  - Status: DONE
  - Max rounds: `3`
  - Round 1 decision:
    - No actionable issues from reviewer
    - Verification command: `npm --prefix web/admin test -- --run src/pages/App.test.ts src/pages/AnalyticsPage.test.ts src/pages/MapPage.test.ts`
    - Verification result: PASS (`31` tests passed)
  - Round 2: not needed
  - Round 3: not needed
  - Commit intent:
    - `git add PROCESS_CHECKLIST.md docs/process-checklists/2026-03-20-admin-copy-simplification-review.md`
    - `git commit -m "docs(ui): record review receipt for admin copy simplification"`

- use superpower:verification-before-completion skill
  - Status: PENDING

## Review

- Round 1: PASS
  - Source: reviewer subagent on range `bc941a14dbbbc56d2dc6c6fb724c32ea9a74e69c..135524ff04c064ca4e5bae354f3987a5b596824f`
  - Findings: none
  - Receipt verification: PASS (`31` tests passed)
- Round 2: NOT_STARTED
- Round 3: NOT_STARTED

## Verification

- Brainstorming阶段无需运行产品测试；spec review 结果：
  - Round 1: FAIL
  - Round 2: PASS
- Writing-plans阶段无需运行产品测试；plan review 结果：
  - Round 1: FAIL
  - Round 5: PASS
- Executing-plans阶段验证：
  - RED focused suite: FAIL
  - GREEN focused suite: PASS
  - Related page regression suite: PASS (`31` tests)
  - Frontend build: PASS

## Hard Check

- `git diff --name-only --cached | rg "^PROCESS_CHECKLIST\\.md$"`: PENDING

## Push / PR

- Branch push: PENDING
- PR creation: PENDING

## Final

- Checklist archive target: PENDING
- `docs/pr/<PR>.md` reference: PENDING
- Status: IN_PROGRESS
