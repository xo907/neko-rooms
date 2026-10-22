package store

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

// avatarVersion selects the avatar version of the user in the outer query,
// 0 when there is none. It doubles as a cache buster in avatar urls.
const avatarVersion = `COALESCE((SELECT CAST(strftime('%s', a.updated_at) AS INTEGER) FROM avatars a WHERE a.user_id = users.id), 0)`

// AvatarPath is the url of the avatar, relative to the api root.
func (u *PublicUser) AvatarPath() string {
	return "public/avatars/" + strconv.FormatInt(u.ID, 10) + "?v=" + strconv.FormatInt(u.Avatar, 10)
}

type Avatar struct {
	UserID    int64
	Mime      string
	Data      []byte
	UpdatedAt time.Time
}

func (s *Store) GetAvatar(userID int64) (*Avatar, error) {
	a := &Avatar{UserID: userID}
	err := s.db.QueryRow(`SELECT mime, data, updated_at FROM avatars WHERE user_id = ?`, userID).Scan(&a.Mime, &a.Data, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

// PutAvatar stores the avatar of a user and returns its new version.
func (s *Store) PutAvatar(userID int64, mime string, data []byte) (int64, error) {
	now := time.Now().UTC().Truncate(time.Second)
	// keep versions increasing, also for uploads within the same second
	var prev sql.NullTime
	s.db.QueryRow(`SELECT updated_at FROM avatars WHERE user_id = ?`, userID).Scan(&prev)
	if prev.Valid && !now.After(prev.Time) {
		now = prev.Time.Add(time.Second)
	}

	_, err := s.db.Exec(
		`INSERT INTO avatars (user_id, mime, data, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET mime = excluded.mime, data = excluded.data, updated_at = excluded.updated_at`,
		userID, mime, data, now,
	)
	return now.Unix(), err
}

func (s *Store) DeleteAvatar(userID int64) error {
	_, err := s.db.Exec(`DELETE FROM avatars WHERE user_id = ?`, userID)
	return err
}

// AvatarsByName maps the given names to users with an avatar. Names match
// usernames first, then display names, case-insensitively.
func (s *Store) AvatarsByName(names []string) (map[string]*PublicUser, error) {
	res := map[string]*PublicUser{}
	if len(names) == 0 {
		return res, nil
	}

	rows, err := s.db.Query(`SELECT id, username, display_name, ` + avatarVersion + ` FROM users WHERE disabled = 0 AND EXISTS (SELECT 1 FROM avatars a WHERE a.user_id = users.id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byUsername := map[string]*PublicUser{}
	byDisplay := map[string]*PublicUser{}
	for rows.Next() {
		u := &PublicUser{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Avatar); err != nil {
			return nil, err
		}
		byUsername[strings.ToLower(u.Username)] = u
		if u.DisplayName != "" {
			byDisplay[strings.ToLower(u.DisplayName)] = u
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if u, ok := byUsername[key]; ok {
			res[n] = u
		} else if u, ok := byDisplay[key]; ok {
			res[n] = u
		}
	}
	return res, nil
}
