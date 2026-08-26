package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Store 封装 SQLite 连接与建表迁移，是全部持久化的唯一入口。
type Store struct {
	DB *sql.DB
}

// scanner 兼容 *sql.Row 与 *sql.Rows 的 Scan 方法。
type scanner interface {
	Scan(dest ...interface{}) error
}

// Open 打开（必要时创建）SQLite 数据库并执行迁移。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者模型：限制单连接可避免 "database is locked"，
	// 并行导入会在连接层串行化，仍保持正确性与原子性。
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.DB.Close()
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS samples (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			compound TEXT NOT NULL,
			solvent TEXT NOT NULL,
			concentration REAL NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS temperature_points (
			id TEXT PRIMARY KEY,
			sample_id TEXT NOT NULL,
			temp_c REAL NOT NULL,
			sort_order INTEGER NOT NULL,
			UNIQUE(sample_id, temp_c)
		)`,
		`CREATE TABLE IF NOT EXISTS spectrum_batches (
			id TEXT PRIMARY KEY,
			sample_id TEXT NOT NULL,
			label TEXT NOT NULL,
			state TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS peaks (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			temperature_id TEXT NOT NULL,
			temp_c REAL NOT NULL,
			unit TEXT NOT NULL,
			observed_shift REAL NOT NULL,
			corrected_shift REAL NOT NULL,
			intensity REAL NOT NULL,
			width_hz REAL NOT NULL,
			is_standard INTEGER NOT NULL DEFAULT 0,
			state TEXT NOT NULL,
			note TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS internal_standards (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			name TEXT NOT NULL,
			locked INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS internal_standard_points (
			id TEXT PRIMARY KEY,
			standard_id TEXT NOT NULL,
			temp_c REAL NOT NULL,
			true_shift REAL NOT NULL,
			UNIQUE(standard_id, temp_c)
		)`,
		`CREATE TABLE IF NOT EXISTS peak_tracks (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			label TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS peak_track_members (
			id TEXT PRIMARY KEY,
			track_id TEXT NOT NULL,
			peak_id TEXT NOT NULL,
			temp_c REAL NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS exchange_candidates (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			track_a_id TEXT NOT NULL,
			track_b_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			score REAL NOT NULL,
			state TEXT NOT NULL,
			reason TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS assignment_snapshots (
			id TEXT PRIMARY KEY,
			batch_id TEXT NOT NULL,
			state TEXT NOT NULL,
			frozen_at TEXT NOT NULL,
			payload TEXT NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
