# Map City List Scroll Design

## Context

- Scope is intentionally narrow:
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/MapPage.test.ts`
  - process artifacts for the requested Codex workflow chain
- User feedback: when the city list grows long, the map page should stay tidy instead of stretching indefinitely; the list should scroll inside a bounded region.
- Existing behavior that must remain intact:
  - right-side map/sidebar layout
  - single metric selector drives both coloring and list ordering
  - default window stays `1d`
- Constraints from the task:
  - work only in the main repository on a normal branch
  - do not use `git worktree`
  - review stage must not use `codex review`
  - every required stage ends with a commit

## Options

### Option 1: Add a dedicated scroll wrapper around the city list

- Keep existing data flow and list rendering.
- Wrap the sidebar list in a bounded container with `max-height` and `overflow-y: auto`.
- Add one focused test that asserts the scroll container exists around the list.

Recommendation: yes. This directly solves the UX issue with the smallest surface area and keeps current layout/data behavior unchanged.

### Option 2: Make the entire sidebar fixed-height and internally scrollable

- Turn the whole sidebar into a constrained panel.
- Header and list would share the same vertical constraint.

Trade-off: still valid, but it changes more layout behavior than needed and makes the header scroll context less precise.

### Option 3: Rebuild the city list into a virtualized component

- Use virtualization for very large result sets.

Trade-off: unnecessary for this request and adds complexity well beyond the stated need.

## Chosen Design

### Layout

- Preserve the existing `map-layout` two-column structure.
- Keep the sidebar header static above the list.
- Add a page-local wrapper such as `city-list-scroll` around the existing `.city-list`.

### Scroll Behavior

- Apply a reasonable `max-height` to the new wrapper and set `overflow-y: auto`.
- Keep the change page-local so other pages using `.city-list` do not change.
- Use a viewport-aware cap so long lists stop stretching the page while still showing several rows at once.

### Testing

- Follow TDD:
  - add a failing test that requires the sidebar list to be rendered inside the new scroll container
  - then implement the minimal template/style change to make it pass
- Re-run the full `MapPage` test file and the production build as required by the task.

## Approval Basis

- The user request already fixes scope and constraints tightly enough that the minimal scroll-wrapper approach is treated as the approved design for this session.
