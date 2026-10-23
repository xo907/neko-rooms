package room

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerStrslice "github.com/docker/docker/api/types/strslice"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/m1k1o/neko-rooms/internal/config"
	"github.com/m1k1o/neko-rooms/internal/policies"
	"github.com/m1k1o/neko-rooms/internal/types"
)

const (
	frontendPort        = 8080
	templateStoragePath = "./templates"
	privateStoragePath  = "./rooms"
	privateStorageUid   = 1000
	privateStorageGid   = 1000
)

func New(client *dockerClient.Client, config *config.Room) *RoomManagerCtx {
	logger := log.With().Str("module", "room").Logger()

	return &RoomManagerCtx{
		logger: logger,
		config: config,
		client: client,
		events: newEvents(config, client),
	}
}

type RoomManagerCtx struct {
	logger zerolog.Logger
	config *config.Room
	client *dockerClient.Client
	events *events
	pulls  imagePulls
}

func (manager *RoomManagerCtx) Config() types.RoomsConfig {
	return types.RoomsConfig{
		Connections:    manager.config.EprMax - manager.config.EprMin + 1,
		NekoImages:     manager.config.NekoImages,
		StorageEnabled: manager.config.StorageEnabled,
		UsesMux:        manager.config.Mux,
		RoomDefaults: types.RoomDefaults{
			Screen:        manager.config.RoomDefaults.Screen,
			VideoCodec:    manager.config.RoomDefaults.VideoCodec,
			VideoPipeline: manager.config.RoomDefaults.VideoPipeline,
		},
	}
}

func (manager *RoomManagerCtx) List(ctx context.Context, labels map[string]string) ([]types.RoomEntry, error) {
	containers, err := manager.listContainers(ctx, labels)
	if err != nil {
		return nil, err
	}

	result := make([]types.RoomEntry, 0, len(containers))
	for _, container := range containers {
		entry, err := manager.containerToEntry(container)
		if err != nil {
			return nil, err
		}

		result = append(result, *entry)
	}

	return result, nil
}

func (manager *RoomManagerCtx) ExportAsDockerCompose(ctx context.Context) ([]byte, error) {
	services := map[string]any{}

	dockerCompose := map[string]any{
		"version": "3.8",
		"networks": map[string]any{
			"default": map[string]any{
				"name":     manager.config.InstanceNetwork,
				"external": true,
			},
		},
		"services": services,
	}

	containers, err := manager.listContainers(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		containerJson, err := manager.inspectContainer(ctx, container.ID)
		if err != nil {
			return nil, err
		}

		labels, err := manager.extractLabels(containerJson.Config.Labels)
		if err != nil {
			return nil, err
		}

		containerName := containerJson.Name
		containerName = strings.TrimPrefix(containerName, "/")

		service := map[string]any{}
		services[containerName] = service

		service["image"] = labels.NekoImage
		service["container_name"] = containerName
		service["hostname"] = containerJson.Config.Hostname
		service["restart"] = containerJson.HostConfig.RestartPolicy.Name

		// privileged
		if containerJson.HostConfig.Privileged {
			service["privileged"] = true
		}

		// total shm memory usage
		service["shm_size"] = containerJson.HostConfig.ShmSize

		// capabilites
		capAdd := []string{}
		for _, cap := range containerJson.HostConfig.CapAdd {
			capAdd = append(capAdd, string(cap))
		}
		if len(capAdd) > 0 {
			service["cap_add"] = capAdd
		}

		// resources
		resources := map[string]any{}
		{
			limits := map[string]string{}
			// TODO: CPUShares
			if containerJson.HostConfig.NanoCPUs > 0 {
				limits["cpus"] = fmt.Sprintf("%f", float64(containerJson.HostConfig.NanoCPUs)/1000000000)
			}
			if containerJson.HostConfig.Memory > 0 {
				limits["memory"] = fmt.Sprintf("%dM", containerJson.HostConfig.Memory/1024/1024)
			}
			if len(limits) > 0 {
				resources["limits"] = limits
			}

			deviceRequests := []any{}
			for _, device := range containerJson.HostConfig.DeviceRequests {
				deviceRequests = append(deviceRequests, map[string]any{
					"driver":       device.Driver,
					"count":        device.Count,
					"capabilities": device.Capabilities,
				})
			}
			if len(deviceRequests) > 0 {
				resources["reservations"] = map[string]any{
					"devices": deviceRequests,
				}
			}
		}
		if len(resources) > 0 {
			service["deploy"] = map[string]any{
				"resources": resources,
			}
		}

		// hostname
		if containerJson.Config.Hostname != containerName {
			service["hostname"] = containerJson.Config.Hostname
		}

		// dns
		if len(containerJson.HostConfig.DNS) > 0 {
			service["dns"] = containerJson.HostConfig.DNS
		}

		// ports
		ports := []string{}
		for port, host := range containerJson.HostConfig.PortBindings {
			for _, binding := range host {
				ports = append(ports, fmt.Sprintf("%s:%s", binding.HostPort, port))
			}
		}
		if len(ports) > 0 {
			service["ports"] = ports
		}

		// environment variables
		if len(containerJson.Config.Env) > 0 {
			service["environment"] = containerJson.Config.Env
		}

		// volumes
		volumes := []string{}
		for _, mount := range container.Mounts {
			if !mount.RW {
				volumes = append(volumes, fmt.Sprintf("%s:%s:ro", mount.Source, mount.Destination))
			} else {
				volumes = append(volumes, fmt.Sprintf("%s:%s", mount.Source, mount.Destination))
			}
		}
		if len(volumes) > 0 {
			service["volumes"] = volumes
		}

		// devices
		devices := []string{}
		for _, device := range containerJson.HostConfig.Devices {
			devices = append(devices, fmt.Sprintf("%s:%s:%s", device.PathOnHost, device.PathInContainer, device.CgroupPermissions))
		}
		if len(devices) > 0 {
			service["devices"] = devices
		}

		// labels
		labelsArr := []string{}
		for key, val := range containerJson.Config.Labels {
			labelsArr = append(labelsArr, fmt.Sprintf("%s=%s", key, val))
		}
		if len(labelsArr) > 0 {
			service["labels"] = labelsArr
		}
	}

	return yaml.Marshal(dockerCompose)
}

func (manager *RoomManagerCtx) Create(ctx context.Context, settings types.RoomSettings) (string, error) {
	used, err := manager.getUsedPorts(ctx)
	if err != nil {
		return "", err
	}

	// images that are not there yet are pulled first
	if slices.Contains(manager.config.NekoImages, settings.NekoImage) {
		if err := manager.ensureImage(ctx, settings.NekoImage); err != nil {
			return "", err
		}
	}

	// a remote engine: the web port is published for neko-rooms to reach
	var httpPort uint16
	spec, err := manager.buildSpec(ctx, settings, specOptions{
		usedPorts:   used,
		imageLabels: manager.imageLabels,
		proxyHost: func(roomName string) (string, error) {
			rh := manager.config.Docker.RoomsHost
			if rh == "" {
				return "", nil
			}
			port, err := manager.allocateHTTPPort(ctx)
			if err != nil {
				return "", err
			}
			httpPort = port
			return net.JoinHostPort(rh, strconv.Itoa(int(port))), nil
		},
	})
	if err != nil {
		return "", err
	}
	settings = spec.Settings

	portBindings := nat.PortMap{}
	for port := spec.Epr.Min; port <= spec.Epr.Max; port++ {
		portBindings[nat.Port(fmt.Sprintf("%d/udp", port))] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", port),
			},
		}

		// expose TCP port as well when using mux
		if manager.config.Mux {
			portBindings[nat.Port(fmt.Sprintf("%d/tcp", port))] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", port),
				},
			}
		}
	}

	if httpPort != 0 {
		portBindings[nat.Port(fmt.Sprintf("%d/tcp", frontendPort))] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", httpPort),
			},
		}
	}

	exposedPorts := nat.PortSet{
		nat.Port(fmt.Sprintf("%d/tcp", frontendPort)): struct{}{},
	}

	for port := range portBindings {
		exposedPorts[port] = struct{}{}
	}

	//
	// Set container configs
	//

	config := &dockerContainer.Config{
		// Hostname
		Hostname: spec.Hostname,
		// Domainname is preventing from running container on LXC (Proxmox)
		// https://www.gitmemory.com/issue/docker/for-linux/743/524569376
		// Domainname: containerName,
		// List of exposed ports
		ExposedPorts: exposedPorts,
		// List of environment variable to set in the container
		Env: spec.Env,
		// Name of the image as it was passed by the operator (e.g. could be symbolic)
		Image: settings.NekoImage,
		// List of labels set to this container
		Labels: spec.Labels,
	}

	hostConfig := &dockerContainer.HostConfig{
		// Port mapping between the exposed port (container) and the host
		PortBindings: portBindings,
		// Configuration of the logs for this container
		LogConfig: dockerContainer.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
		},
		// Restart policy to be used for the container
		RestartPolicy: dockerContainer.RestartPolicy{
			Name: "unless-stopped",
		},
		// List of kernel capabilities to add to the container
		CapAdd: dockerStrslice.StrSlice{
			"SYS_ADMIN",
		},
		// Total shm memory usage
		ShmSize: settings.Resources.ShmSize,
		// Mounts specs used by the container
		Mounts: spec.Mounts,
		// Resources contains container's resources (cgroups config, ulimits...)
		Resources: dockerContainer.Resources{
			CPUShares:      settings.Resources.CPUShares,
			NanoCPUs:       settings.Resources.NanoCPUs,
			Memory:         settings.Resources.Memory,
			DeviceRequests: spec.DeviceRequests,
			Devices:        spec.Devices,
		},
		// DNS
		DNS: settings.DNS,
		// Privileged
		Privileged: spec.Privileged,
	}

	networkingConfig := &dockerNetwork.NetworkingConfig{
		EndpointsConfig: map[string]*dockerNetwork.EndpointSettings{
			manager.config.InstanceNetwork: {},
		},
	}

	// Creating the actual container
	container, err := manager.client.ContainerCreate(
		ctx,
		config,
		hostConfig,
		networkingConfig,
		nil,
		spec.ContainerName,
	)

	if err != nil {
		return "", err
	}

	return container.ID[:12], nil
}

// imageLabels are the labels of an image on the docker host
func (manager *RoomManagerCtx) imageLabels(ctx context.Context, image string) (map[string]string, error) {
	inspect, err := manager.client.ImageInspect(ctx, image)
	if err != nil {
		return nil, err
	}
	if inspect.Config == nil {
		return map[string]string{}, nil
	}
	return inspect.Config.Labels, nil
}

func (manager *RoomManagerCtx) GetEntry(ctx context.Context, id string) (*types.RoomEntry, error) {
	// we don't support id shorter than 12 chars
	// because they can be ambiguous
	if len(id) < 12 {
		return nil, types.ErrRoomNotFound
	}

	container, err := manager.containerById(ctx, id)
	if err != nil {
		return nil, err
	}

	return manager.containerToEntry(*container)
}

func (manager *RoomManagerCtx) GetEntryByName(ctx context.Context, name string) (*types.RoomEntry, error) {
	container, err := manager.containerByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return manager.containerToEntry(*container)
}

func (manager *RoomManagerCtx) Remove(ctx context.Context, id string) error {
	_, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return err
	}

	// Stop the actual container
	err = manager.client.ContainerStop(ctx, id, dockerContainer.StopOptions{
		Signal:  "SIGTERM",
		Timeout: &manager.config.StopTimeoutSec,
	})

	if err != nil {
		return err
	}

	// Remove the actual container
	err = manager.client.ContainerRemove(ctx, id, dockerContainer.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	return err
}

func (manager *RoomManagerCtx) GetSettings(ctx context.Context, id string) (*types.RoomSettings, error) {
	container, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	labels, err := manager.extractLabels(container.Config.Labels)
	if err != nil {
		return nil, err
	}

	privateStorageRoot := path.Join(manager.config.StorageExternal, privateStoragePath, labels.Name)
	templateStorageRoot := path.Join(manager.config.StorageExternal, templateStoragePath)

	mounts := []types.RoomMount{}
	for _, mount := range container.Mounts {
		mountType := types.MountPublic
		hostPath := mount.Source

		if strings.HasPrefix(hostPath, privateStorageRoot) {
			mountType = types.MountPrivate
			hostPath = strings.TrimPrefix(hostPath, privateStorageRoot)
		} else if strings.HasPrefix(hostPath, templateStorageRoot) {
			mountType = types.MountTemplate
			hostPath = strings.TrimPrefix(hostPath, templateStorageRoot)
		} else if !mount.RW {
			mountType = types.MountProtected
		}

		mounts = append(mounts, types.RoomMount{
			Type:          mountType,
			HostPath:      hostPath,
			ContainerPath: mount.Destination,
		})
	}

	var browserPolicy *types.BrowserPolicy
	if labels.BrowserPolicy != nil {
		browserPolicy = &types.BrowserPolicy{
			Type: labels.BrowserPolicy.Type,
			Path: labels.BrowserPolicy.Path,
		}

		var policyMount *types.RoomMount
		for _, mount := range mounts {
			if mount.ContainerPath == labels.BrowserPolicy.Path {
				policyMount = &mount
				break
			}
		}

		// TODO: Refactor.
		if policyMount != nil && policyMount.Type == types.MountTemplate {
			templateInternalPath := path.Join(manager.config.StorageInternal, templateStoragePath, policyMount.HostPath)
			if _, err := os.Stat(templateInternalPath); !os.IsNotExist(err) {
				if data, err := os.ReadFile(templateInternalPath); err == nil {
					if content, err := policies.Parse(string(data), labels.BrowserPolicy.Type); err == nil {
						browserPolicy.Content = *content
					}
				}
			}
		}
	}

	var roomResources types.RoomResources
	if container.HostConfig != nil {
		gpus := []string{}
		for _, req := range container.HostConfig.DeviceRequests {
			var isGpu bool
			var caps []string
			for _, cc := range req.Capabilities {
				for _, c := range cc {
					if c == "gpu" {
						isGpu = true
						continue
					}
					caps = append(caps, c)
				}
			}
			if !isGpu {
				continue
			}

			if req.Count > 1 {
				gpus = append(gpus, fmt.Sprintf("count=%d", req.Count))
			} else if req.Count == -1 {
				gpus = append(gpus, "all")
			}
			if req.Driver != "" {
				gpus = append(gpus, fmt.Sprintf("driver=%s", req.Driver))
			}
			if len(req.DeviceIDs) > 0 {
				gpus = append(gpus, fmt.Sprintf("device=%s", strings.Join(req.DeviceIDs, ",")))
			}
			if len(caps) > 0 {
				gpus = append(gpus, fmt.Sprintf("capabilities=%s", strings.Join(caps, ",")))
			}
			var opts []string
			for key, val := range req.Options {
				opts = append(opts, fmt.Sprintf("%s=%s", key, val))
			}
			if len(opts) > 0 {
				gpus = append(gpus, fmt.Sprintf("options=%s", strings.Join(opts, ",")))
			}
		}

		devices := []string{}
		for _, dev := range container.HostConfig.Devices {
			// TODO: dev.CgroupPermissions
			if dev.PathOnHost == dev.PathInContainer {
				devices = append(devices, dev.PathOnHost)
			} else {
				devices = append(devices, fmt.Sprintf("%s:%s", dev.PathOnHost, dev.PathInContainer))
			}
		}

		roomResources = types.RoomResources{
			CPUShares: container.HostConfig.CPUShares,
			NanoCPUs:  container.HostConfig.NanoCPUs,
			ShmSize:   container.HostConfig.ShmSize,
			Memory:    container.HostConfig.Memory,
			Gpus:      gpus,
			Devices:   devices,
		}
	}

	settings := types.RoomSettings{
		ApiVersion:     labels.ApiVersion,
		Name:           labels.Name,
		NekoImage:      labels.NekoImage,
		MaxConnections: labels.Epr.Max - labels.Epr.Min + 1,
		Labels:         labels.UserDefined,
		Mounts:         mounts,
		Resources:      roomResources,
		Hostname:       container.Config.Hostname,
		DNS:            container.HostConfig.DNS,
		BrowserPolicy:  browserPolicy,
	}

	if labels.Mux {
		settings.MaxConnections = 0
	}

	err = settings.FromEnv(labels.ApiVersion, container.Config.Env)
	return &settings, err
}

func (manager *RoomManagerCtx) GetStats(ctx context.Context, id string) (*types.RoomStats, error) {
	container, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	labels, err := manager.extractLabels(container.Config.Labels)
	if err != nil {
		return nil, err
	}

	settings := types.RoomSettings{}
	err = settings.FromEnv(labels.ApiVersion, container.Config.Env)
	if err != nil {
		return nil, err
	}

	var startedAt time.Time
	if container.State.StartedAt != "" {
		startedAt, err = time.Parse(time.RFC3339, container.State.StartedAt)
		if err != nil {
			return nil, err
		}
	}

	return nekoStats(ctx, labels.ApiVersion, settings.AdminPass, manager.execFetch(id), startedAt)
}

// nekoFetch gets a path of a room's neko http api
type nekoFetch func(ctx context.Context, path string) ([]byte, error)

// execFetch asks neko from inside its container
func (manager *RoomManagerCtx) execFetch(id string) nekoFetch {
	return func(ctx context.Context, path string) ([]byte, error) {
		return manager.containerExecRaw(ctx, id, []string{"wget", "-q", "-O-", fmt.Sprintf("http://127.0.0.1:%d%s", frontendPort, path)})
	}
}

// nekoStats asks neko who is in the room
func nekoStats(ctx context.Context, apiVersion int, adminPass string, fetch nekoFetch, startedAt time.Time) (*types.RoomStats, error) {
	var stats types.RoomStats
	switch apiVersion {
	case 2:
		output, err := fetch(ctx, "/stats?pwd="+url.QueryEscape(adminPass))
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(output, &stats); err != nil {
			return nil, err
		}
	case 3:
		output, err := fetch(ctx, "/api/sessions?token="+url.QueryEscape(adminPass))
		if err != nil {
			return nil, err
		}

		var sessions []struct {
			ID      string `json:"id"`
			Profile struct {
				Name    string `json:"name"`
				IsAdmin bool   `json:"is_admin"`
			} `json:"profile"`
			State struct {
				IsConnected       bool       `json:"is_connected"`
				NotConnectedSince *time.Time `json:"not_connected_since"`
			} `json:"state"`
		}

		if err := json.Unmarshal(output, &sessions); err != nil {
			return nil, err
		}

		// create empty array so that it's not null in json
		stats.Members = []*types.RoomMember{}

		for _, session := range sessions {
			if session.State.IsConnected {
				stats.Connections++
				// append members
				stats.Members = append(stats.Members, &types.RoomMember{
					ID:    session.ID,
					Name:  session.Profile.Name,
					Admin: session.Profile.IsAdmin,
					Muted: false, // not supported
				})
			} else if session.State.NotConnectedSince != nil {
				// populate last admin left time
				if session.Profile.IsAdmin && (stats.LastAdminLeftAt == nil || (*session.State.NotConnectedSince).After(*stats.LastAdminLeftAt)) {
					stats.LastAdminLeftAt = session.State.NotConnectedSince
				}
				// populate last user left time
				if !session.Profile.IsAdmin && (stats.LastUserLeftAt == nil || (*session.State.NotConnectedSince).After(*stats.LastUserLeftAt)) {
					stats.LastUserLeftAt = session.State.NotConnectedSince
				}
			}
		}

		stats.ServerStartedAt = startedAt

		// TODO: settings & host
	default:
		return nil, fmt.Errorf("unsupported API version: %d", apiVersion)
	}

	return &stats, nil
}

func (manager *RoomManagerCtx) Start(ctx context.Context, id string) error {
	container, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return err
	}

	// If paused, we need to unpause the container
	if container.State.Paused {
		if err := manager.client.ContainerUnpause(ctx, id); err != nil {
			return err
		}

		return nil
	}

	// Start the actual container
	return manager.client.ContainerStart(ctx, id, dockerContainer.StartOptions{})
}

func (manager *RoomManagerCtx) Stop(ctx context.Context, id string) error {
	_, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return err
	}

	// Stop the actual container
	return manager.client.ContainerStop(ctx, id, dockerContainer.StopOptions{
		Signal:  "SIGTERM",
		Timeout: &manager.config.StopTimeoutSec,
	})
}

func (manager *RoomManagerCtx) Restart(ctx context.Context, id string) error {
	_, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return err
	}

	// Restart the actual container
	return manager.client.ContainerRestart(ctx, id, dockerContainer.StopOptions{
		Signal:  "SIGTERM",
		Timeout: &manager.config.StopTimeoutSec,
	})
}

func (manager *RoomManagerCtx) Pause(ctx context.Context, id string) error {
	_, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return err
	}

	// Pause the actual container
	return manager.client.ContainerPause(ctx, id)
}

// events

func (manager *RoomManagerCtx) EventsLoopStart() {
	manager.events.Start()
}

func (manager *RoomManagerCtx) EventsLoopStop() error {
	return manager.events.Shutdown()
}

func (manager *RoomManagerCtx) Events(ctx context.Context) (<-chan types.RoomEvent, <-chan error) {
	return manager.events.Events(ctx)
}

// GetScreenshot returns a JPEG screenshot of the room screen.
func (manager *RoomManagerCtx) GetScreenshot(ctx context.Context, id string) ([]byte, error) {
	container, err := manager.inspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	labels, err := manager.extractLabels(container.Config.Labels)
	if err != nil {
		return nil, err
	}

	settings := types.RoomSettings{}
	if err := settings.FromEnv(labels.ApiVersion, container.Config.Env); err != nil {
		return nil, err
	}

	return nekoScreenshot(ctx, labels.ApiVersion, settings.AdminPass, manager.execFetch(id))
}

// nekoScreenshot asks neko for a JPEG of the screen
func nekoScreenshot(ctx context.Context, apiVersion int, adminPass string, fetch nekoFetch) ([]byte, error) {
	var path string
	switch apiVersion {
	case 2:
		path = "/screenshot.jpg?pwd=" + url.QueryEscape(adminPass)
	case 3:
		path = "/api/room/screen/shot.jpg?quality=60&token=" + url.QueryEscape(adminPass)
	default:
		return nil, fmt.Errorf("unsupported API version: %d", apiVersion)
	}

	data, err := fetch(ctx, path)
	if err != nil {
		return nil, err
	}

	// must be a jpeg
	if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, fmt.Errorf("invalid screenshot data")
	}

	return data, nil
}

// x11 runs a command against the room's X display.
func (manager *RoomManagerCtx) x11(ctx context.Context, id string, script string, args ...string) ([]byte, error) {
	cmd := append([]string{"sh", "-c", `DISPLAY="${DISPLAY:-:99.0}"; export DISPLAY; ` + script, "sh"}, args...)
	return manager.containerExecRaw(ctx, id, cmd)
}

// GetWindowTitle returns the title of the active browser window, usually
// the title of the page that is open.
func (manager *RoomManagerCtx) GetWindowTitle(ctx context.Context, id string) (string, error) {
	out, err := manager.x11(ctx, id, `xdotool getactivewindow getwindowname`)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// OpenURL opens a page in the room's browser, by typing it in the address bar.
func (manager *RoomManagerCtx) OpenURL(ctx context.Context, id string, url string) error {
	_, err := manager.x11(ctx, id, `xdotool key --clearmodifiers ctrl+l && sleep 0.2 && xdotool type --delay 4 -- "$1" && sleep 0.1 && xdotool key Return`, url)
	return err
}

// NekoPost sends a JSON request to neko inside the room's container.
func (manager *RoomManagerCtx) NekoPost(ctx context.Context, id string, path string, body []byte) ([]byte, error) {
	return manager.containerExecRaw(ctx, id, []string{
		"wget", "-q", "-O-",
		"--header=Content-Type: application/json",
		"--post-data=" + string(body),
		fmt.Sprintf("http://127.0.0.1:%d%s", frontendPort, path),
	})
}
