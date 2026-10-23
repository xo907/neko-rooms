# Where rooms run

By default neko-rooms starts every room as a container on the docker engine it runs next to. It can also drive a docker engine on another machine, run rooms as Docker Swarm services, or run them as pods in a Kubernetes cluster.

| | Docker (default) | Remote docker | Swarm | Kubernetes |
|---|---|---|---|---|
| `NEKO_ROOMS_RUNTIME` | `docker` | `docker` | `swarm` | `k8s` |
| A room is | a container | a container on the other host | a service with one replica | a deployment + service |
| Start / stop | container | container | scale 1 / 0 | scale 1 / 0 |
| Pause | yes | yes | no | no |
| Storage and mounts | yes | only if neko-rooms shares the host's disk | bind mounts on the node | host paths on the node |
| Export as compose | yes | yes | no | no |
| Now playing, opening links | yes | yes | rooms on the same node | yes |

Video (WebRTC) always goes straight from the room to the viewer: the room's UDP/TCP ports (`NEKO_ROOMS_EPR`) are published on the machine it runs on, and `NEKO_ROOMS_NAT1TO1` must be that machine's public address. With several machines, keep rooms on the one that address belongs to (swarm constraints, a kubernetes node selector).

## Remote docker (TCP or SSH)

Point neko-rooms at another engine with `NEKO_ROOMS_DOCKER_HOST` (or the usual `DOCKER_HOST`):

```yaml
environment:
  # over SSH, with the keys of the container's root user
  - "NEKO_ROOMS_DOCKER_HOST=ssh://deploy@rooms.example.com"
  # or over TCP with TLS (DOCKER_TLS_VERIFY and DOCKER_CERT_PATH work as usual)
  # - "NEKO_ROOMS_DOCKER_HOST=tcp://rooms.example.com:2376"
  # - "DOCKER_TLS_VERIFY=1"
  # - "DOCKER_CERT_PATH=/certs"

  # how neko-rooms reaches that machine; rooms publish their web port there
  - "NEKO_ROOMS_DOCKER_ROOMS_HOST=rooms.example.com"
  # ports for those web ports, default 58000-58999
  # - "NEKO_ROOMS_DOCKER_HTTP_PORTS=58000-58999"
volumes:
  - "./id_ed25519:/root/.ssh/id_ed25519:ro"
```

- SSH needs `docker` on the remote machine for the user you log in as (`ssh deploy@host docker version` must work). The host key is accepted on first connect.
- `NEKO_ROOMS_INSTANCE_NETWORK` is a network on the remote engine: `docker network create neko-rooms` there.
- Rooms' web ports only need to be reachable by neko-rooms, not the internet. Firewall them to its address.
- Storage (`NEKO_ROOMS_STORAGE_*`) writes files next to neko-rooms, it only works when that is the same disk the remote engine sees.
- If the remote engine is away for a while, neko-rooms keeps reconnecting and catches up once it is back.

## Docker Swarm

Rooms become swarm services, neko-rooms runs as a service on a manager node and reaches rooms over an overlay network:

```bash
docker network create -d overlay --attachable neko-rooms

docker service create --name neko-rooms \
  --network neko-rooms -p 8080:8080 \
  --constraint node.role==manager \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  --mount type=volume,src=neko-rooms-data,dst=/data \
  -e NEKO_ROOMS_RUNTIME=swarm \
  -e NEKO_ROOMS_INSTANCE_NETWORK=neko-rooms \
  -e NEKO_ROOMS_TRAEFIK_ENABLED=false \
  -e NEKO_ROOMS_PATH_PREFIX=/room/ \
  -e NEKO_ROOMS_MUX=true \
  -e NEKO_ROOMS_EPR=59000-59049 \
  -e NEKO_ROOMS_NAT1TO1=<public IP of the room node> \
  -e NEKO_ROOMS_SWARM_CONSTRAINTS=node.labels.neko==true \
  ghcr.io/xo907/neko-rooms:latest

docker node update --label-add neko=true <room node>
```

- WebRTC ports are published in host mode on the node a room runs on.
- Swarm has no shared memory size, rooms get a memory backed `/dev/shm` instead. GPUs, devices and privileged rooms are not available.
- Nodes pull images themselves. "Update" recreates the room so the node pulls the newest image.

## Kubernetes

[`deploy/k8s/neko-rooms.yaml`](../deploy/k8s/neko-rooms.yaml) has everything: a namespace, a service account allowed to manage deployments, services and pod exec in it, storage for the database, neko-rooms and its service.

```bash
# set NEKO_ROOMS_NAT1TO1 in the file to the public address of the room node first
kubectl apply -f deploy/k8s/neko-rooms.yaml
kubectl -n neko-rooms port-forward svc/neko-rooms 8080:8080
```

- Each room is a deployment (one replica, none while stopped) plus a service for its web port. Room names become kubernetes names, so lowercase letters, digits and dashes only.
- WebRTC ports are host ports on the node the room runs on. Pin rooms with `NEKO_ROOMS_K8S_NODE_SELECTOR=neko=true` and `kubectl label node <node> neko=true`.
- Rooms pull their image on every start (`NEKO_ROOMS_K8S_PULL_POLICY=Always`), so "Update" gets the newest one. Use `IfNotPresent` for clusters that can not reach the registry.
- GPUs are requested as `nvidia.com/gpu` (with the NVIDIA device plugin). Devices are not available.
- Expose the site with an ingress (example at the end of the file), keep long timeouts for websockets.
- Outside the cluster, `NEKO_ROOMS_K8S_KUBECONFIG=/path/to/kubeconfig` works for managing rooms, but neko-rooms has to reach room services by their cluster names for the proxy, so running inside is the way.
