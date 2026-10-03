package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleUser
}

type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         Role       `json:"role"`
	Disabled     bool       `json:"disabled"`
	RoomLimit    int        `json:"room_limit"` // -1 = use global default, 0 = unlimited
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

const userColumns = `id, username, display_name, email, password_hash, role, disabled, room_limit, created_at, updated_at, last_login_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*User, error) {
	u := &User{}
	var lastLogin sql.NullTime
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Role, &u.Disabled, &u.RoomLimit, &u.CreatedAt, &u.UpdatedAt, &lastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return u, nil
}

func isUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CountAdmins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ? AND disabled = 0`, RoleAdmin).Scan(&n)
	return n, err
}

func (s *Store) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`SELECT ` + userColumns + ` FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) GetUser(id int64) (*User, error) {
	return scanUser(s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = ?`, id))
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	return scanUser(s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ?`, username))
}

func (s *Store) CreateUser(u *User) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO users (username, display_name, email, password_hash, role, disabled, room_limit, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.DisplayName, u.Email, u.PasswordHash, u.Role, u.Disabled, u.RoomLimit, now, now,
	)
	if isUniqueErr(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}

	u.ID, err = res.LastInsertId()
	u.CreatedAt, u.UpdatedAt = now, now
	return err
}

func (s *Store) UpdateUser(u *User) error {
	u.UpdatedAt = time.Now().UTC()
	res, err := s.db.Exec(
		`UPDATE users SET username = ?, display_name = ?, email = ?, password_hash = ?, role = ?, disabled = ?, room_limit = ?, updated_at = ? WHERE id = ?`,
		u.Username, u.DisplayName, u.Email, u.PasswordHash, u.Role, u.Disabled, u.RoomLimit, u.UpdatedAt, u.ID,
	)
	if isUniqueErr(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TouchUserLogin(id int64) error {
	_, err := s.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now().UTC(), id)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	res, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
