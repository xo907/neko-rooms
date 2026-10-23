package community

import (
	"testing"
	"time"
)

func TestClockDistance(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2030, 1, 1, h, m, 0, 0, time.UTC) }
	if d := clockDistance(at(20, 0), at(20, 15)); d != 15*time.Minute {
		t.Errorf("20:00 to 20:15 = %v", d)
	}
	if d := clockDistance(at(23, 55), at(0, 5).AddDate(0, 0, 3)); d != 10*time.Minute {
		t.Errorf("around midnight = %v", d)
	}
}

func TestExpectedSoon(t *testing.T) {
	// a friday, 19:57: three minutes before the usual 20:00
	now := time.Date(2030, 3, 8, 19, 57, 0, 0, time.UTC)
	usual := func(daysAgo int) time.Time {
		return time.Date(2030, 3, 8, 20, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
	}

	if expectedSoon(nil, now) {
		t.Error("a room without history is expected")
	}
	if expectedSoon([]time.Time{usual(7)}, now) {
		t.Error("one friday is a habit")
	}
	if !expectedSoon([]time.Time{usual(7), usual(21)}, now) {
		t.Error("two of the last fridays are not a habit")
	}
	if !expectedSoon([]time.Time{usual(1), usual(2), usual(3), usual(5)}, now) {
		t.Error("four evenings in a week are not a habit")
	}
	if expectedSoon([]time.Time{usual(1), usual(2), usual(3)}, now) {
		t.Error("three evenings are a habit")
	}
	// joins at another time of day do not count
	noon := []time.Time{usual(7).Add(-8 * time.Hour), usual(14).Add(-8 * time.Hour)}
	if expectedSoon(noon, now) {
		t.Error("lunchtime joins wake the room in the evening")
	}
	// earlier today does not count, the room was just used
	if expectedSoon([]time.Time{now.Add(-time.Hour), now.Add(-2 * time.Hour)}, now) {
		t.Error("today counts")
	}
	// a bit off the usual time still counts
	if !expectedSoon([]time.Time{usual(7).Add(12 * time.Minute), usual(14).Add(-10 * time.Minute)}, now) {
		t.Error("joins a few minutes off do not count")
	}
}
