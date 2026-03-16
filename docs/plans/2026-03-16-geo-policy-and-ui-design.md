# Geo Policy 前置拦截与 UI 改进设计

## 0. 核心设计决策

### Q1: 配置应该按国家码/省市码，还是名称，还是双轨？

**决策：双轨（编码 + 名称），优先编码，名称作为补充。**

| 字段 | 推荐方式 | 原因 |
|------|----------|------|
| 国家 | **国家码（ISO 3166-1 alpha-2）** | ip2region 直接返回，无歧义，如 `CN`、`US` |
| 省/市 | **名称** | ip2region 返回名称，国内库以中文名为主 |
| 精确匹配 | **行政区划码（adcode）** | 最精确，避免同名歧义，如 `110000`（北京）、`310000`（上海） |

**配置字段设计：**
```yaml
rules:
  - country: "CN"                    # 国家码（必填）
    provinces: ["北京", "上海"]       # 省名（可选，名称匹配）
    cities: ["海淀"]                 # 市名（可选，名称匹配）
    adcodes: ["110108", "310101"]    # 行政区划码（可选，最精确）
```

**选择理由：**
- 国家码：ip2region 返回标准 ISO 代码，直接使用无转换成本
- 省/市名：ip2region 返回中文名称，配置直接使用；名称经 `normalize` 标准化（去"省"、"市"后缀）
- adcode：最精确，但仅国内 IP 可匹配（国外 IP 无 adcode）；适用于需要精确到区县级的场景

### Q2: 多级从属关系（国家>省>市）如何表达？

**决策：层级嵌套 + 扁平列表结合。**

```yaml
# 方式一：层级嵌套（直观，但重复多）
rules:
  - country: "CN"
    provinces: ["北京"]
    cities: ["海淀"]
  - country: "CN"
    provinces: ["上海"]

# 方式二：扁平列表（推荐，简洁）
rules:
  - country: "CN"
    provinces: ["北京", "上海"]    # 同国家多省
  - country: "CN"
    adcodes: ["110108", "310101"]  # 同国家多城市（精确码）
```

**语义说明：**
- `provinces` 与 `cities` 是 **AND 关系**：必须同时匹配
- `provinces`、`cities`、`adcodes` 之间是 **OR 关系**：任一匹配即可
- `country` 是必填，作为过滤的前提条件

**匹配逻辑（伪代码）：**
```
match = country_match(country)
if match && provinces:
    match = province_in(provinces)
    if match && cities:
        match = city_in(cities) && province_in(provinces)
if match && adcodes:
    match = adcode_in(adcodes)  # 最高优先级
```

### Q3: 匹配优先级怎么定（deny/allow/default）？

**决策：deny > allow > default（默认放行）。**

| 模式 | 命中规则 | 未命中规则 |
|------|----------|------------|
| `allow` + `require_allow_hit: false` | 放行 | 放行（默认放行） |
| `allow` + `require_allow_hit: true` | 放行 | **拒绝**（白名单模式） |
| `deny` | **拒绝** | 放行（黑名单模式） |

**规则内优先级：**
1. `adcodes` 匹配（最精确，优先）
2. `cities` + `provinces` 匹配
3. `provinces` 匹配
4. `country` 匹配（最宽泛）

**多规则匹配：** 首条匹配生效，规则顺序重要。

### Q4: 如何覆盖典型场景？

#### 场景 A：不允许国外流量
```yaml
geo_policy:
  mode: "allow"
  require_allow_hit: true      # 必须命中才放行
  rules:
    - country: "CN"            # 仅允许中国
```

#### 场景 B：禁止某省/某市
```yaml
geo_policy:
  mode: "deny"
  rules:
    - country: "CN"
      provinces: ["福建"]      # 禁止福建
    - country: "CN"
      adcodes: ["440300"]      # 禁止深圳（精确码）
```

#### 场景 C：仅允许白名单城市
```yaml
geo_policy:
  mode: "allow"
  require_allow_hit: true
  rules:
    - country: "CN"
      adcodes: ["110000", "310000", "440100"]  # 仅北京、上海、广州
```

#### 场景 D：允许中国 + 禁止某省
```yaml
geo_policy:
  mode: "deny"
  rules:
    - country: "CN"
      provinces: ["福建", "广东"]  # 禁止福建、广东
# 未命中的 CN 流量放行，非 CN 流量也放行（deny 模式特性）
```

#### 场景 E：仅允许中国，但禁止某省
```yaml
# 需要两条规则组合
geo_policy:
  mode: "allow"
  require_allow_hit: true
  rules:
    - country: "CN"                    # 允许中国
geo_policy_deny:                       # 扩展字段（未实现，建议后续）
  rules:
    - country: "CN"
      provinces: ["福建"]              # 禁止福建
```

**当前限制：** 单一 policy 仅支持单一模式。如需"允许中国但禁止某省"，需要：
- 方案 1：配置两个 proxy_rule（不同端口）
- 方案 2：后续扩展支持 `allow_rules` + `deny_rules` 双列表

---

## IP 库字段映射与边界

### ip2region 返回字段

| 字段 | 类型 | 示例 | 说明 |
|------|------|------|------|
| `Country` | string | `"CN"`, `"US"` | ISO 3166-1 alpha-2 国家码 |
| `Province` | string | `"北京"`, `"广东省"` | 省份名称（可能带后缀） |
| `City` | string | `"北京"`, `"深圳"` | 城市名称 |
| `ISP` | string | `"电信"` | 运营商（当前未使用） |
| `Code` | string | `"CN"` | 国家码（与 Country 相同） |

### areacity 匹配字段

| 字段 | 类型 | 示例 | 说明 |
|------|------|------|------|
| `Adcode` | string | `"110000"` | 6位行政区划码 |
| `Province` | string | `"北京"` | 标准化省名 |
| `City` | string | `"北京"` | 标准化市名 |
| `Lat`/`Lng` | float | `39.9042, 116.4074` | 经纬度 |

### 能力边界

| 能保证 | 不能保证 |
|--------|----------|
| 国家码准确（ip2region） | 省市名称可能有歧义（如"吉林"省市同名） |
| 国内 IP 有 adcode | 国外 IP 无 adcode |
| 省市名标准化后匹配 | 区县级精度依赖 adcode |
| 名称匹配经 normalize | 名称可能为空（内网 IP） |

### GeoInfo 最终字段

```go
type GeoInfo struct {
    Country  string  // 国家码 (CN, US, ...)
    Province string  // 标准化省名 (北京, 广东, ...)
    City     string  // 标准化市名 (北京, 深圳, ...)
    Adcode   string  // 行政区划码 (仅国内 IP 有值)
}
```

---

## 1. 需求概述

### 1.1 Geo 前置拦截策略
- 在连接进入后、转发前执行 geo 判定
- 支持 allow/deny 两种模式
- 支持多级从属关系（国家 > 省 > 市）
- 被拦截时记录状态与原因

### 1.2 前端/后端 UI 改进
- 地图页面字节显示人类友好
- 地图页面参考统计页面接口口径
- 分析/地图页面支持 limit 选择
- 实时页面 rule 下拉筛选

## 2. 数据模型设计

### 2.1 Geo Policy 配置结构

```go
// GeoPolicy 定义 geo 拦截策略
type GeoPolicy struct {
    Mode           string       `yaml:"mode"`             // "allow" | "deny"
    RequireAllowHit bool        `yaml:"require_allow_hit"` // 白名单模式下是否必须命中
    Rules           []GeoRule   `yaml:"rules"`            // 规则列表
}

// GeoRule 定义单条 geo 规则
type GeoRule struct {
    Country   string   `yaml:"country"`    // 国家码 (ISO 3166-1 alpha-2)
    Provinces []string `yaml:"provinces"`  // 省列表（可选）
    Cities    []string `yaml:"cities"`     // 城市列表（可选）
    Adcodes   []string `yaml:"adcodes"`    // 行政区划码列表（可选）
}
```

### 2.2 配置示例

```yaml
proxy_rules:
  - id: "demo-rule"
    listen: "0.0.0.0:10001"
    forward: "127.0.0.1:10002"
    geo_policy:                           # 可选
      mode: "allow"                       # allow=白名单, deny=黑名单
      require_allow_hit: true             # 必须命中 allow 才放行
      rules:
        - country: "CN"                   # 允许中国流量
        - country: "US"                   # 允许美国流量
          provinces: ["California"]       # 仅加州
```

### 2.3 匹配优先级

1. **精确匹配优先**：adcode > city+province > province > country
2. **规则顺序**：先配置的规则先匹配
3. **模式行为**：
   - `allow` 模式：命中规则则放行，未命中取决于 `require_allow_hit`
   - `deny` 模式：命中规则则拒绝，未命中则放行

### 2.4 典型场景

| 场景 | 配置 |
|------|------|
| 不允许国外流量 | `mode: allow, require_allow_hit: true, rules: [{country: "CN"}]` |
| 禁止某省/某市 | `mode: deny, rules: [{country: "CN", provinces: ["某省"]}]` |
| 仅允许白名单城市 | `mode: allow, require_allow_hit: true, rules: [{country: "CN", adcodes: ["110000", "310000"]}]` |

## 3. 数据库变更

### 3.1 conn_events 表新增字段

```sql
ALTER TABLE conn_events ADD COLUMN blocked_reason TEXT NOT NULL DEFAULT '';
-- blocked_reason 取值：'' (未拦截), 'geo_denied', 'geo_not_in_allowlist'
```

### 3.2 索引优化

```sql
CREATE INDEX IF NOT EXISTS idx_conn_events_blocked_reason ON conn_events(blocked_reason);
```

## 4. API 变更

### 4.1 地图 API 增强

```
GET /api/map/china?window=1h&metric=conn&limit=100
```

新增 `limit` 参数，控制返回点数量。

### 4.2 分析 API 增强

```
GET /api/analytics?window=1d&top_n=100
```

`top_n` 参数支持 10/50/100/1000。

### 4.3 实时会话 options API

```
GET /api/sessions/options?window=15m
```

返回可用的 rule_id 列表，供下拉选择使用。

## 5. 前端变更

### 5.1 MapPage.vue
- 字节数使用 `formatBytes` 格式化
- 新增 limit 选择器（10/50/100/1000）
- tooltip 和列表显示友好字节格式

### 5.2 AnalyticsPage.vue
- limit 选择器（10/50/100/1000）

### 5.3 SessionsPage.vue
- rule_id 改为下拉选择
- 新增 `/api/sessions/options` 调用

## 6. 核心执行流程

```
连接进入
    ↓
获取客户端 IP
    ↓
Geo 查询（ip2region）
    ↓
Geo Policy 匹配
    ↓
命中 deny 规则? ──→ blocked_reason='geo_denied' → 写入事件 → 关闭连接
    ↓ 否
allow 模式 + require_allow_hit + 未命中? ──→ blocked_reason='geo_not_in_allowlist' → 写入事件 → 关闭连接
    ↓ 否
正常转发
```

## 7. 文件改动清单

### 后端
- `internal/config/config.go` - GeoPolicy 配置结构
- `internal/gateway/rule/rule.go` - Rule 扩展
- `internal/gateway/geo/policy.go` - Geo 策略匹配逻辑（新增）
- `internal/gateway/session/session.go` - blocked_reason 字段
- `internal/gateway/proxy/runner.go` - 注入 geo 检查
- `internal/store/sqlite/schema.sql` - blocked_reason 字段
- `internal/store/sqlite/writer.go` - 写入 blocked_reason
- `internal/admin/service/query.go` - limit 参数
- `internal/admin/http/handlers.go` - limit 参数处理
- `cmd/pipescope/main.go` - 传递 geo policy

### 前端
- `web/admin/src/pages/MapPage.vue` - formatBytes + limit
- `web/admin/src/pages/AnalyticsPage.vue` - limit 选择器
- `web/admin/src/pages/SessionsPage.vue` - rule 下拉
- `web/admin/src/api/client.ts` - limit 参数 + options API

### 测试
- `internal/gateway/geo/policy_test.go` - geo 策略匹配测试
- `internal/config/config_test.go` - 配置解析测试
- 后端/前端相关测试更新

## 8. 风险与后续

### 风险
1. ip2region 国家码可能不一致（需要标准化）
2. 性能影响：每次连接增加一次 IP 查询（已有缓存机制）
3. 向后兼容：geo_policy 为可选配置

### 后续建议
1. 支持 geo 数据库热更新
2. 支持按 rule 配置不同的 geo policy
3. 管理 UI 配置 geo policy
