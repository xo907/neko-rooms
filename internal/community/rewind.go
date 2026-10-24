package community

import (
	"os"
	"time"

	"github.com/m1k1o/neko-rooms/internal/rewind"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// SetRewind turns on rewinding, the recordings live in m.
func (s *Service) SetRewind(m *rewind.Manager) {
	s.rewind = m
}

// syncRewind records the rooms people watch, after every stats refresh
func (s *Service) syncRewind(entries []types.RoomEntry, stats map[string]roomStats) {
	if s.rewind == nil {
		return
	}
	rooms := make([]rewind.Room, 0, len(entries))
	for _, e := range entries {
		rooms = append(rooms, rewind.Room{
			Name:    e.Name,
			ID:      e.ID,
			Ready:   e.Running && e.IsReady && !e.Paused,
			Viewers: stats[e.ID].connections,
		})
	}
	s.rewind.Sync(rooms, time.Duration(s.Integrations().RewindMinutes)*time.Minute)
}

// RewindIndex is what can be rewound in a room, for someone in it.
func (s *Service) RewindIndex(room, ticket string) (rewind.Index, error) {
	if _, err := s.ParseTicket(room, ticket); err != nil {
		return rewind.Index{}, err
	}
	if s.rewind == nil {
		return rewind.Index{Segments: []rewind.Segment{}}, nil
	}
	idx, ok := s.rewind.Index(room)
	if !ok {
		return rewind.Index{Segments: []rewind.Segment{}}, nil
	}
	return idx, nil
}

// RewindSegment opens a piece of the recording of a room, for someone in it.
func (s *Service) RewindSegment(room, ticket string, n int) (*os.File, rewind.Segment, error) {
	if _, err := s.ParseTicket(room, ticket); err != nil {
		return nil, rewind.Segment{}, err
	}
	if s.rewind == nil {
		return nil, rewind.Segment{}, os.ErrNotExist
	}
	return s.rewind.Open(room, n)
}
