package auth

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := newLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if blocked, _ := l.Blocked("k"); blocked {
			t.Fatalf("blocked too early at %d", i)
		}
		l.Fail("k")
	}
	if blocked, _ := l.Blocked("k"); !blocked {
		t.Fatal("expected block after max failures")
	}
	l.Reset("k")
	if blocked, _ := l.Blocked("k"); blocked {
		t.Fatal("expected reset")
	}
}

func TestValidateUsername(t *testing.T) {
	for _, ok := range []string{"bob", "Bob_1", "a.b-c"} {
		if err := ValidateUsername(ok); err != nil {
			t.Errorf("%q should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a", "-bob", "bob smith", "<script>", "averyveryveryverylongusernamethatistoolong"} {
		if err := ValidateUsername(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestPolicyValidate(t *testing.T) {
	p := DefaultPolicy()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.PasswordMinLength = 2
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for short password length")
	}
}

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "correct horse") || CheckPassword(h, "wrong") {
		t.Fatal("password check failed")
	}
}
