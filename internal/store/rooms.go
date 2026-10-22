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
	KeepAwake   bool       `json:"keep_awake"`   // never put to sleep when empty
	Sleeping    bool       `json:"sleeping"`     // stopped because it was empty, wakes when someone joins
	ScheduledAt *time.Time `json:"scheduled_at"` // start of a watch party
	StartURL    string     `json:"start_url"`    // page opened when the room starts
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

const roomMetaColumns = `name, owner_id, visibility, title, description, category, featured, hidden, invite_code, keep_awake, sleeping, scheduled_at, start_url, created_at, updated_at`

func scanRoomMeta(row scanner) (*RoomMeta, error) {
	m := &RoomMeta{}
	var owner sql.NullInt64
	var scheduled sql.NullTime
	err := row.Scan(&m.Name, &owner, &m.Visibility, &m.Title, &m.Description, &m.Category, &m.Featured, &m.Hidden, &m.InviteCode, &m.KeepAwake, &m.Sleeping, &scheduled, &m.StartURL, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if owner.Valid {
		m.OwnerID = &owner.Int64
	}
	if scheduled.Valid {
		t := scheduled.Time.UTC()
		m.ScheduledAt = &t
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
	var scheduled any
	if m.ScheduledAt != nil {
		scheduled = m.ScheduledAt.UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO room_meta (`+roomMetaColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET owner_id = excluded.owner_id, visibility = excluded.visibility,
		   title = excluded.title, description = excluded.description, category = excluded.category,
		   featured = excluded.featured, hidden = excluded.hidden, invite_code = excluded.invite_code,
		   keep_awake = excluded.keep_awake, sleeping = excluded.sleeping, scheduled_at = excluded.scheduled_at,
		   start_url = excluded.start_url, updated_at = excluded.updated_at`,
		m.Name, m.OwnerID, m.Visibility, m.Title, m.Description, m.Category, m.Featured, m.Hidden, m.InviteCode,
		m.KeepAwake, m.Sleeping, scheduled, m.StartURL, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// SetRoomSleeping marks a room as put to sleep, or as awake. Rooms without
// metadata get a row with the defaults.
func (s *Store) SetRoomSleeping(name string, sleeping bool) error {
	_, err := s.db.Exec(
		`INSERT INTO room_meta (name, sleeping) VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET sleeping = excluded.sleeping`,
		name, sleeping,
	)
	return err
}

//
// watch party RSVPs
//

func (s *Store) SetRSVP(room string, userID int64, going bool) error {
	var err error
	if going {
		_, err = s.db.Exec(`INSERT OR IGNORE INTO room_rsvps (room, user_id) VALUES (?, ?)`, room, userID)
	} else {
		_, err = s.db.Exec(`DELETE FROM room_rsvps WHERE room = ? AND user_id = ?`, room, userID)
	}
	return err
}

func (s *Store) ClearRSVPs(room string) error {
	_, err := s.db.Exec(`DELETE FROM room_rsvps WHERE room = ?`, room)
	return err
}

// RSVPs returns the user ids going to each room's watch party.
func (s *Store) RSVPs() (map[string][]int64, error) {
	rows, err := s.db.Query(`SELECT room, user_id FROM room_rsvps ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]int64{}
	for rows.Next() {
		var room string
		var id int64
		if err := rows.Scan(&room, &id); err != nil {
			return nil, err
		}
		result[room] = append(result[room], id)
	}
	return result, rows.Err()
}

func (s *Store) DeleteRoomMeta(name string) error {
	_, err := s.db.Exec(`DELETE FROM room_meta WHERE name = ?`, name)
	if err == nil {
		err = s.ClearRSVPs(name)
	}
	return err
}

func (s *Store) CountRoomsOwnedBy(userID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM room_meta WHERE owner_id = ?`, userID).Scan(&n)
	return n, err
}
