package store

import (
	"database/sql"
	"errors"
	"time"
)

const (
	FriendPending  = "pending"
	FriendAccepted = "accepted"
)

// Friend is a friendship as seen from one user.
type Friend struct {
	User      *PublicUser `json:"user"`
	Status    string      `json:"status"`    // pending | accepted
	Direction string      `json:"direction"` // incoming | outgoing (for pending requests)
	CreatedAt time.Time   `json:"created_at"`
}

// PublicUser is what other users may see about an account.
type PublicUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (u *User) Public() *PublicUser {
	return &PublicUser{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName}
}

func (s *Store) ListFriends(userID int64) ([]*Friend, error) {
	rows, err := s.db.Query(`
		SELECT f.user_id, f.status, f.created_at, u.id, u.username, u.display_name
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.user_id = ? THEN f.friend_id ELSE f.user_id END
		WHERE (f.user_id = ? OR f.friend_id = ?) AND u.disabled = 0
		ORDER BY f.status, u.username`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	friends := []*Friend{}
	for rows.Next() {
		var requester int64
		f := &Friend{User: &PublicUser{}}
		if err := rows.Scan(&requester, &f.Status, &f.CreatedAt, &f.User.ID, &f.User.Username, &f.User.DisplayName); err != nil {
			return nil, err
		}
		if requester == userID {
			f.Direction = "outgoing"
		} else {
			f.Direction = "incoming"
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

// FriendIDs returns IDs of accepted friends.
func (s *Store) FriendIDs(userID int64) (map[int64]bool, error) {
	rows, err := s.db.Query(`
		SELECT CASE WHEN user_id = ? THEN friend_id ELSE user_id END
		FROM friendships WHERE (user_id = ? OR friend_id = ?) AND status = ?`,
		userID, userID, userID, FriendAccepted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// friendship returns the row between two users in any direction.
func (s *Store) friendship(a, b int64) (requester int64, status string, err error) {
	err = s.db.QueryRow(`SELECT user_id, status FROM friendships WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`, a, b, b, a).
		Scan(&requester, &status)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

// RequestFriend sends a request from -> to. If `to` already requested
// `from`, the friendship is accepted instead. Returns resulting status.
func (s *Store) RequestFriend(from, to int64) (string, error) {
	requester, status, err := s.friendship(from, to)
	if errors.Is(err, ErrNotFound) {
		_, err = s.db.Exec(`INSERT INTO friendships (user_id, friend_id, status, created_at) VALUES (?, ?, ?, ?)`, from, to, FriendPending, time.Now().UTC())
		return FriendPending, err
	}
	if err != nil {
		return "", err
	}
	if status == FriendPending && requester == to {
		return FriendAccepted, s.AcceptFriend(from, to)
	}
	return status, nil
}

// AcceptFriend accepts a pending request sent by `from` to `user`.
func (s *Store) AcceptFriend(user, from int64) error {
	res, err := s.db.Exec(`UPDATE friendships SET status = ? WHERE user_id = ? AND friend_id = ? AND status = ?`, FriendAccepted, from, user, FriendPending)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveFriend deletes a friendship or request in any direction.
func (s *Store) RemoveFriend(a, b int64) error {
	_, err := s.db.Exec(`DELETE FROM friendships WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`, a, b, b, a)
	return err
}

func (s *Store) SearchUsers(query string, limit int) ([]*PublicUser, error) {
	rows, err := s.db.Query(`SELECT id, username, display_name FROM users WHERE disabled = 0 AND (username LIKE ? OR display_name LIKE ?) ORDER BY username LIMIT ?`,
		"%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*PublicUser{}
	for rows.Next() {
		u := &PublicUser{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
