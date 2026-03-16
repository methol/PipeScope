package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) InitSchema(ctx context.Context) error {
	if err := s.migrateLegacyConnEvents(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, schemaSQL)
	return err
}

func (s *Store) migrateLegacyConnEvents(ctx context.Context) error {
	tableExists, err := s.hasTable(ctx, "conn_events")
	if err != nil {
		return err
	}
	if !tableExists {
		return nil
	}

	required := []struct {
		name string
		ddl  string
	}{
		{name: "listen_port", ddl: "ALTER TABLE conn_events ADD COLUMN listen_port INTEGER NOT NULL DEFAULT 0"},
		{name: "src_addr", ddl: "ALTER TABLE conn_events ADD COLUMN src_addr TEXT NOT NULL DEFAULT ''"},
		{name: "src_ip", ddl: "ALTER TABLE conn_events ADD COLUMN src_ip TEXT NOT NULL DEFAULT ''"},
		{name: "dst_addr", ddl: "ALTER TABLE conn_events ADD COLUMN dst_addr TEXT NOT NULL DEFAULT ''"},
		{name: "dst_host", ddl: "ALTER TABLE conn_events ADD COLUMN dst_host TEXT NOT NULL DEFAULT ''"},
		{name: "dst_port", ddl: "ALTER TABLE conn_events ADD COLUMN dst_port INTEGER NOT NULL DEFAULT 0"},
		{name: "start_ts", ddl: "ALTER TABLE conn_events ADD COLUMN start_ts INTEGER NOT NULL DEFAULT 0"},
		{name: "end_ts", ddl: "ALTER TABLE conn_events ADD COLUMN end_ts INTEGER NOT NULL DEFAULT 0"},
		{name: "duration_ms", ddl: "ALTER TABLE conn_events ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 0"},
		{name: "up_bytes", ddl: "ALTER TABLE conn_events ADD COLUMN up_bytes INTEGER NOT NULL DEFAULT 0"},
		{name: "down_bytes", ddl: "ALTER TABLE conn_events ADD COLUMN down_bytes INTEGER NOT NULL DEFAULT 0"},
		{name: "total_bytes", ddl: "ALTER TABLE conn_events ADD COLUMN total_bytes INTEGER NOT NULL DEFAULT 0"},
		{name: "err_msg", ddl: "ALTER TABLE conn_events ADD COLUMN err_msg TEXT NOT NULL DEFAULT ''"},
		{name: "blocked_reason", ddl: "ALTER TABLE conn_events ADD COLUMN blocked_reason TEXT NOT NULL DEFAULT ''"},
		{name: "country", ddl: "ALTER TABLE conn_events ADD COLUMN country TEXT NOT NULL DEFAULT ''"},
		{name: "province", ddl: "ALTER TABLE conn_events ADD COLUMN province TEXT NOT NULL DEFAULT ''"},
		{name: "city", ddl: "ALTER TABLE conn_events ADD COLUMN city TEXT NOT NULL DEFAULT ''"},
		{name: "adcode", ddl: "ALTER TABLE conn_events ADD COLUMN adcode TEXT NOT NULL DEFAULT ''"},
		{name: "lat", ddl: "ALTER TABLE conn_events ADD COLUMN lat REAL NOT NULL DEFAULT 0"},
		{name: "lng", ddl: "ALTER TABLE conn_events ADD COLUMN lng REAL NOT NULL DEFAULT 0"},
		{name: "created_at", ddl: "ALTER TABLE conn_events ADD COLUMN created_at INTEGER NOT NULL DEFAULT 0"},
	}

	for _, col := range required {
		exists, err := s.hasColumn(ctx, "conn_events", col.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("migrate conn_events add column %s: %w", col.name, err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE conn_events
SET created_at = CASE
	WHEN end_ts > 0 THEN end_ts
	WHEN start_ts > 0 THEN start_ts
	ELSE created_at
END
WHERE created_at = 0
`); err != nil {
		return fmt.Errorf("backfill conn_events created_at: %w", err)
	}
	return nil
}

func (s *Store) hasTable(ctx context.Context, table string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) hasColumn(ctx context.Context, table, column string) (bool, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}
