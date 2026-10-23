package room

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
)

type EprPorts struct {
	Min uint16
	Max uint16
}

// allocateFrom finds sum free ports in min-max, next to the used ones
func allocateFrom(used []EprPorts, min, max, sum uint16) (EprPorts, error) {
	if sum < 1 {
		return EprPorts{}, fmt.Errorf("unable to allocate 0 ports")
	}

	epr := EprPorts{
		Min: min,
		Max: min + sum - 1,
	}

	sort.SliceStable(used, func(i, j int) bool {
		return used[i].Min < used[j].Min
	})

	for _, port := range used {
		if (epr.Min >= port.Min && epr.Min <= port.Max) ||
			(epr.Max >= port.Min && epr.Max <= port.Max) {
			epr.Min = port.Max + 1
			epr.Max = port.Max + sum
		}
	}

	if epr.Min > max || epr.Max > max {
		return epr, fmt.Errorf("unable to allocate ports: not enough ports")
	}

	return epr, nil
}

func (manager *RoomManagerCtx) getUsedPorts(ctx context.Context) ([]EprPorts, error) {
	containers, err := manager.listContainers(ctx, nil)
	if err != nil {
		return nil, err
	}

	result := []EprPorts{}
	for _, container := range containers {
		labels, err := manager.extractLabels(container.Labels)
		if err != nil {
			return nil, err
		}

		result = append(result, labels.Epr)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Min < result[j].Min
	})

	return result, nil
}

// ProxyHostLabel holds the address neko-rooms reaches a room's web port on,
// when it is not the container itself (remote engines, swarm, k8s).
const ProxyHostLabel = "m1k1o.neko_rooms.proxy.host"

// allocateHTTPPort finds a free port for a room's web port on a remote
// docker host.
func (manager *RoomManagerCtx) allocateHTTPPort(ctx context.Context) (uint16, error) {
	containers, err := manager.listContainers(ctx, nil)
	if err != nil {
		return 0, err
	}

	used := map[uint16]bool{}
	for _, container := range containers {
		if addr, ok := container.Labels[ProxyHostLabel]; ok {
			if _, port, err := net.SplitHostPort(addr); err == nil {
				if p, err := strconv.ParseUint(port, 10, 16); err == nil {
					used[uint16(p)] = true
				}
			}
		}
	}

	for port := manager.config.Docker.HTTPMin; port <= manager.config.Docker.HTTPMax && port != 0; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("unable to allocate a web port: all of %d-%d are taken", manager.config.Docker.HTTPMin, manager.config.Docker.HTTPMax)
}
