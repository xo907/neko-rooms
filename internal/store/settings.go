package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// GetSetting decodes a JSON setting into v. If the setting does not exist,
// v is left untouched (so callers can pre-fill defaults) and ErrNotFound is returned.
func (s *Store) GetSetting(key string, v any) error {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), v)
}

func (s *Store) SetSetting(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, string(raw), time.Now().UTC(),
	)
	return err
}

type Asset struct {
	Name      string    `json:"name"`
	Mime      string    `json:"mime"`
	Data      []byte    `json:"-"`
	Hash      string    `json:"hash"`
	Size      int       `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) GetAsset(name string) (*Asset, error) {
	a := &Asset{}
	err := s.db.QueryRow(`SELECT name, mime, data, hash, updated_at FROM assets WHERE name = ?`, name).
		Scan(&a.Name, &a.Mime, &a.Data, &a.Hash, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	a.Size = len(a.Data)
	return a, err
}

func (s *Store) ListAssets() ([]*Asset, error) {
	rows, err := s.db.Query(`SELECT name, mime, hash, length(data), updated_at FROM assets ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := []*Asset{}
	for rows.Next() {
		a := &Asset{}
		if err := rows.Scan(&a.Name, &a.Mime, &a.Hash, &a.Size, &a.UpdatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (s *Store) PutAsset(name, mime string, data []byte) (*Asset, error) {
	sum := sha256.Sum256(data)
	a := &Asset{
		Name:      name,
		Mime:      mime,
		Data:      data,
		Hash:      hex.EncodeToString(sum[:8]),
		Size:      len(data),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO assets (name, mime, data, hash, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET mime = excluded.mime, data = excluded.data, hash = excluded.hash, updated_at = excluded.updated_at`,
		a.Name, a.Mime, a.Data, a.Hash, a.UpdatedAt,
	)
	return a, err
}

func (s *Store) DeleteAsset(name string) error {
	_, err := s.db.Exec(`DELETE FROM assets WHERE name = ?`, name)
	return err
}

type AuditEntry struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    *int64    `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Details   string    `json:"details"`
	IP        string    `json:"ip"`
}

func (s *Store) AddAudit(e *AuditEntry) error {
	_, err := s.db.Exec(
		`INSERT INTO audit_log (created_at, user_id, username, action, target, details, ip) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		time.Now().UTC(), e.UserID, e.Username, e.Action, e.Target, e.Details, e.IP,
	)
	return err
}

func (s *Store) ListAudit(limit, offset int) ([]*AuditEntry, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`SELECT id, created_at, user_id, username, action, target, details, ip FROM audit_log ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := []*AuditEntry{}
	for rows.Next() {
		e := &AuditEntry{}
		var uid sql.NullInt64
		if err := rows.Scan(&e.ID, &e.CreatedAt, &uid, &e.Username, &e.Action, &e.Target, &e.Details, &e.IP); err != nil {
			return nil, 0, err
		}
		if uid.Valid {
			e.UserID = &uid.Int64
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// PruneAudit keeps only the newest `keep` entries.
func (s *Store) PruneAudit(keep int) error {
	_, err := s.db.Exec(`DELETE FROM audit_log WHERE id NOT IN (SELECT id FROM audit_log ORDER BY id DESC LIMIT ?)`, keep)
	return err
}
