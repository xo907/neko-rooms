package community

import (
	"testing"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
)

func ptr(v int64) *int64 { return &v }

func TestVisibilityRules(t *testing.T) {
	owner := &store.User{ID: 1, Role: store.RoleUser}
	friend := &store.User{ID: 2, Role: store.RoleUser}
	stranger := &store.User{ID: 3, Role: store.RoleUser}
	admin := &store.User{ID: 4, Role: store.RoleAdmin}

	policy := auth.DefaultPolicy()
	mk := func(u *store.User) *viewer {
		v := &viewer{user: u, friends: map[int64]bool{}, policy: policy}
		if u != nil && u.ID == friend.ID {
			v.friends[owner.ID] = true
		}
		return v
	}

	pub := &store.RoomMeta{OwnerID: ptr(1), Visibility: store.VisibilityPublic}
	fr := &store.RoomMeta{OwnerID: ptr(1), Visibility: store.VisibilityFriends}
	priv := &store.RoomMeta{OwnerID: ptr(1), Visibility: store.VisibilityPrivate, InviteCode: "secret"}
	hidden := &store.RoomMeta{OwnerID: ptr(1), Visibility: store.VisibilityPublic, Hidden: true}

	tests := []struct {
		name       string
		v          *viewer
		meta       *store.RoomMeta
		invite     string
		list, join bool
	}{
		{"guest public", mk(nil), pub, "", true, true},
		{"guest friends", mk(nil), fr, "", false, false},
		{"stranger friends", mk(stranger), fr, "", false, false},
		{"friend friends", mk(friend), fr, "", true, true},
		{"friend private", mk(friend), priv, "", false, false},
		{"stranger private with invite", mk(stranger), priv, "secret", false, true},
		{"stranger private wrong invite", mk(stranger), priv, "nope", false, false},
		{"owner private", mk(owner), priv, "", true, true},
		{"admin private", mk(admin), priv, "", false, true},
		{"stranger hidden", mk(stranger), hidden, "", false, false},
		{"owner hidden", mk(owner), hidden, "", true, true},
		{"guest unmanaged room", mk(nil), nil, "", false, false},
		{"admin unmanaged room", mk(admin), nil, "", false, true},
	}

	for _, tt := range tests {
		if got := tt.v.canList(tt.meta); got != tt.list {
			t.Errorf("%s: canList = %v, want %v", tt.name, got, tt.list)
		}
		if got := tt.v.canJoin(tt.meta, tt.invite); got != tt.join {
			t.Errorf("%s: canJoin = %v, want %v", tt.name, got, tt.join)
		}
	}
}

func TestGuestsBlockedWhenLoginRequired(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.RoomsRequireLogin = true
	v := &viewer{policy: policy, friends: map[int64]bool{}}
	if v.canJoin(&store.RoomMeta{Visibility: store.VisibilityPublic}, "") {
		t.Fatal("guests must not join when login is required")
	}

	policy.GuestsCanBrowse = false
	v.policy = policy
	if v.canList(&store.RoomMeta{Visibility: store.VisibilityPublic}) {
		t.Fatal("guests must not browse when disabled")
	}
}
