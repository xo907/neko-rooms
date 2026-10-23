package room

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/docker/cli/opts"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerMount "github.com/docker/docker/api/types/mount"
	dockerNames "github.com/docker/docker/daemon/names"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/policies"
	"github.com/m1k1o/neko-rooms/internal/types"
	"github.com/m1k1o/neko-rooms/internal/utils"
)

// roomSpec is a room as every runtime needs it: names, labels, environment,
// ports and mounts. Docker turns it into a container, swarm into a service
// and kubernetes into a pod.
type roomSpec struct {
	Settings      types.RoomSettings // with the api version filled in
	Name          string
	ContainerName string
	Hostname      string
	Epr           EprPorts
	Labels        map[string]string
	Env           []string

	Mounts         []dockerMount.Mount
	DeviceRequests []dockerContainer.DeviceRequest
	Devices        []dockerContainer.DeviceMapping
	Privileged     bool
}

type specOptions struct {
	// webrtc ports taken by other rooms
	usedPorts []EprPorts
	// image labels, to find the neko api version of an image
	imageLabels func(ctx context.Context, image string) (map[string]string, error)
	// the address neko-rooms reaches the room's web port on, when it is
	// not the container on the shared network
	proxyHost func(roomName string) (string, error)
}

// apiVersion reads the neko api version of an image from its labels.
func apiVersion(image string, labels map[string]string) (int, error) {
	// based on image label (preferred)
	if val, ok := labels["net.m1k1o.neko.api-version"]; ok {
		return strconv.Atoi(val)
	}

	// based on opencontainers image url label
	if val, ok := labels["org.opencontainers.image.url"]; ok {
		// TODO: this should be removed in future, but since we have a lot of legacy images, we need to support it
		switch val {
		case "https://github.com/m1k1o/neko",
			"https://github.com/m1k1o/neko-apps":
			return 2, nil
		case "https://github.com/demodesk/neko":
			return 3, nil
		}
	}

	// still unable to detect api version
	// TODO: this should be removed in future, but since we have a lot of v2 images, we need to support it
	log.Warn().Str("image", image).Msg("unable to detect api version, fallback to v2")
	return 2, nil
}

func (manager *RoomManagerCtx) buildSpec(ctx context.Context, settings types.RoomSettings, opt specOptions) (*roomSpec, error) {
	if settings.Name != "" && !dockerNames.RestrictedNamePattern.MatchString(settings.Name) {
		return nil, fmt.Errorf("invalid container name, must match %s", dockerNames.RestrictedNameChars)
	}

	if !slices.Contains(manager.config.NekoImages, settings.NekoImage) {
		return nil, fmt.Errorf("invalid neko image")
	}

	// if api version is not set, try to detect it
	if settings.ApiVersion == 0 {
		labels, err := opt.imageLabels(ctx, settings.NekoImage)
		if err != nil {
			return nil, err
		}
		settings.ApiVersion, err = apiVersion(settings.NekoImage, labels)
		if err != nil {
			return nil, err
		}
	}

	// TODO: Check if path name exists.
	roomName := settings.Name
	if roomName == "" {
		var err error
		roomName, err = utils.NewUID(8)
		if err != nil {
			return nil, err
		}
	}

	containerName := manager.config.InstanceName + "-" + roomName

	//
	// Allocate ports
	//

	portsNeeded := settings.MaxConnections
	if manager.config.Mux {
		portsNeeded = 1
	}

	epr, err := allocateFrom(opt.usedPorts, manager.config.EprMin, manager.config.EprMax, portsNeeded)
	if err != nil {
		return nil, err
	}

	//
	// Set internal labels
	//

	var browserPolicyLabels *BrowserPolicyLabels
	if settings.BrowserPolicy != nil {
		browserPolicyLabels = &BrowserPolicyLabels{
			Type: settings.BrowserPolicy.Type,
			Path: settings.BrowserPolicy.Path,
		}
	}

	labels := manager.serializeLabels(RoomLabels{
		Name: roomName,
		Mux:  manager.config.Mux,
		Epr:  epr,

		NekoImage:  settings.NekoImage,
		ApiVersion: settings.ApiVersion,

		BrowserPolicy: browserPolicyLabels,
		UserDefined:   settings.Labels,
	})

	//
	// Set traefik labels
	//

	pathPrefix := path.Join("/", manager.config.PathPrefix, roomName)

	if t := manager.config.Traefik; t.Enabled {
		// create traefik rule
		traefikRule := "PathPrefix(`" + pathPrefix + "`)"
		if t.Domain != "" && t.Domain != "*" {
			// match *.domain.tld as subdomain
			if after, ok := strings.CutPrefix(t.Domain, "*."); ok {
				traefikRule = fmt.Sprintf(
					"Host(`%s.%s`)",
					roomName,
					after,
				)
			} else {
				traefikRule += " && Host(`" + t.Domain + "`)"
			}
		} else {
			traefikRule += " && HostRegexp(`{host:.+}`)"
		}

		labels["traefik.enable"] = "true"
		labels["traefik.http.services."+containerName+"-frontend.loadbalancer.server.port"] = fmt.Sprintf("%d", frontendPort)
		labels["traefik.http.routers."+containerName+".entrypoints"] = t.Entrypoint
		labels["traefik.http.routers."+containerName+".rule"] = traefikRule
		labels["traefik.http.middlewares."+containerName+"-rdr.redirectregex.regex"] = pathPrefix + "$$"
		labels["traefik.http.middlewares."+containerName+"-rdr.redirectregex.replacement"] = pathPrefix + "/"
		labels["traefik.http.middlewares."+containerName+"-prf.stripprefix.prefixes"] = pathPrefix + "/"
		labels["traefik.http.routers."+containerName+".middlewares"] = containerName + "-rdr," + containerName + "-prf"
		labels["traefik.http.routers."+containerName+".service"] = containerName + "-frontend"

		// optional HTTPS
		if t.Certresolver != "" {
			labels["traefik.http.routers."+containerName+".tls"] = "true"
			labels["traefik.http.routers."+containerName+".tls.certresolver"] = t.Certresolver
		}
	} else {
		labels["m1k1o.neko_rooms.proxy.enabled"] = "true"
		labels["m1k1o.neko_rooms.proxy.path"] = pathPrefix
		labels["m1k1o.neko_rooms.proxy.port"] = fmt.Sprintf("%d", frontendPort)

		if opt.proxyHost != nil {
			host, err := opt.proxyHost(roomName)
			if err != nil {
				return nil, err
			}
			if host != "" {
				labels[ProxyHostLabel] = host
			}
		}
	}

	// add custom labels
	for _, label := range manager.config.Labels {
		// replace dynamic values in labels
		label = strings.ReplaceAll(label, "{containerName}", containerName)
		label = strings.ReplaceAll(label, "{roomName}", roomName)

		if t := manager.config.Traefik; t.Enabled {
			label = strings.ReplaceAll(label, "{traefikEntrypoint}", t.Entrypoint)
			label = strings.ReplaceAll(label, "{traefikCertresolver}", t.Certresolver)
		}

		v := strings.SplitN(label, "=", 2)
		if len(v) != 2 {
			manager.logger.Warn().Str("label", label).Msg("invalid custom label")
			continue
		}

		key, val := v[0], v[1]
		labels[key] = val
	}

	//
	// Set environment variables
	//

	env, err := settings.ToEnv(
		manager.config,
		types.PortSettings{
			FrontendPort: frontendPort,
			EprMin:       epr.Min,
			EprMax:       epr.Max,
		})
	if err != nil {
		return nil, err
	}

	//
	// Set browser policies
	//

	if settings.BrowserPolicy != nil {
		if !manager.config.StorageEnabled {
			return nil, fmt.Errorf("policies cannot be specified, because storage is disabled or unavailable")
		}

		policyJson, err := policies.Generate(settings.BrowserPolicy.Content, settings.BrowserPolicy.Type)
		if err != nil {
			return nil, err
		}

		// create policy path (+ also get host path)
		policyPath := fmt.Sprintf("/%s-%s-policy.json", roomName, settings.BrowserPolicy.Type)
		templateInternalPath := path.Join(manager.config.StorageInternal, templateStoragePath)
		policyInternalPath := path.Join(templateInternalPath, policyPath)

		// create dir if does not exist
		if _, err := os.Stat(templateInternalPath); os.IsNotExist(err) {
			if err := os.MkdirAll(templateInternalPath, os.ModePerm); err != nil {
				return nil, err
			}
		}

		// write policy to file
		if err := os.WriteFile(policyInternalPath, []byte(policyJson), 0644); err != nil {
			return nil, err
		}

		// mount policy file
		settings.Mounts = append(settings.Mounts, types.RoomMount{
			Type:          types.MountTemplate,
			HostPath:      policyPath,
			ContainerPath: settings.BrowserPolicy.Path,
		})
	}

	//
	// Set container mounts
	//

	paths := map[string]bool{}
	mounts := []dockerMount.Mount{}
	for _, mount := range settings.Mounts {
		// ignore duplicates
		if _, ok := paths[mount.ContainerPath]; ok {
			continue
		}

		readOnly := false

		hostPath := filepath.Clean(mount.HostPath)
		containerPath := filepath.Clean(mount.ContainerPath)

		if !filepath.IsAbs(hostPath) || !filepath.IsAbs(containerPath) {
			return nil, fmt.Errorf("mount paths must be absolute")
		}

		switch mount.Type {
		case types.MountPrivate:
			if !manager.config.StorageEnabled {
				return nil, fmt.Errorf("private mounts cannot be specified, because storage is disabled or unavailable")
			}

			// ensure that target exists with correct permissions
			internalPath := path.Join(manager.config.StorageInternal, privateStoragePath, roomName, hostPath)
			if _, err := os.Stat(internalPath); os.IsNotExist(err) {
				if err := os.MkdirAll(internalPath, os.ModePerm); err != nil {
					return nil, err
				}

				if err := utils.ChownR(internalPath, privateStorageUid, privateStorageGid); err != nil {
					return nil, err
				}
			}

			// prefix host path
			hostPath = path.Join(manager.config.StorageExternal, privateStoragePath, roomName, hostPath)
		case types.MountTemplate:
			if !manager.config.StorageEnabled {
				return nil, fmt.Errorf("template mounts cannot be specified, because storage is disabled or unavailable")
			}

			// readonly template data
			readOnly = true

			// prefix host path
			hostPath = path.Join(manager.config.StorageExternal, templateStoragePath, hostPath)
		case types.MountProtected, types.MountPublic:
			// readonly if mount type is protected
			readOnly = mount.Type == types.MountProtected

			// public whitelisted mounts
			var isAllowed = false
			for _, path := range manager.config.MountsWhitelist {
				if strings.HasPrefix(hostPath, path) {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				return nil, fmt.Errorf("mount path is not whitelisted in config")
			}
		default:
			return nil, fmt.Errorf("unknown mount type %q", mount.Type)
		}

		mounts = append(mounts,
			dockerMount.Mount{
				Type:        dockerMount.TypeBind,
				Source:      hostPath,
				Target:      containerPath,
				ReadOnly:    readOnly,
				Consistency: dockerMount.ConsistencyDefault,

				BindOptions: &dockerMount.BindOptions{
					Propagation:  dockerMount.PropagationRPrivate,
					NonRecursive: false,
				},
			},
		)

		paths[mount.ContainerPath] = true
	}

	//
	// Set container device requests
	//

	var deviceRequests []dockerContainer.DeviceRequest

	if len(settings.Resources.Gpus) > 0 {
		gpuOpts := opts.GpuOpts{}

		// convert to csv
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		if err := w.Write(settings.Resources.Gpus); err != nil {
			return nil, err
		}
		w.Flush()

		// set GPU opts
		if err := gpuOpts.Set(buf.String()); err != nil {
			return nil, err
		}

		deviceRequests = append(deviceRequests, gpuOpts.Value()...)
	}

	//
	// Set container devices
	//

	var devices []dockerContainer.DeviceMapping
	for _, device := range settings.Resources.Devices {
		devices = append(devices, dockerContainer.DeviceMapping{
			PathOnHost:        device,
			PathInContainer:   device,
			CgroupPermissions: "rwm",
		})
	}

	hostname := containerName
	if settings.Hostname != "" {
		hostname = settings.Hostname
	}

	return &roomSpec{
		Settings:       settings,
		Name:           roomName,
		ContainerName:  containerName,
		Hostname:       hostname,
		Epr:            epr,
		Labels:         labels,
		Env:            env,
		Mounts:         mounts,
		DeviceRequests: deviceRequests,
		Devices:        devices,
		Privileged:     slices.Contains(manager.config.NekoPrivilegedImages, settings.NekoImage),
	}, nil
}
