package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// migrations are applied in order, each exactly once
var migrations = []string{
	// 1: initial schema
	`
	CREATE TABLE users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
		display_name  TEXT NOT NULL DEFAULT '',
		email         TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		role          TEXT NOT NULL DEFAULT 'user',
		disabled      INTEGER NOT NULL DEFAULT 0,
		room_limit    INTEGER NOT NULL DEFAULT -1,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_login_at DATETIME
	);

	CREATE TABLE sessions (
		id           TEXT PRIMARY KEY,
		user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at   DATETIME NOT NULL,
		ip           TEXT NOT NULL DEFAULT '',
		user_agent   TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX sessions_user_id ON sessions(user_id);

	CREATE TABLE settings (
		key        TEXT PRIMARY KEY,
		value      TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE assets (
		name       TEXT PRIMARY KEY,
		mime       TEXT NOT NULL,
		data       BLOB NOT NULL,
		hash       TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE audit_log (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		user_id    INTEGER,
		username   TEXT NOT NULL DEFAULT '',
		action     TEXT NOT NULL,
		target     TEXT NOT NULL DEFAULT '',
		details    TEXT NOT NULL DEFAULT '',
		ip         TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX audit_log_created_at ON audit_log(created_at);
	`,
	// 2: room metadata & friends
	`
	CREATE TABLE room_meta (
		name        TEXT PRIMARY KEY,
		owner_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
		visibility  TEXT NOT NULL DEFAULT 'private',
		title       TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		category    TEXT NOT NULL DEFAULT '',
		featured    INTEGER NOT NULL DEFAULT 0,
		hidden      INTEGER NOT NULL DEFAULT 0,
		invite_code TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX room_meta_owner ON room_meta(owner_id);

	CREATE TABLE friendships (
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		friend_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status     TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, friend_id)
	);
	CREATE INDEX friendships_friend ON friendships(friend_id);
	`,
}

type Store struct {
	db *sql.DB
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("unable to create data dir: %w", err)
	}

	dsn := "file:" + filepath.Join(dataDir, "neko-rooms.db") +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// sqlite allows only a single writer
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to migrate database: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}

	for i := version; i < len(migrations); i++ {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
