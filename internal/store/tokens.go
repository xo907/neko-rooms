package store

import (
	"database/sql"
	"errors"
	"time"
)

// APIToken lets scripts use the API as a user, sent as a bearer token.
// Only a hash of the token is kept.
type APIToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	Hint       string     `json:"hint"` // the last characters, to tell tokens apart
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

const apiTokenColumns = `id, user_id, name, hint, created_at, last_used_at, expires_at`

func scanAPIToken(row scanner) (*APIToken, error) {
	t := &APIToken{}
	var used, expires sql.NullTime
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.Hint, &t.CreatedAt, &used, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if used.Valid {
		v := used.Time.UTC()
		t.LastUsedAt = &v
	}
	if expires.Valid {
		v := expires.Time.UTC()
		t.ExpiresAt = &v
	}
	return t, nil
}

func (s *Store) CreateAPIToken(t *APIToken, hash string) error {
	t.CreatedAt = time.Now().UTC()
	var expires any
	if t.ExpiresAt != nil {
		expires = t.ExpiresAt.UTC()
	}
	res, err := s.db.Exec(
		`INSERT INTO api_tokens (user_id, name, hint, hash, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.UserID, t.Name, t.Hint, hash, t.CreatedAt, expires,
	)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *Store) GetAPITokenByHash(hash string) (*APIToken, error) {
	return scanAPIToken(s.db.QueryRow(`SELECT `+apiTokenColumns+` FROM api_tokens WHERE hash = ?`, hash))
}

func (s *Store) ListAPITokens(userID int64) ([]*APIToken, error) {
	rows, err := s.db.Query(`SELECT `+apiTokenColumns+` FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*APIToken{}
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (s *Store) TouchAPIToken(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, at.UTC(), id)
	return err
}

// DeleteAPIToken removes a token of a user.
func (s *Store) DeleteAPIToken(userID, id int64) error {
	res, err := s.db.Exec(`DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
