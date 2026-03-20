# Admin Copy Simplification Design

## Context

- Scope is limited to the existing admin frontend copy in:
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/AnalyticsPage.vue`
  - `web/admin/src/App.vue`
- Scope boundary:
  - visible copy cleanup in this task applies only to strings rendered from the three files above
  - documentation files and non-visible historical references are out of scope
  - if a same-class redundant hint is found elsewhere inside those three files, it should be cleaned up in the same pass
- Process artifacts:
  - `PROCESS_CHECKLIST.md`, archived checklist docs, and PR reference docs are required delivery artifacts for this task
  - these files are workflow traceability outputs, not part of the user-visible copy-change product scope
- User-requested cleanup is specific:
  - delete or shrink redundant top-of-page prompts such as:
    - `城市连接热度（市级边界） · 分析型页面（不自动刷新）`
    - `已载入 13 城市 · Top 1000 上限`
    - `分析型页面：不自动刷新（手动检索）`
  - unify every visible `统计/分析` label to `统计`
  - clean same-class duplicated hints for consistency
- Delivery constraints:
  - do not use `git worktree`
  - stay on a feature branch created from the latest `main`
  - commit after each required workflow stage
  - review must not use the `codex review` subcommand
- Non-goals:
  - no API changes
  - no new page/component split
  - no layout redesign beyond removing redundant copy

## Options

### Option 1: Minimal in-place copy cleanup

- Remove or hide only the redundant descriptive meta copy.
- Rename the analytics tab and page heading from `统计/分析` to `统计`.
- Update focused tests for removed/renamed copy.

Recommendation: yes. This matches the request directly and keeps behavior and layout risk low.

### Option 2: Shared copy constants for all page headers

- Introduce a shared copy map or constants module and route all labels through it.

Trade-off: better centralization, but unnecessary for three small strings and increases change surface.

### Option 3: Rework page headers as a unified header component

- Build a reusable header block and standardize all pages around it.

Trade-off: broader cleanup, but exceeds the requested scope and raises regression risk.

## Chosen Design

### Map Page

- Keep the existing map controls and chart/sidebar layout intact.
- Remove the redundant top meta sentence that combines the heatmap label with the “analysis page / no auto-refresh” note.
- Remove the returned-city-count helper text in the sidebar header because it duplicates the selected `Top` control and does not affect task completion.
- Keep functional states such as:
  - `筛选项加载中...`
  - `加载中...`
  - explicit error text
  - empty-state hint

### Analytics Page

- Rename the page heading from `统计/分析` to `统计`.
- Remove the top meta sentence `分析型页面：不自动刷新（手动检索）`.
- Keep all filters, loading states, and manual search behavior unchanged.

### Global Navigation

- Rename the analytics tab label from `统计/分析` to `统计`.

## Acceptance Checklist

- Rename:
  - `App.vue` analytics tab `统计/分析` -> `统计`
  - `AnalyticsPage.vue` heading `统计/分析` -> `统计`
- Remove:
  - `MapPage.vue` top meta `{{ title }} · 分析型页面（不自动刷新）`
  - `MapPage.vue` sidebar meta `已载入 N 城市 · Top M 上限`
  - `AnalyticsPage.vue` top meta `分析型页面：不自动刷新（手动检索）`
- Keep:
  - map/analytics filter controls and manual action buttons unchanged
  - loading, error, and empty-state status text unchanged unless directly duplicated by the removed hints
  - existing page layout structure unchanged apart from the removed text nodes

### Testing

- Add or update focused tests first so the copy change is verified in RED/GREEN order.
- Cover:
  - navigation tab label is `统计`
  - analytics page heading is `统计` and redundant meta copy is absent
  - map page no longer renders the removed redundant prompts
- Also verify retained conditional states still render when applicable:
  - `MapPage.vue`: loading, error, and empty-state hints
  - `AnalyticsPage.vue`: loading and error hints
- Re-run the related frontend tests and production build after implementation.
