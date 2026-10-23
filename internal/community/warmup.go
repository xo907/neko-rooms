package community

import (
	"sync"
	"time"
)

// Predictive warm-up: rooms learn when people usually join them and a
// sleeping room is woken a few minutes before, so joining is instant. A room
// woken for nobody goes back to sleep on the usual timer.

const (
	warmLead     = 3 * time.Minute  // wake this long before the usual time
	warmSlot     = 20 * time.Minute // joins this close to the time of day count
	warmWeeks    = 6                // weeks of history for weekly habits
	warmKeep     = 60 * 24 * time.Hour
	warmCooldown = 2 * time.Hour // one warm-up per room in this time
	warmRecord   = 15 * time.Minute
)

type warmup struct {
	mu       sync.Mutex
	recorded map[string]time.Time // room -> last activity written
	woken    map[string]time.Time // room -> last warm-up
	checked  time.Time
	pruned   time.Time
}

// clockDistance is how far apart two times are on the clock, ignoring dates
func clockDistance(a, b time.Time) time.Duration {
	day := 24 * time.Hour
	sa := time.Duration(a.UTC().Hour())*time.Hour + time.Duration(a.UTC().Minute())*time.Minute + time.Duration(a.UTC().Second())*time.Second
	sb := time.Duration(b.UTC().Hour())*time.Hour + time.Duration(b.UTC().Minute())*time.Minute + time.Duration(b.UTC().Second())*time.Second
	d := sa - sb
	if d < 0 {
		d = -d
	}
	if d > day/2 {
		d = day - d
	}
	return d
}

// expectedSoon is true when people usually join a room a few minutes from
// now: around this time on the same weekday in at least 2 of the last 6
// weeks, or on at least 4 of the last 7 days. Today does not count.
func expectedSoon(history []time.Time, now time.Time) bool {
	target := now.Add(warmLead)
	weeks := map[int]bool{}
	days := map[int]bool{}
	for _, at := range history {
		if clockDistance(at, target) > warmSlot {
			continue
		}
		ago := int(target.Sub(at).Hours()/24 + 0.5)
		if ago < 1 {
			continue
		}
		if ago <= 7 {
			days[ago] = true
		}
		if ago%7 == 0 && ago/7 <= warmWeeks {
			weeks[ago/7] = true
		}
	}
	return len(weeks) >= 2 || len(days) >= 4
}

// recordActivity remembers that someone joined a room, at most every few
// minutes per room
func (s *Service) recordActivity(room string) {
	w := &s.warm
	now := time.Now()
	w.mu.Lock()
	if w.recorded == nil {
		w.recorded = map[string]time.Time{}
	}
	if last, ok := w.recorded[room]; ok && now.Sub(last) < warmRecord {
		w.mu.Unlock()
		return
	}
	w.recorded[room] = now
	w.mu.Unlock()

	if err := s.store.AddRoomActivity(room, now); err != nil {
		s.logger.Warn().Err(err).Str("room", room).Msg("unable to record room activity")
	}
}

// warmUp wakes sleeping rooms people are about to join, once a minute
func (s *Service) warmUp(sleeping map[string]func()) {
	w := &s.warm
	now := time.Now()
	w.mu.Lock()
	if now.Sub(w.checked) < time.Minute {
		w.mu.Unlock()
		return
	}
	w.checked = now
	prune := now.Sub(w.pruned) > 24*time.Hour
	if prune {
		w.pruned = now
	}
	if w.woken == nil {
		w.woken = map[string]time.Time{}
	}
	w.mu.Unlock()

	if prune {
		if err := s.store.PruneRoomActivity(now.Add(-warmKeep)); err != nil {
			s.logger.Warn().Err(err).Msg("unable to prune room activity")
		}
	}
	if len(sleeping) == 0 {
		return
	}

	history, err := s.store.RoomActivitySince(now.AddDate(0, 0, -7*warmWeeks-1))
	if err != nil {
		s.logger.Warn().Err(err).Msg("unable to read room activity")
		return
	}

	for room, wake := range sleeping {
		if !expectedSoon(history[room], now) {
			continue
		}
		w.mu.Lock()
		recent := now.Sub(w.woken[room]) < warmCooldown
		if !recent {
			w.woken[room] = now
		}
		w.mu.Unlock()
		if recent {
			continue
		}
		s.logger.Info().Str("room", room).Msg("people usually join soon, waking the room")
		go wake()
	}
}
