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

- use superpower:writing-plans skill
  - Status: PENDING
  - Artifact target: `docs/superpowers/plans/2026-03-20-admin-copy-simplification.md`

- use superpower:executing-plans skill
  - Status: PENDING
  - TDD requirement: RED -> GREEN -> regression verification

- use superpower:requesting-code-review skill
  - Status: PENDING
  - Constraint: review stage must not use `codex review`

- use superpower:receiving-code-review skill
  - Status: PENDING
  - Max rounds: `3`

- use superpower:verification-before-completion skill
  - Status: PENDING

## Review

- Round 1: PENDING
- Round 2: NOT_STARTED
- Round 3: NOT_STARTED

## Verification

- Brainstorming阶段无需运行产品测试；spec review 结果：
  - Round 1: FAIL
  - Round 2: PASS

## Hard Check

- `git diff --name-only --cached | rg "^PROCESS_CHECKLIST\\.md$"`: PENDING

## Push / PR

- Branch push: PENDING
- PR creation: PENDING

## Final

- Checklist archive target: PENDING
- `docs/pr/<PR>.md` reference: PENDING
- Status: IN_PROGRESS
