package store

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	ID         string    `json:"id"` // sha256 of the cookie token, never the token itself
	UserID     int64     `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
}

const sessionColumns = `id, user_id, created_at, last_seen_at, expires_at, ip, user_agent`

func scanSession(row scanner) (*Session, error) {
	s := &Session{}
	err := row.Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt, &s.IP, &s.UserAgent)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (s *Store) CreateSession(sess *Session) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (`+sessionColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.UserID, sess.CreatedAt, sess.LastSeenAt, sess.ExpiresAt, sess.IP, sess.UserAgent,
	)
	return err
}

func (s *Store) GetSession(id string) (*Session, error) {
	return scanSession(s.db.QueryRow(`SELECT `+sessionColumns+` FROM sessions WHERE id = ?`, id))
}

func (s *Store) ListUserSessions(userID int64) ([]*Session, error) {
	rows, err := s.db.Query(`SELECT `+sessionColumns+` FROM sessions WHERE user_id = ? AND expires_at > ? ORDER BY last_seen_at DESC`, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []*Session{}
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) TouchSession(id string, lastSeen, expires time.Time) error {
	_, err := s.db.Exec(`UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`, lastSeen, expires, id)
	return err
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteUserSessions removes all sessions of a user, except the one given (may be empty).
func (s *Store) DeleteUserSessions(userID int64, except string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND id != ?`, userID, except)
	return err
}

func (s *Store) DeleteExpiredSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().UTC())
	return err
}
