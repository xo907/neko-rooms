package room

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	dockerImage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/m1k1o/neko-rooms/internal/types"
)

// how long a request waits for an image before it is told to come back
const pullWait = 40 * time.Second

// a pull shared by everyone who needs the same image
type imagePull struct {
	done chan struct{}
	err  error
}

type imagePulls struct {
	mu    sync.Mutex
	pulls map[string]*imagePull
}

// pull downloads an image once at a time, callers that need the same image
// wait for the same download. It keeps going in the background when the
// caller gives up waiting.
func (manager *RoomManagerCtx) pull(ctx context.Context, image string) error {
	p := &manager.pulls
	p.mu.Lock()
	if p.pulls == nil {
		p.pulls = map[string]*imagePull{}
	}
	job, ok := p.pulls[image]
	if !ok {
		job = &imagePull{done: make(chan struct{})}
		p.pulls[image] = job
		go func() {
			job.err = manager.pullNow(image)
			close(job.done)
			p.mu.Lock()
			delete(p.pulls, image)
			p.mu.Unlock()
		}()
	}
	p.mu.Unlock()

	select {
	case <-job.done:
		return job.err
	case <-ctx.Done():
		return types.ErrImagePulling
	}
}

func (manager *RoomManagerCtx) pullNow(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	manager.logger.Info().Str("image", image).Msg("pulling image")
	reader, err := manager.client.ImagePull(ctx, image, dockerImage.PullOptions{})
	if err != nil {
		manager.logger.Err(err).Str("image", image).Msg("unable to pull image")
		return err
	}
	defer reader.Close()

	// the stream carries errors as messages
	if err := jsonmessage.DisplayJSONMessagesStream(reader, io.Discard, 0, false, nil); err != nil {
		manager.logger.Err(err).Str("image", image).Msg("unable to pull image")
		return err
	}
	manager.logger.Info().Str("image", image).Msg("pulled image")
	return nil
}

// EnsureImage pulls an image the docker host does not have yet, it gives up
// waiting after a while with types.ErrImagePulling.
func (manager *RoomManagerCtx) EnsureImage(ctx context.Context, image string) error {
	return manager.ensureImage(ctx, image)
}

func (manager *RoomManagerCtx) ensureImage(ctx context.Context, image string) error {
	_, err := manager.client.ImageInspect(ctx, image)
	if err == nil || !errdefs.IsNotFound(err) {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, pullWait)
	defer cancel()
	return manager.pull(ctx, image)
}

func (manager *RoomManagerCtx) imageID(ctx context.Context, image string) string {
	inspect, err := manager.client.ImageInspect(ctx, image)
	if err != nil {
		return ""
	}
	return inspect.ID
}

// UpdateImage pulls the newest version of an image, it is true when there
// was a newer one.
func (manager *RoomManagerCtx) UpdateImage(ctx context.Context, image string) (bool, error) {
	before := manager.imageID(ctx, image)
	if err := manager.pull(ctx, image); err != nil {
		return false, err
	}
	return manager.imageID(ctx, image) != before, nil
}
