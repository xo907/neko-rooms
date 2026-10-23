package room

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerMount "github.com/docker/docker/api/types/mount"
	dockerSwarm "github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/docker/docker/client"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/config"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// Swarm runs rooms as docker swarm services, one replica each, attached to
// the instance overlay network. neko-rooms has to be on that network too: it
// reaches rooms by their service name, for the proxy, stats and screenshots.
//
// Starting and stopping scales the service between one replica and none,
// pausing is not supported. Room settings are kept on the service, so a room
// can be recreated from them.

// SettingsLabel holds the room settings, for runtimes that can not read them
// back from the container (swarm, k8s).
const SettingsLabel = "m1k1o.neko_rooms.settings"

type SwarmManagerCtx struct {
	*RoomManagerCtx // labels, specs and images are shared with docker

	poller *poller
	http   *http.Client

	readyMu sync.Mutex
	ready   map[string]bool // service id -> neko answers
}

func NewSwarm(client *dockerClient.Client, config *config.Room) *SwarmManagerCtx {
	m := &SwarmManagerCtx{
		RoomManagerCtx: New(client, config),
		http:           &http.Client{Timeout: 5 * time.Second},
		ready:          map[string]bool{},
	}
	m.logger = log.With().Str("module", "room").Str("runtime", "swarm").Logger()
	m.poller = newPoller(m.logger, m.listProbed)
	return m
}

func encodeSettings(s types.RoomSettings) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func decodeSettings(v string) (*types.RoomSettings, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	var s types.RoomSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

//
// services
//

func (manager *SwarmManagerCtx) services(ctx context.Context, labels map[string]string) ([]dockerSwarm.Service, error) {
	args := dockerFilters.NewArgs(
		dockerFilters.Arg("label", fmt.Sprintf("m1k1o.neko_rooms.instance=%s", manager.config.InstanceName)),
	)
	for key, val := range labels {
		args.Add("label", fmt.Sprintf("m1k1o.neko_rooms.x-%s=%s", key, val))
	}
	return manager.client.ServiceList(ctx, dockerSwarm.ServiceListOptions{Filters: args, Status: true})
}

func (manager *SwarmManagerCtx) service(ctx context.Context, id string) (*dockerSwarm.Service, error) {
	if len(id) < 12 {
		return nil, types.ErrRoomNotFound
	}
	list, err := manager.services(ctx, nil)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.HasPrefix(list[i].ID, id) {
			return &list[i], nil
		}
	}
	return nil, types.ErrRoomNotFound
}

func (manager *SwarmManagerCtx) toEntry(s dockerSwarm.Service) (*types.RoomEntry, error) {
	labels, err := manager.extractLabels(s.Spec.Labels)
	if err != nil {
		return nil, err
	}

	id := s.ID[:12]
	var running, desired uint64
	if s.ServiceStatus != nil {
		running, desired = s.ServiceStatus.RunningTasks, s.ServiceStatus.DesiredTasks
	}

	status := "Stopped"
	switch {
	case running > 0:
		status = fmt.Sprintf("Running (%d/%d)", running, desired)
	case desired > 0:
		status = "Starting"
	}

	entry := &types.RoomEntry{
		ID:              id,
		URL:             labels.URL,
		Name:            labels.Name,
		NekoImage:       labels.NekoImage,
		MaxConnections:  labels.Epr.Max - labels.Epr.Min + 1,
		Running:         running > 0,
		IsReady:         running > 0 && manager.isReady(id),
		Status:          status,
		Created:         s.CreatedAt,
		Labels:          labels.UserDefined,
		ContainerLabels: s.Spec.Labels,
	}
	if labels.Mux {
		entry.MaxConnections = 0
	}
	return entry, nil
}

func (manager *SwarmManagerCtx) isReady(id string) bool {
	manager.readyMu.Lock()
	defer manager.readyMu.Unlock()
	return manager.ready[id]
}

// listProbed lists the rooms and asks the running ones if neko answers, the
// poller reports rooms becoming ready from it
func (manager *SwarmManagerCtx) listProbed(ctx context.Context) ([]types.RoomEntry, error) {
	list, err := manager.services(ctx, nil)
	if err != nil {
		return nil, err
	}

	ready := map[string]bool{}
	for _, s := range list {
		if s.ServiceStatus == nil || s.ServiceStatus.RunningTasks == 0 {
			continue
		}
		addr := s.Spec.Labels[ProxyHostLabel]
		if addr == "" {
			continue
		}
		res, err := manager.http.Get("http://" + addr + "/health")
		if err == nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			ready[s.ID[:12]] = res.StatusCode < 500
		}
	}
	manager.readyMu.Lock()
	manager.ready = ready
	manager.readyMu.Unlock()

	result := make([]types.RoomEntry, 0, len(list))
	for _, s := range list {
		e, err := manager.toEntry(s)
		if err != nil {
			continue
		}
		result = append(result, *e)
	}
	return result, nil
}

func (manager *SwarmManagerCtx) List(ctx context.Context, labels map[string]string) ([]types.RoomEntry, error) {
	list, err := manager.services(ctx, labels)
	if err != nil {
		return nil, err
	}
	result := make([]types.RoomEntry, 0, len(list))
	for _, s := range list {
		e, err := manager.toEntry(s)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	return result, nil
}

func (manager *SwarmManagerCtx) GetEntry(ctx context.Context, id string) (*types.RoomEntry, error) {
	s, err := manager.service(ctx, id)
	if err != nil {
		return nil, err
	}
	return manager.toEntry(*s)
}

func (manager *SwarmManagerCtx) GetEntryByName(ctx context.Context, name string) (*types.RoomEntry, error) {
	list, err := manager.services(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if s.Spec.Labels["m1k1o.neko_rooms.name"] == name {
			return manager.toEntry(s)
		}
	}
	return nil, types.ErrRoomNotFound
}

func (manager *SwarmManagerCtx) usedPorts(ctx context.Context) ([]EprPorts, error) {
	list, err := manager.services(ctx, nil)
	if err != nil {
		return nil, err
	}
	result := []EprPorts{}
	for _, s := range list {
		labels, err := manager.extractLabels(s.Spec.Labels)
		if err != nil {
			return nil, err
		}
		result = append(result, labels.Epr)
	}
	return result, nil
}

//
// create
//

func (manager *SwarmManagerCtx) Create(ctx context.Context, settings types.RoomSettings) (string, error) {
	if settings.Resources.Gpus != nil && len(settings.Resources.Gpus) > 0 {
		return "", errors.New("GPUs are not supported on swarm")
	}
	if len(settings.Resources.Devices) > 0 {
		return "", errors.New("devices are not supported on swarm")
	}

	used, err := manager.usedPorts(ctx)
	if err != nil {
		return "", err
	}

	spec, err := manager.buildSpec(ctx, settings, specOptions{
		usedPorts: used,
		// nodes pull the image themselves, the api version comes from a
		// copy here when it is not known
		imageLabels: func(ctx context.Context, image string) (map[string]string, error) {
			if err := manager.ensureImage(ctx, image); err != nil {
				return nil, err
			}
			return manager.imageLabels(ctx, image)
		},
		proxyHost: func(roomName string) (string, error) {
			return fmt.Sprintf("%s-%s:%d", manager.config.InstanceName, roomName, frontendPort), nil
		},
	})
	if err != nil {
		return "", err
	}
	if spec.Privileged {
		return "", errors.New("privileged rooms are not supported on swarm")
	}
	settings = spec.Settings

	stored, err := encodeSettings(settings)
	if err != nil {
		return "", err
	}
	spec.Labels[SettingsLabel] = stored

	// webrtc ports are published on the node the room runs on
	ports := []dockerSwarm.PortConfig{}
	for port := spec.Epr.Min; port <= spec.Epr.Max; port++ {
		ports = append(ports, dockerSwarm.PortConfig{
			Protocol:      dockerSwarm.PortConfigProtocolUDP,
			TargetPort:    uint32(port),
			PublishedPort: uint32(port),
			PublishMode:   dockerSwarm.PortConfigPublishModeHost,
		})
		if manager.config.Mux {
			ports = append(ports, dockerSwarm.PortConfig{
				Protocol:      dockerSwarm.PortConfigProtocolTCP,
				TargetPort:    uint32(port),
				PublishedPort: uint32(port),
				PublishMode:   dockerSwarm.PortConfigPublishModeHost,
			})
		}
	}

	mounts := append([]dockerMount.Mount{}, spec.Mounts...)
	if settings.Resources.ShmSize > 0 {
		// swarm has no shm size, a memory backed mount does the same
		mounts = append(mounts, dockerMount.Mount{
			Type:         dockerMount.TypeTmpfs,
			Target:       "/dev/shm",
			TmpfsOptions: &dockerMount.TmpfsOptions{SizeBytes: settings.Resources.ShmSize},
		})
	}

	var dnsConfig *dockerSwarm.DNSConfig
	if len(settings.DNS) > 0 {
		dnsConfig = &dockerSwarm.DNSConfig{Nameservers: settings.DNS}
	}

	var resources *dockerSwarm.ResourceRequirements
	if settings.Resources.NanoCPUs > 0 || settings.Resources.Memory > 0 {
		resources = &dockerSwarm.ResourceRequirements{
			Limits: &dockerSwarm.Limit{
				NanoCPUs:    settings.Resources.NanoCPUs,
				MemoryBytes: settings.Resources.Memory,
			},
		}
	}

	stopGrace := time.Duration(manager.config.StopTimeoutSec) * time.Second
	replicas := uint64(0)

	serviceSpec := dockerSwarm.ServiceSpec{
		Annotations: dockerSwarm.Annotations{
			Name:   spec.ContainerName,
			Labels: spec.Labels,
		},
		TaskTemplate: dockerSwarm.TaskSpec{
			ContainerSpec: &dockerSwarm.ContainerSpec{
				Image:           settings.NekoImage,
				Labels:          spec.Labels,
				Env:             spec.Env,
				Hostname:        spec.Hostname,
				Mounts:          mounts,
				CapabilityAdd:   []string{"SYS_ADMIN"},
				DNSConfig:       dnsConfig,
				StopGracePeriod: &stopGrace,
			},
			Resources: resources,
			RestartPolicy: &dockerSwarm.RestartPolicy{
				Condition: dockerSwarm.RestartPolicyConditionAny,
			},
			Placement: &dockerSwarm.Placement{
				Constraints: manager.config.Swarm.Constraints,
			},
			Networks: []dockerSwarm.NetworkAttachmentConfig{
				{Target: manager.config.InstanceNetwork},
			},
		},
		Mode: dockerSwarm.ServiceMode{
			Replicated: &dockerSwarm.ReplicatedService{Replicas: &replicas},
		},
		EndpointSpec: &dockerSwarm.EndpointSpec{
			Mode:  dockerSwarm.ResolutionModeVIP,
			Ports: ports,
		},
	}

	res, err := manager.client.ServiceCreate(ctx, serviceSpec, dockerSwarm.ServiceCreateOptions{QueryRegistry: true})
	if err != nil {
		return "", err
	}
	return res.ID[:12], nil
}

func (manager *SwarmManagerCtx) GetSettings(ctx context.Context, id string) (*types.RoomSettings, error) {
	s, err := manager.service(ctx, id)
	if err != nil {
		return nil, err
	}
	stored, ok := s.Spec.Labels[SettingsLabel]
	if !ok {
		return nil, errors.New("room settings are missing")
	}
	return decodeSettings(stored)
}

//
// lifecycle
//

func (manager *SwarmManagerCtx) update(ctx context.Context, id string, change func(spec *dockerSwarm.ServiceSpec)) error {
	// swarm bumps the version of new services on its own, an update right
	// after can race it: read the newest version and try again
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		var s *dockerSwarm.Service
		s, err = manager.service(ctx, id)
		if err != nil {
			return err
		}
		spec := s.Spec
		change(&spec)
		_, err = manager.client.ServiceUpdate(ctx, s.ID, s.Version, spec, dockerSwarm.ServiceUpdateOptions{})
		if err == nil || !strings.Contains(err.Error(), "out of sequence") {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
		}
	}
	return err
}

func (manager *SwarmManagerCtx) scale(ctx context.Context, id string, n uint64) error {
	return manager.update(ctx, id, func(spec *dockerSwarm.ServiceSpec) {
		spec.Mode.Replicated = &dockerSwarm.ReplicatedService{Replicas: &n}
	})
}

func (manager *SwarmManagerCtx) Start(ctx context.Context, id string) error {
	return manager.scale(ctx, id, 1)
}

func (manager *SwarmManagerCtx) Stop(ctx context.Context, id string) error {
	return manager.scale(ctx, id, 0)
}

func (manager *SwarmManagerCtx) Restart(ctx context.Context, id string) error {
	return manager.update(ctx, id, func(spec *dockerSwarm.ServiceSpec) {
		spec.TaskTemplate.ForceUpdate++
	})
}

func (manager *SwarmManagerCtx) Pause(ctx context.Context, id string) error {
	return errors.New("pausing rooms is not supported on swarm, stop them instead")
}

func (manager *SwarmManagerCtx) Remove(ctx context.Context, id string) error {
	s, err := manager.service(ctx, id)
	if err != nil {
		return err
	}
	return manager.client.ServiceRemove(ctx, s.ID)
}

func (manager *SwarmManagerCtx) ExportAsDockerCompose(ctx context.Context) ([]byte, error) {
	return nil, errors.New("exporting rooms is not supported on swarm")
}

//
// neko
//

// httpFetch asks neko over the network, by service name
func (manager *SwarmManagerCtx) httpFetch(addr string) nekoFetch {
	return func(ctx context.Context, path string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+path, nil)
		if err != nil {
			return nil, err
		}
		res, err := manager.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("neko answered %s", res.Status)
		}
		return io.ReadAll(io.LimitReader(res.Body, 16<<20))
	}
}

func (manager *SwarmManagerCtx) nekoOf(ctx context.Context, id string) (*types.RoomSettings, nekoFetch, *dockerSwarm.Service, error) {
	s, err := manager.service(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	settings, err := decodeSettings(s.Spec.Labels[SettingsLabel])
	if err != nil {
		return nil, nil, nil, err
	}
	return settings, manager.httpFetch(s.Spec.Labels[ProxyHostLabel]), s, nil
}

func (manager *SwarmManagerCtx) GetStats(ctx context.Context, id string) (*types.RoomStats, error) {
	settings, fetch, s, err := manager.nekoOf(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoStats(ctx, settings.ApiVersion, settings.AdminPass, fetch, s.UpdatedAt)
}

func (manager *SwarmManagerCtx) GetScreenshot(ctx context.Context, id string) ([]byte, error) {
	settings, fetch, _, err := manager.nekoOf(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoScreenshot(ctx, settings.ApiVersion, settings.AdminPass, fetch)
}

// localContainer finds the room's container when it runs on this node, the
// only place commands can run in it
func (manager *SwarmManagerCtx) localContainer(ctx context.Context, id string) (string, error) {
	s, err := manager.service(ctx, id)
	if err != nil {
		return "", err
	}
	list, err := manager.client.ContainerList(ctx, dockerContainer.ListOptions{
		Filters: dockerFilters.NewArgs(dockerFilters.Arg("label", "com.docker.swarm.service.id="+s.ID)),
	})
	if err != nil {
		return "", err
	}
	for _, c := range list {
		if c.State == "running" {
			return c.ID, nil
		}
	}
	return "", errors.New("the room runs on another node")
}

func (manager *SwarmManagerCtx) GetWindowTitle(ctx context.Context, id string) (string, error) {
	cid, err := manager.localContainer(ctx, id)
	if err != nil {
		return "", err
	}
	return manager.RoomManagerCtx.GetWindowTitle(ctx, cid)
}

func (manager *SwarmManagerCtx) OpenURL(ctx context.Context, id string, url string) error {
	cid, err := manager.localContainer(ctx, id)
	if err != nil {
		return err
	}
	return manager.RoomManagerCtx.OpenURL(ctx, cid, url)
}

// nodes pull images when a room starts
func (manager *SwarmManagerCtx) EnsureImage(ctx context.Context, image string) error {
	return nil
}

// UpdateImage can not tell from here, the nodes resolve the newest image when
// the room is recreated
func (manager *SwarmManagerCtx) UpdateImage(ctx context.Context, image string) (bool, error) {
	return true, nil
}

//
// events
//

func (manager *SwarmManagerCtx) EventsLoopStart() {
	manager.poller.Start()
}

func (manager *SwarmManagerCtx) EventsLoopStop() error {
	return manager.poller.Shutdown()
}

func (manager *SwarmManagerCtx) Events(ctx context.Context) (<-chan types.RoomEvent, <-chan error) {
	return manager.poller.Events(ctx)
}
