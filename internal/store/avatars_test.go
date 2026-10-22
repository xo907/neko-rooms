package store

import (
	"errors"
	"testing"
)

func TestAvatars(t *testing.T) {
	s := newTestStore(t)
	alice := mkUser(t, s, "alice", RoleUser)
	bob := mkUser(t, s, "bob", RoleUser)
	bob.DisplayName = "Bobby"
	if err := s.UpdateUser(bob); err != nil {
		t.Fatal(err)
	}

	if _, err := s.GetAvatar(alice.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected no avatar, got %v", err)
	}

	v1, err := s.PutAvatar(alice.ID, "image/png", []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	// a second upload within the same second still gets a new version
	v2, err := s.PutAvatar(alice.ID, "image/webp", []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	if v2 <= v1 {
		t.Fatalf("versions must increase: %d, %d", v1, v2)
	}
	if _, err := s.PutAvatar(bob.ID, "image/png", []byte("bob")); err != nil {
		t.Fatal(err)
	}

	a, err := s.GetAvatar(alice.ID)
	if err != nil || a.Mime != "image/webp" || string(a.Data) != "two" {
		t.Fatalf("unexpected avatar: %+v %v", a, err)
	}

	u, err := s.GetUser(alice.ID)
	if err != nil || u.Avatar != v2 {
		t.Fatalf("user avatar version: got %d, want %d (%v)", u.Avatar, v2, err)
	}
	if p := u.Public(); p.Avatar != v2 || p.AvatarPath() == "" {
		t.Fatalf("public user without avatar: %+v", p)
	}

	found, err := s.SearchUsers("ali", 10)
	if err != nil || len(found) != 1 || found[0].Avatar != v2 {
		t.Fatalf("search without avatar: %+v %v", found, err)
	}

	// usernames and display names, case-insensitive, only names asked for
	m, err := s.AvatarsByName([]string{"ALICE", "bobby", "carol"})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 || m["ALICE"].ID != alice.ID || m["bobby"].ID != bob.ID {
		t.Fatalf("unexpected lookup: %+v", m)
	}

	if err := s.DeleteAvatar(alice.ID); err != nil {
		t.Fatal(err)
	}
	if u, _ := s.GetUser(alice.ID); u.Avatar != 0 {
		t.Fatalf("avatar not removed: %d", u.Avatar)
	}

	// avatars go away with their user
	if err := s.DeleteUser(bob.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAvatar(bob.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("avatar of deleted user still exists: %v", err)
	}
}
