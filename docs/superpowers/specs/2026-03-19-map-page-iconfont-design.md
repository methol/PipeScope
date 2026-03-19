# Map Page Iconfont Follow-up Design

## Context

- Scope stays inside `web/admin/src/pages/MapPage.vue` and `web/admin/src/pages/MapPage.test.ts`, plus PR/process docs for this follow-up.
- This is an incremental follow-up on top of the existing compact-right-sidebar work already present on `feature/map-page-ui-compact-20260319`.
- The user request is already concrete enough to act as design approval for a minimal in-place follow-up:
  - replace the current `连/流` short-text chips with clearer iconfont-style stats
  - keep tooltip hints for both stats
  - do not regress right-sidebar layout, one-metric coloring/order linkage, or default `1d`
- Constraints:
  - do not use `git worktree`
  - stay on the current branch
  - commit after each required workflow stage
  - review in-session without `codex review`
  - review/fix loop capped at 3 rounds

## Options

### Option 1: Inline SVG badges with iconfont-style visual treatment

- Keep the existing sidebar structure.
- Replace `连/流` text prefixes with small inline SVG glyphs, styled like lightweight iconfont chips.
- Preserve tooltip text on each stat chip and add hidden accessible labels.

Recommendation: yes. No new dependency, clear icon meaning, low regression risk, and straightforward to test.

### Option 2: Local iconfont asset + CSS classes

- Add a local font or icon sprite and reference it from page-local CSS classes.

Trade-off: closer to literal iconfont delivery, but adds asset plumbing for a very small UI change.

### Option 3: Third-party icon library

- Install a Vue/icon package and render stock icons in the badges.

Trade-off: easiest to author, but violates the low-risk/minimal-dependency goal.

## Chosen Design

### Stat Badge Structure

- Keep two badges per city row:
  - connection badge
  - traffic badge
- Each badge renders:
  - decorative icon (`aria-hidden="true"`)
  - visible numeric value
  - hidden text label for screen readers
- Tooltip stays on the badge container via `title`, so hover behavior remains unchanged.

### Visual Direction

- Keep the compact pill form factor from the current sidebar.
- Upgrade the badge style to feel more icon-led:
  - icon set in a small circular or inset carrier
  - slightly stronger contrast between connection and traffic variants
  - tighter numeric rhythm so the icon is the primary identifier instead of the first Chinese character
- Do not expand the sidebar width or add new controls.

### Behavior Preservation

- No change to:
  - right-side sidebar placement
  - single `metric` selector driving both map coloring and sidebar order
  - default window `1d`
  - tooltip content semantics
- Sorting remains descending by the selected metric.

### Testing

- Update the focused sidebar test first so it expects icon-style badges rather than `连/流` prefixes.
- Keep the required focused run for:
  - compact city stats in the right sidebar
  - one metric selector driving map coloring and sidebar order
  - default `1d` window requests
- Re-run the full `mapCity` + `MapPage` regression and production build after implementation.
