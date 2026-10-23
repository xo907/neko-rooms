package store

import "time"

// AddRoomActivity records that someone joined a room.
func (s *Store) AddRoomActivity(room string, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO room_activity (room, at) VALUES (?, ?)`, room, at.UTC())
	return err
}

// RoomActivitySince returns when each room was joined since a time.
func (s *Store) RoomActivitySince(since time.Time) (map[string][]time.Time, error) {
	rows, err := s.db.Query(`SELECT room, at FROM room_activity WHERE at >= ? ORDER BY at`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]time.Time{}
	for rows.Next() {
		var room string
		var at time.Time
		if err := rows.Scan(&room, &at); err != nil {
			return nil, err
		}
		result[room] = append(result[room], at.UTC())
	}
	return result, rows.Err()
}

// PruneRoomActivity forgets activity older than a time.
func (s *Store) PruneRoomActivity(before time.Time) error {
	_, err := s.db.Exec(`DELETE FROM room_activity WHERE at < ?`, before.UTC())
	return err
}
