# Map Page Layout Height Fix Design

## Context

- Scope stays focused on [`web/admin/src/pages/MapPage.vue`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.vue) and [`web/admin/src/pages/MapPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.test.ts) for production behavior, plus the required process docs.
- User feedback is specific:
  - the map on `MapPage` looks too small
  - scrolling down exposes a large blank area under the map
  - the fix must keep the right sidebar layout
  - one metric selector must still drive map coloring and list order
  - default window must remain `1d`
- Existing page structure already has:
  - a two-column `map-layout`
  - a right sidebar city list scroll container
  - a fixed global `.chart { height: 360px; }`
- Root cause:
  - the map column is visually capped by the global `360px` chart height
  - the sidebar height is independent, so it can exceed the map height
  - when that happens, the grid reserves the taller sidebar height and leaves blank space below the shorter map

## Options

### Option 1: Minimal height contract inside `MapPage.vue`

- Keep the current page structure and data flow.
- Add a page-local desktop height contract that both columns share.
- Override the page-local chart height so the map viewport is much larger than `360px`.
- Make the sidebar body a flex column so only the city list scrolls inside the available height.

Recommendation: yes. This directly addresses both complaints with the smallest production diff and keeps existing behavior intact.

### Option 2: Increase chart height only

- Override `.chart` to a taller responsive height.
- Leave the sidebar sizing logic unchanged.

Trade-off: improves the small map, but it does not fully solve the height mismatch that creates blank space.

### Option 3: Broader card/layout rewrite

- Split the page into more explicit shells and redesign the panel structure.

Trade-off: could look cleaner, but it exceeds hotfix scope and adds regression risk.

## Chosen Design

### Desktop layout

- Keep the current two-column `map-layout`.
- Introduce a bounded responsive desktop height on the layout, for example through a page-local CSS variable using `clamp(...)`.
- Wrap the chart in a dedicated map shell so the chart can fill the shared layout height.
- Make the sidebar stretch to the same height and reserve the remaining space for the city-list scroller.

### Sidebar scrolling

- Keep the current sidebar header content and ordering.
- Add a dedicated sidebar body wrapper with `display: flex`, `flex-direction: column`, and `min-height: 0`.
- Change `.city-list-scroll` from an independent `max-height` rule to a flexing scroll region that consumes the remaining sidebar height.

### Mobile behavior

- On narrow screens, keep the existing stacked layout.
- Remove the shared desktop height contract on mobile so the page remains natural and does not trap content in a short viewport.
- Keep a sensible, smaller responsive chart height override for mobile.

### Testing

- Add one RED test first in [`web/admin/src/pages/MapPage.test.ts`](/Users/methol/code/github.com/methol/PipeScope/web/admin/src/pages/MapPage.test.ts) that locks the new layout shells used to share height between the map and sidebar.
- Keep the existing regression coverage for:
  - right sidebar layout
  - one metric selector driving coloring and list order
  - default `window=1d`
  - existing geo/tooltip/no-data behavior

## Approval Note

- The user request already specifies the layout direction and acceptable scope, so this design matches the requested hotfix without adding new behavior.
