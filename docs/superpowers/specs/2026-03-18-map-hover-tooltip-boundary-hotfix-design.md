# Map Hover Tooltip Boundary Hotfix Design

## Context

- User request time: 2026-03-18 19:41 GMT+8
- Scope: `web/admin/src/pages/MapPage.vue` and related front-end tests
- Confirmed background:
  - `limit=1000` only means upper bound; current data window can legitimately return fewer cities
  - Current complaint is focused on hover/tooltip not responding and province boundaries being hard to see

## Approaches

### Option 1: Minimal fix on current `geo + map + lines` structure

- Re-enable map hover by removing the `geo` interaction suppression
- Keep the existing province-boundary `lines` series and only strengthen its style/layering
- Add a meta line that states the actual returned city count in the current window

Recommendation: yes. It matches the user's "only these three changes" requirement and avoids backend or data-pipeline churn.

### Option 2: Remove the `lines` overlay again

- This would reduce visual noise quickly
- Rejected because the user explicitly wants to keep the current `lines` approach and make boundaries more visible

### Option 3: Replace the overlay with a new province-level GeoJSON source

- This could produce cleaner province outlines
- Rejected for this hotfix because it changes assets/build flow and exceeds the "no new complex configuration" constraint

## Design

### Interaction

- Keep the current `map` series tooltip and emphasis label formatter logic
- Change `geo` so it no longer suppresses hover/tooltip interaction
- Keep the province-boundary `lines` series `silent: true` so the overlay does not steal mouse events from city regions

Approval basis: the user already specified the exact interaction outcome and acceptable implementation boundary, so no further design clarification is required before implementation.

### Boundary visibility

- Preserve `extractProvinceBoundarySegments(...)`
- Increase the visual contrast of the `lines` series through color, width, and z-order only
- Do not change backend semantics, map asset semantics, or selector options

### Returned-city explanation

- Add a dedicated meta line on the page
- Format: `当前窗口返回 N 城市（Top X 为上限，不是保底）`
- `N` comes from the merged `cityItems` list after current-window API data is loaded

### Testing

- Add a RED test that proves `geo.silent` and `geo.emphasis.disabled` currently suppress the desired interaction
- Add a RED test that locks the stronger boundary style contract
- Add a RED test for the returned-city meta text
- Keep existing tooltip and no-data tests to ensure province/city + conn + bytes stay intact
