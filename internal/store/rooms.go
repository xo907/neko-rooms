package store

import (
	"database/sql"
	"errors"
	"time"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"  // listed on the homepage, anyone can join
	VisibilityFriends Visibility = "friends" // listed for the owner's friends
	VisibilityPrivate Visibility = "private" // owner, admins and invite link holders only
)

func (v Visibility) Valid() bool {
	return v == VisibilityPublic || v == VisibilityFriends || v == VisibilityPrivate
}

// RoomMeta holds community data about a room, keyed by the room name.
type RoomMeta struct {
	Name        string     `json:"name"`
	OwnerID     *int64     `json:"owner_id"`
	Visibility  Visibility `json:"visibility"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	Featured    bool       `json:"featured"`
	Hidden      bool       `json:"hidden"` // hidden by a moderator, never listed
	InviteCode  string     `json:"invite_code,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

const roomMetaColumns = `name, owner_id, visibility, title, description, category, featured, hidden, invite_code, created_at, updated_at`

func scanRoomMeta(row scanner) (*RoomMeta, error) {
	m := &RoomMeta{}
	var owner sql.NullInt64
	err := row.Scan(&m.Name, &owner, &m.Visibility, &m.Title, &m.Description, &m.Category, &m.Featured, &m.Hidden, &m.InviteCode, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if owner.Valid {
		m.OwnerID = &owner.Int64
	}
	return m, nil
}

func (s *Store) GetRoomMeta(name string) (*RoomMeta, error) {
	return scanRoomMeta(s.db.QueryRow(`SELECT `+roomMetaColumns+` FROM room_meta WHERE name = ?`, name))
}

func (s *Store) ListRoomMeta() (map[string]*RoomMeta, error) {
	rows, err := s.db.Query(`SELECT ` + roomMetaColumns + ` FROM room_meta`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]*RoomMeta{}
	for rows.Next() {
		m, err := scanRoomMeta(rows)
		if err != nil {
			return nil, err
		}
		result[m.Name] = m
	}
	return result, rows.Err()
}

func (s *Store) PutRoomMeta(m *RoomMeta) error {
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	_, err := s.db.Exec(
		`INSERT INTO room_meta (`+roomMetaColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET owner_id = excluded.owner_id, visibility = excluded.visibility,
		   title = excluded.title, description = excluded.description, category = excluded.category,
		   featured = excluded.featured, hidden = excluded.hidden, invite_code = excluded.invite_code,
		   updated_at = excluded.updated_at`,
		m.Name, m.OwnerID, m.Visibility, m.Title, m.Description, m.Category, m.Featured, m.Hidden, m.InviteCode, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (s *Store) DeleteRoomMeta(name string) error {
	_, err := s.db.Exec(`DELETE FROM room_meta WHERE name = ?`, name)
	return err
}

func (s *Store) CountRoomsOwnedBy(userID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM room_meta WHERE owner_id = ?`, userID).Scan(&n)
	return n, err
}
