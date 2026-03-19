# Map Page UI Compact Design

## Context

- Scope: `web/admin/src/pages/MapPage.vue`, `web/admin/src/pages/MapPage.test.ts`, and any page-local styling needed for the new layout.
- Continuation note: this design formalizes the already-started local WIP on `feature/map-page-ui-compact-20260319`; the current task request fixes the scope and acts as approval to continue this compact UI direction.
- User requirements are already concrete:
  - map-side city stats should use compact text/icon-like labels instead of verbose `连接 X · 流量 Y`
  - city and traffic info should move to the right side of the map
  - map coloring and list sorting should be driven by one selector
  - default window should be `1d`
- Delivery constraints:
  - stay on the existing branch and do not use `git worktree`
  - commit after each workflow stage, including docs-only stages
  - perform review in-session without the `codex review` subcommand
  - final docs must reflect the real git history, verification evidence, and push outcome
- Non-goals:
  - no backend/API contract changes
  - no geo join / tooltip / province-boundary logic rewrite
  - no new component split unless current file becomes hard to follow

## Options

### Option 1: Minimal in-place UI compaction

- Keep `MapPage.vue` as the only production file.
- Replace the bottom list with a right-side compact sidebar.
- Use the existing `metric` selector as the single source for both map coloring and list ordering.
- Fix list order to descending by the chosen metric and remove extra sort controls.

Recommendation: yes. This matches the request directly, keeps behavior predictable, and limits regression risk to one page.

### Option 2: Extract a dedicated stats sidebar component

- Add a new sidebar component and move list rendering there.
- Keep the same user-facing behavior as Option 1.

Trade-off: cleaner structure, but higher change surface and unnecessary for this small UI update.

### Option 3: Rework the map page into multiple cards

- Split map, sidebar, and controls into multiple panels with a broader layout redesign.

Trade-off: could look cleaner, but exceeds the requested scope and risks unrelated CSS regressions.

## Chosen Design

### Layout

- Introduce a `map-layout` two-column area:
  - left: chart and existing map metadata
  - right: compact city stats sidebar
- Keep responsive behavior by stacking the sidebar below the map on narrow screens only.
- Move the returned-city summary into the sidebar header so the lower area stays clear.

### Compact Stat Expression

- Replace `连接 X · 流量 Y` with two compact stat chips:
  - `连 X`
  - `流 Y`
- Preserve detailed meaning through `title` attributes on the chips and on the sidebar summary text.
- Keep city name and province readable, but visually secondary to the compact metric chips.

### Unified Metric Logic

- Remove the separate sort-field selector and sort-order selector from the UI.
- Use `metric` as the only selector for both:
  - ECharts `visualMap` formatter / value source
  - right-side city list ordering
- Keep ordering descending so the sidebar remains a true top list.

### Default Window

- Change initial `windowText` from `1h` to `1d`.
- Existing data loading flows (`fetchAnalyticsOptions`, `fetchChinaMap`) continue unchanged, just with the new default query value.

### Testing

- Add RED tests first for:
  - default `1d` window requests
  - right-side compact sidebar structure and chip titles
  - unified metric selection driving both rendering and list order
- Keep existing tooltip, geo map, no-data, and error-state regressions intact.
