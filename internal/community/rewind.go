package community

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/m1k1o/neko-rooms/internal/rewind"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// SetRewind turns on rewinding, the recordings live in m.
func (s *Service) SetRewind(m *rewind.Manager) {
	s.rewind = m
	m.OnAudio(s.captions.Audio)
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
	s.rewindMu.Lock()
	s.rewindRooms = rooms
	s.rewindMu.Unlock()
	s.resyncRewind()
}

// resyncRewind applies the last refresh again, e.g. when someone turned on
// subtitles and the room has to be listened to right away
func (s *Service) resyncRewind() {
	if s.rewind == nil {
		return
	}
	s.rewindMu.Lock()
	rooms := append([]rewind.Room(nil), s.rewindRooms...)
	s.rewindMu.Unlock()
	for i := range rooms {
		rooms[i].Audio = s.captions.Wanted(rooms[i].Name)
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

// RewindStream writes the recording of a room from segment n on as one
// WebM, for someone in it.
func (s *Service) RewindStream(ctx context.Context, room, ticket string, n int, w io.Writer, flush func()) error {
	if _, err := s.ParseTicket(room, ticket); err != nil {
		return err
	}
	if s.rewind == nil {
		return os.ErrNotExist
	}
	return s.rewind.Stream(ctx, room, n, w, flush)
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
