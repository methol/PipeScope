# 2026-03-18 Map Province Boundary Render

## Stage 1 - Brainstorming

### 问题定义

- 目标问题：修复地图页面“整图变成单一蓝紫色，省界渲染异常”的回归。
- 当前阶段范围：只做 Stage 1 排查设计与证据收敛，不做实现代码改动。
- 当前优先级：先锁定前端渲染链路，重点检查 `v0.1.16` 引入的地图相关改动；除非发现明确证据，否则不把问题扩大到后端聚合、analytics 筛选或 Geo 数据生成流程。
- 非目标：本阶段不重构地图页、不替换整套图表库、不改 sessions/analytics 业务语义。

### 需求边界与验收标准

#### 需求边界

- 修复对象限定在 `web/admin/src/pages/MapPage.vue`、`web/admin/src/pages/mapCity.ts` 及其依赖的 `china-cities.geojson` 渲染使用方式。
- 默认假设数据接口 `/api/map/china` 未发生协议变化；只有在前端渲染面排除后，才继续下钻 API 数据形态。
- 允许调整“省界 overlay”的实现方式、样式或数据来源，但不能顺手回退 `v0.1.16` 已修好的 hover 标签与 city key 冲突问题。

#### 验收标准

- 地图页恢复为“按城市热度着色”的表现，不再出现整图统一蓝紫色覆盖感。
- 省界只表现省级外轮廓，不出现大面积市界误绘、密集线网、断裂或明显错位。
- hover / tooltip 继续显示可读的城市名，不退回到 `440300` 之类 key，也不重新引入海南/湖北/新疆等直管地区的错名问题。
- 无数据区域仍保持中性底色，不能因为省界方案调整让整张底图被强调色统一染色。
- 现有地图相关单测继续通过，并在 Stage 2/3 补足能覆盖真实视觉回归的测试或验证手段。

### 可疑改动面（重点锁定 v0.1.16 相关地图改动）

#### 最高优先级：`7d1e02e feat(map): fix hover labels and add province boundaries`

- `web/admin/src/pages/MapPage.vue`
  - `v0.1.15` 只有单个 `map` series。
  - `v0.1.16` 新增 `geo` 组件、`geoIndex: 0` 绑定，以及一个 `type: 'lines'` 的“省界” overlay。
  - 这是第一处与“省界异常”和“整图视觉被统一染色”直接对应的改动面。
- `web/admin/src/pages/mapCity.ts`
  - `v0.1.16` 新增 `extractProvinceBoundarySegments(features)`，不是加载省级 GeoJSON，而是从市级 GeoJSON 线段反推省界。
  - 该算法依赖同省相邻城市边界线段在 6 位小数精度下完全重合，才能通过偶数计数消掉内部市界。
  - 如果真实数据中的相邻城市边界经过简化后顶点不完全一致，就会把内部市界残留成“省界” overlay。

#### 高优先级：`aae4730 feat(admin): finalize realtime analytics and map closure`

- `web/admin/src/pages/mapCity.ts`
  - `cityKey()` 从旧的 4 位 adcode 前缀切到规范化 6 位 adcode。
  - 这能解释 hover 标签修复，但也改变了地图数据 join 行为；如果数据 join 大面积失配，可能让热度图退化为接近统一底色。
  - 该改动更像“热度数据映射异常”的次级嫌疑，而不是“省界异常”的主因。

#### 证据收敛后的判断

- `git diff --stat v0.1.15..v0.1.16 -- web/admin/src/pages/MapPage.vue web/admin/src/pages/mapCity.ts web/admin/src/pages/MapPage.test.ts` 只显示这 3 个文件变化，没有 GeoJSON 产物变化。
- `git diff --stat v0.1.16..HEAD -- web/admin/src/pages/MapPage.vue web/admin/src/pages/mapCity.ts web/admin/src/pages/MapPage.test.ts` 为空，说明当前主干上的地图实现仍然就是 `v0.1.16` 那组逻辑。
- 因而本次问题优先定位为“前端渲染回归”，不是地图数据文件在 `v0.1.17` 之后再次漂移。

### 最小排查路径

1. 先复现并确认范围不漂移
   - 以当前 `HEAD` 为基线复现问题。
   - 同时确认 map 文件相对 `v0.1.16` 无新增改动，避免把排查范围误扩到无关提交。

2. 首先隔离“省界 overlay”是否为主因
   - 保留 `city_key` / hover label 修复，仅临时移除或置空 `series[1]` 的 `lines` overlay。
   - 如果整图蓝紫感明显消失，说明主因在 `extractProvinceBoundarySegments()` 或 overlay 线样式，而不是 API 数据。

3. 若 overlay 是主因，再检查“由市界反推省界”的算法可靠性
   - 用真实 `china-cities.geojson` 统计 `provinceBoundarySegments` 数量、按省份分布和异常样本。
   - 核查同省相邻城市是否因顶点不完全重合而残留内部市界。
   - 评估是否应改为：
     - 使用真正的省级边界数据源；
     - 或在现有市级数据上做更稳健的拓扑合并，而不是线段精确相消。

4. 如果移除 overlay 后问题仍在，再隔离 `geo` / `geoIndex` 组合
   - 对比 `v0.1.15` 与 `v0.1.16` 的 `geo` 配置差异。
   - 验证是 `geo` 组件叠加导致底图统一着色，还是 `map` series 本身样式/visualMap 行为变化。

5. 只有当前两步都不能解释现象时，再检查 6 位 adcode join
   - 对比 `cityItems.value.length`、过滤后的 `cityData.length`、`mapCityNameByKey.size`。
   - 判断是否存在 API 返回正常、但前端 join 失败导致热度图近似失效的情况。

### 风险与回归点

- 不能粗暴回滚整个 `v0.1.16` 地图提交，否则会丢掉已修复的 hover 标签和直管地区 city key 冲突问题。
- 省界若继续从市级 GeoJSON 反推，真实数据中的边界简化误差可能让问题反复出现，修一次样式但不修数据/算法会留下隐患。
- 如果改动 `cityKey()` 或 join 逻辑，需要回归：
  - 海南直管县级市/县
  - 湖北神农架林区
  - 新疆兵团/直管市
  - 无数据区域 tooltip / label 回退逻辑
- 如果改成新的边界数据源，还要回归构建产物同步链路：`web/admin/public/maps`、`web/dist/maps`、`internal/admin/http/static/maps`。
- 当前测试覆盖的重点是 option 结构和命名正确性，不足以兜住真实视觉回归；Stage 2/3 必须补充更接近真实数据的回归验证。

### 阶段结论

- 第一嫌疑是 `v0.1.16` 新增的“省界 overlay”链路：`extractProvinceBoundarySegments()` + `lines` series。
- 第二嫌疑是同一提交里新增的 `geo` / `geoIndex` 叠加方式，它可能放大了 overlay 的视觉污染，或改变了整图底色表现。
- `cityKey()` 从 4 位切到 6 位是需要保留关注的次级嫌疑，但它更像热度数据映射问题，不是省界异常的主解释。
- Stage 2 的最小动作应先做“关闭 overlay 的对照实验”，而不是一上来重写地图页。

### 关键证据

- 版本面：
  - `v0.1.16` 的地图相关提交直接命名为 `feat(map): fix hover labels and add province boundaries`（`7d1e02e`）。
  - 当前 `HEAD` 对 map 文件相对 `v0.1.16` 无差异，问题源头仍锁在该版本逻辑。
- 代码面：
  - `MapPage.vue` 在 `v0.1.16` 新增 `geo` 和 `series[1] = lines`。
  - `mapCity.ts` 在 `v0.1.16` 新增省界提取算法，且算法输入是市级 GeoJSON，不是省级边界数据。
- 数据面：
  - `web/admin/public/maps/china-cities.geojson` 当前只有 370 个 feature。
  - 基于当前算法统计，真实数据会产出约 770 条省界 overlay 线段，说明“省界”是由大量市级线段拼出来的，而不是独立省界源。
- 测试面：
  - `npm test -- --run src/pages/mapCity.test.ts src/pages/MapPage.test.ts` 当前为 PASS（15 tests）。
  - 现有测试只能证明配置结构和命名逻辑通过，无法证明真实地图不会出现“整图蓝紫化/省界线网化”的视觉回归。
