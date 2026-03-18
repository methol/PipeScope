# 2026-03-18 Map Static Assets Sync Hotfix

## Stage 1 - Brainstorming
- 现象：线上 `v0.1.18` 仍出现地图整图蓝紫色/省界异常线。
- 根因：源码已移除省界 overlay，但嵌入静态资源 `internal/admin/http/static/assets/index-*.js` 仍是旧 bundle，继续包含 `省界 lines` 逻辑。
- 最小修复：重建前端并同步 `web/dist` 与 `internal/admin/http/static`，重新发版上线。

## Stage 2 - Plan
1. 执行 `make build-web sync-web` 重建并同步静态资产。
2. 用关键字检索确认新 bundle 不再包含 `省界/lines/provinceBoundary` 相关逻辑。
3. 运行前端回归测试（MapPage/mapCity）。
4. 提交热修 PR，合并并发布新版本。

## Stage 3 - Execute
- 已执行：`make build-web sync-web`
- 结果：`web/dist` 与 `internal/admin/http/static` 均替换为新 hash 资产（`index-B-6neUZ5.js`）。
- 检查：对新 bundle 检索 `省界|type:"lines"|provinceBoundaryData|extractProvinceBoundarySegments`，无命中。

## Stage 4 - Review
- 审核结论：问题由构建产物未同步导致，不是运行时逻辑回归；修复方式正确且最小。
- 未发现新的代码级回归风险。

## Stage 5 - Verify
- `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts` -> PASS (16 tests)

## Risk
- 仅替换静态 bundle，风险集中在前端资源刷新与缓存；服务端逻辑无改动。

## Next
- 提交 PR -> 合并 -> 发布 `v0.1.19` -> 线上升级并验证。
