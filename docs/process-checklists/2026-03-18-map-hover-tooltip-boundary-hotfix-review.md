# Map Hover Tooltip Boundary Hotfix Review

## Scope

- Branch: `fix/map-hover-tooltip-boundary-20260318`
- Reviewed files:
  - `web/admin/src/pages/MapPage.vue`
  - `web/admin/src/pages/MapPage.test.ts`
  - `PROCESS_CHECKLIST.md`
- Review method:
  - manual review against the user-confirmed hotfix requirements and current diff

## Round 1

### Findings

- medium: `web/admin/src/pages/MapPage.vue`
  - the branch base changed the city list from fixed `sortedCityItems.slice(0, 12)` to a Top-based `visibleCityItems` list, which is outside the requested hotfix scope
- medium: `web/admin/src/pages/MapPage.vue`
  - the returned-city hint counted `visibleCityItems` instead of the actual merged API `cityItems`, which weakens the "current window returned city count" explanation

### Fix status

- fixed in current branch
- validation:
  - removed `visibleCityItems`
  - restored the 12-row list rendering
  - changed the meta hint to `当前窗口返回 N 城市（Top X 为上限，不是保底）`

## Round 2

### Findings

- no actionable issues

### Verification focus

- `geo` interaction is no longer suppressed
- province-boundary `lines` overlay remains `silent: true`
- tooltip still shows province/city + conn + bytes with zero fallback
- Top selector options remain `100 / 1000 / 5000 / 10000`
