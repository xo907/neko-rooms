package auth

import (
	"fmt"
)

const policyKey = "policy"

// Policy holds instance-wide access rules, editable by admins.
type Policy struct {
	// self-service sign up
	RegistrationEnabled  bool `json:"registration_enabled"`
	RegistrationApproval bool `json:"registration_approval"` // new accounts stay disabled until an admin enables them

	// what regular users may do
	UsersCanCreateRooms bool     `json:"users_can_create_rooms"`
	UsersCanPullImages  bool     `json:"users_can_pull_images"`
	UsersCanUseMounts   bool     `json:"users_can_use_mounts"` // host path mounts are dangerous, admin-only by default
	UsersCanSetDevices  bool     `json:"users_can_set_devices"`
	UserNekoImages      []string `json:"user_neko_images"` // empty = all configured images
	DefaultRoomLimit    int      `json:"default_room_limit"`
	UserMaxConnections  int      `json:"user_max_connections"` // 0 = unlimited

	// room access
	RoomsRequireLogin bool `json:"rooms_require_login"`

	// community / homepage
	HomepageEnabled    bool `json:"homepage_enabled"`      // public lobby with room directory
	GuestsCanBrowse    bool `json:"guests_can_browse"`     // signed-out visitors can see public rooms
	UsersCanMakePublic bool `json:"users_can_make_public"` // regular users can list rooms publicly
	FriendsEnabled     bool `json:"friends_enabled"`
	ThumbnailsEnabled  bool `json:"thumbnails_enabled"`
	ShowMemberNames    bool `json:"show_member_names"` // show who is in a room on the homepage

	// security
	PasswordMinLength int `json:"password_min_length"`
	SessionTTLHours   int `json:"session_ttl_hours"`
	AuditRetention    int `json:"audit_retention"`
}

func DefaultPolicy() Policy {
	return Policy{
		RegistrationEnabled:  false,
		RegistrationApproval: true,
		UsersCanCreateRooms:  true,
		UsersCanPullImages:   false,
		UsersCanUseMounts:    false,
		UsersCanSetDevices:   false,
		UserNekoImages:       []string{},
		DefaultRoomLimit:     3,
		UserMaxConnections:   0,
		RoomsRequireLogin:    false,
		HomepageEnabled:      true,
		GuestsCanBrowse:      true,
		UsersCanMakePublic:   true,
		FriendsEnabled:       true,
		ThumbnailsEnabled:    true,
		ShowMemberNames:      true,
		PasswordMinLength:    8,
		SessionTTLHours:      24 * 7,
		AuditRetention:       5000,
	}
}

func (p *Policy) Validate() error {
	if p.DefaultRoomLimit < 0 {
		return fmt.Errorf("default room limit must not be negative")
	}
	if p.UserMaxConnections < 0 {
		return fmt.Errorf("max connections must not be negative")
	}
	if p.PasswordMinLength < 6 || p.PasswordMinLength > 128 {
		return fmt.Errorf("password minimum length must be between 6 and 128")
	}
	if p.SessionTTLHours < 1 || p.SessionTTLHours > 24*365 {
		return fmt.Errorf("session lifetime must be between 1 hour and 1 year")
	}
	if p.AuditRetention < 100 {
		return fmt.Errorf("audit retention must be at least 100 entries")
	}
	if p.UserNekoImages == nil {
		p.UserNekoImages = []string{}
	}
	return nil
}

// RoomLimit resolves the effective room limit of a user, 0 = unlimited.
func (p *Policy) RoomLimit(userLimit int) int {
	if userLimit >= 0 {
		return userLimit
	}
	return p.DefaultRoomLimit
}
