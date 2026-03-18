# PROCESS_CHECKLIST

## Task
地图修复热更：同步嵌入静态资产，消除旧省界 overlay bundle。

## Branch
`fix/map-static-assets-sync-20260318`

## Stage Status
- Stage1 brainstorming: DONE
- Stage2 writing-plans: DONE
- Stage3 executing-plans: DONE
- Stage4 requesting-code-review: DONE
- Stage5 receiving-code-review: DONE

## Verification
- `make build-web sync-web` PASS
- 关键字检查新 bundle 不含旧 overlay 逻辑：PASS
- `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts` PASS

## Final
- Status: LOCAL_VERIFIED_READY_FOR_PR
