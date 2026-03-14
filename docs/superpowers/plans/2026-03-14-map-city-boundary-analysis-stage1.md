# 阶段1设计：地图市级边界 + 实时/统计页面拆分

> use superpower: brainstorming skill

## 1) 数据模型 / API 变更清单

### 1.1 数据模型（后端 service DTO）
- `MapPoint` 继续以 city 维度作为基础，不再引入 county/district 维度可视化字段。
- 新增（或约定）前端使用键：`city_key = province + "-" + city`（后端可不改字段，前端拼接）。

### 1.2 现有 API 的语义调整（兼容优先）
- `GET /api/map/china?window=&metric=`
  - 保持接口不变，语义明确为：**按 city 聚合**。
- `GET /api/sessions?window=&rule_id=&limit=&offset=`
  - 继续保留：用于“实时”与“统计”页面的明细/检索基础数据源。

### 1.3 新增 API（建议）
- `GET /api/analytics/sessions`
  - 参数：`window, rule_id, province, city, status, limit, offset`
  - 返回：明细列表（与 sessions 类似）
- `GET /api/analytics/summary`
  - 参数：`window, rule_id, province, city, status`
  - 返回：`conn_count, total_bytes, avg_duration_ms, active_rules, active_cities`
- `GET /api/analytics/group-by`
  - 参数：`window, group=rule|city|province|status, metric=conn|bytes, ...filters`
  - 返回：聚合桶列表

> 本次实施可先采用“前端基于 /api/sessions 做聚合计算”的过渡方案，后续平滑切后端聚合 API。

---

## 2) 前端页面结构变更（Map / 实时 / 统计）

### 2.1 Map 页面（分析型）
- 改为加载 `china-cities.geojson`（仅市级边界）。
- 移除自动刷新（取消 5s interval）。
- 保留手动刷新按钮。
- tooltip / hover / 高亮均以 city 命中：显示“城市名 + 指标值”。

### 2.2 “明细”页重命名为“实时”
- Tab 名称：`明细` -> `实时`。
- 去掉“时间窗口”下拉，固定为短窗（例如 5m）。
- 保留自动刷新（实时查看定位）。

### 2.3 新增“统计/分析”页面
- 支持更长时间范围（1h/6h/1d/7d/30d）。
- 默认不自动刷新（分析型页面）。
- 支持条件检索（rule/province/city/status）。
- 支持聚合展示（总览 + top 城市 + top 规则）。

---

## 3) 市级边界数据来源与转换方案

### 数据来源
- 继续使用仓库已有 `data/ok_geo.csv`（稳定、本地可离线构建）。

### 转换规则（离线预处理）
- 过滤 `deep=1` 行（市级边界）。
- 排除港澳台或非主数据层级（与现有生成逻辑一致）。
- `polygon` -> GeoJSON MultiPolygon。
- 可选简化（Douglas-Peucker）+ 精度截断。
- 产出文件：`web/admin/public/maps/china-cities.geojson`。

### 前端加载约束
- 地图页面仅加载 `china-cities.geojson`。
- 不再加载 county layer 文件用于可视化。

---

## 4) 迁移步骤（可回滚）

1. 新增城市 GeoJSON 生成与产物文件（不替换旧县区文件）。
2. Map 页面切换到城市边界渲染；保留旧文件在仓库中。
3. App 导航改造：`实时` + `统计` 新页面并存。
4. 若线上出现异常：
   - 前端回滚到旧 MapPage（county）版本；
   - API 保持兼容，无需 DB 迁移；
   - GeoJSON 文件保留双份，不影响回滚。

---

## 5) 验收标准

1. 地图放大后仅看到市级边界，无区县边界。
2. 鼠标悬停时高亮单个市，并显示城市名（非省名）。
3. Map 页面默认不自动刷新，仅手动刷新触发数据更新。
4. 原“明细”页改名“实时”，去掉时间窗口选择，保持自动刷新。
5. 存在独立“统计/分析”页面：
   - 可选长时间窗口；
   - 不自动刷新；
   - 支持条件检索；
   - 有聚合结果展示。
6. 回归通过：规则页、会话 API、构建流程不受破坏。
