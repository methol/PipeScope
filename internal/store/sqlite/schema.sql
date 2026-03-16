CREATE TABLE IF NOT EXISTS conn_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id TEXT NOT NULL,
    listen_port INTEGER NOT NULL DEFAULT 0,
    src_addr TEXT NOT NULL DEFAULT '',
    src_ip TEXT NOT NULL DEFAULT '',
    dst_addr TEXT NOT NULL DEFAULT '',
    dst_host TEXT NOT NULL DEFAULT '',
    dst_port INTEGER NOT NULL DEFAULT 0,
    start_ts INTEGER NOT NULL DEFAULT 0,
    end_ts INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    up_bytes INTEGER NOT NULL DEFAULT 0,
    down_bytes INTEGER NOT NULL DEFAULT 0,
    total_bytes INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ok',
    err_msg TEXT NOT NULL DEFAULT '',
    blocked_reason TEXT NOT NULL DEFAULT '',
    province TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    adcode TEXT NOT NULL DEFAULT '',
    lat REAL NOT NULL DEFAULT 0,
    lng REAL NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
);

CREATE INDEX IF NOT EXISTS idx_conn_events_start_ts ON conn_events(start_ts);
CREATE INDEX IF NOT EXISTS idx_conn_events_rule_id ON conn_events(rule_id);
CREATE INDEX IF NOT EXISTS idx_conn_events_adcode ON conn_events(adcode);
CREATE INDEX IF NOT EXISTS idx_conn_events_province_city ON conn_events(province, city);
CREATE INDEX IF NOT EXISTS idx_conn_events_status ON conn_events(status);
CREATE INDEX IF NOT EXISTS idx_conn_events_blocked_reason ON conn_events(blocked_reason);

CREATE TABLE IF NOT EXISTS dim_adcode (
    adcode TEXT PRIMARY KEY,
    province TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    district TEXT NOT NULL DEFAULT '',
    lat REAL NOT NULL DEFAULT 0,
    lng REAL NOT NULL DEFAULT 0,
    normalized_province TEXT NOT NULL DEFAULT '',
    normalized_city TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_dim_adcode_norm_pc ON dim_adcode(normalized_province, normalized_city);

CREATE TABLE IF NOT EXISTS app_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
