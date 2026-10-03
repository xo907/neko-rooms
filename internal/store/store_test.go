package store

import (
	"errors"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mkUser(t *testing.T, s *Store, name string, role Role) *User {
	t.Helper()
	u := &User{Username: name, DisplayName: name, PasswordHash: "x", Role: role, RoomLimit: -1}
	if err := s.CreateUser(u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUsersUniqueCaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	mkUser(t, s, "Alice", RoleUser)

	err := s.CreateUser(&User{Username: "alice", PasswordHash: "x", Role: RoleUser})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	u, err := s.GetUserByUsername("ALICE")
	if err != nil || u.Username != "Alice" {
		t.Fatalf("lookup failed: %v %v", u, err)
	}
}

func TestFriendshipFlow(t *testing.T) {
	s := newTestStore(t)
	a := mkUser(t, s, "a", RoleUser)
	b := mkUser(t, s, "b", RoleUser)

	status, err := s.RequestFriend(a.ID, b.ID)
	if err != nil || status != FriendPending {
		t.Fatalf("request: %v %v", status, err)
	}

	// not friends until accepted
	ids, _ := s.FriendIDs(a.ID)
	if ids[b.ID] {
		t.Fatal("pending request must not count as friendship")
	}

	// requester can not accept their own request
	if err := s.AcceptFriend(a.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	// a counter-request accepts
	status, err = s.RequestFriend(b.ID, a.ID)
	if err != nil || status != FriendAccepted {
		t.Fatalf("counter request: %v %v", status, err)
	}

	ids, _ = s.FriendIDs(b.ID)
	if !ids[a.ID] {
		t.Fatal("expected friendship")
	}

	if err := s.RemoveFriend(b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	ids, _ = s.FriendIDs(a.ID)
	if len(ids) != 0 {
		t.Fatal("expected no friends after removal")
	}
}

func TestRoomMetaOwnerSetNullOnDelete(t *testing.T) {
	s := newTestStore(t)
	u := mkUser(t, s, "owner", RoleUser)

	if err := s.PutRoomMeta(&RoomMeta{Name: "r1", OwnerID: &u.ID, Visibility: VisibilityPublic}); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountRoomsOwnedBy(u.ID); n != 1 {
		t.Fatalf("expected 1 room, got %d", n)
	}

	if err := s.DeleteUser(u.ID); err != nil {
		t.Fatal(err)
	}

	m, err := s.GetRoomMeta("r1")
	if err != nil {
		t.Fatal(err)
	}
	if m.OwnerID != nil {
		t.Fatal("owner must be cleared when the user is deleted")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	var v struct{ A int }
	if err := s.GetSetting("x", &v); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := s.SetSetting("x", struct{ A int }{42}); err != nil {
		t.Fatal(err)
	}
	if err := s.GetSetting("x", &v); err != nil || v.A != 42 {
		t.Fatalf("got %v %v", v, err)
	}
}
