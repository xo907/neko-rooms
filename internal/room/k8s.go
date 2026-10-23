package room

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/m1k1o/neko-rooms/internal/config"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// K8s runs every room as a deployment with one replica (none while stopped)
// and a service for its web port. neko-rooms reaches rooms by service name,
// so it runs in the cluster too. WebRTC ports are host ports on the node the
// room runs on, nat1to1 is that node's public address; a node selector keeps
// rooms on the right nodes.
//
// Room labels live in annotations, kubernetes labels can not hold them.
// Pausing is not supported.

const (
	k8sInstanceLabel = "neko-rooms.m1k1o.net/instance"
	k8sRoomLabel     = "neko-rooms.m1k1o.net/room"
)

var k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

type K8sManagerCtx struct {
	logger zerolog.Logger
	config *config.Room
	rest   *rest.Config
	client kubernetes.Interface
	http   *http.Client
	poller *poller

	// labels and specs are shared with docker, no docker client is used
	base *RoomManagerCtx

	mu sync.Mutex
}

func NewK8s(config *config.Room) (*K8sManagerCtx, error) {
	var cfg *rest.Config
	var err error
	if config.K8s.Kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", config.K8s.Kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	if config.Traefik.Enabled {
		return nil, errors.New("traefik labels are not supported on kubernetes, use an ingress for neko-rooms instead")
	}

	m := &K8sManagerCtx{
		logger: log.With().Str("module", "room").Str("runtime", "k8s").Logger(),
		config: config,
		rest:   cfg,
		client: client,
		http:   &http.Client{Timeout: 5 * time.Second},
		base:   New(nil, config),
	}
	m.poller = newPoller(m.logger, func(ctx context.Context) ([]types.RoomEntry, error) { return m.List(ctx, nil) })
	return m, nil
}

func (m *K8sManagerCtx) ns() string { return m.config.K8s.Namespace }

func (m *K8sManagerCtx) Config() types.RoomsConfig { return m.base.Config() }

//
// listing
//

func (m *K8sManagerCtx) deployments(ctx context.Context) ([]appsv1.Deployment, error) {
	list, err := m.client.AppsV1().Deployments(m.ns()).List(ctx, metav1.ListOptions{
		LabelSelector: k8sInstanceLabel + "=" + m.config.InstanceName,
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func k8sID(d *appsv1.Deployment) string {
	return strings.ReplaceAll(string(d.UID), "-", "")[:12]
}

func (m *K8sManagerCtx) toEntry(d *appsv1.Deployment) (*types.RoomEntry, error) {
	labels, err := m.base.extractLabels(d.Annotations)
	if err != nil {
		return nil, err
	}

	var want int32
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	running := want > 0 && d.Status.Replicas > 0
	ready := running && d.Status.ReadyReplicas > 0

	status := "Stopped"
	switch {
	case ready:
		status = fmt.Sprintf("Running (%d/%d ready)", d.Status.ReadyReplicas, want)
	case running:
		status = "Starting"
	case want > 0:
		status = "Scheduling"
	}

	e := &types.RoomEntry{
		ID:              k8sID(d),
		URL:             labels.URL,
		Name:            labels.Name,
		NekoImage:       labels.NekoImage,
		MaxConnections:  labels.Epr.Max - labels.Epr.Min + 1,
		Running:         running,
		IsReady:         ready,
		Status:          status,
		Created:         d.CreationTimestamp.Time,
		Labels:          labels.UserDefined,
		ContainerLabels: d.Annotations,
	}
	if labels.Mux {
		e.MaxConnections = 0
	}
	return e, nil
}

func (m *K8sManagerCtx) List(ctx context.Context, filter map[string]string) ([]types.RoomEntry, error) {
	list, err := m.deployments(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]types.RoomEntry, 0, len(list))
next:
	for i := range list {
		for key, val := range filter {
			if list[i].Annotations["m1k1o.neko_rooms.x-"+key] != val {
				continue next
			}
		}
		e, err := m.toEntry(&list[i])
		if err != nil {
			continue
		}
		result = append(result, *e)
	}
	return result, nil
}

func (m *K8sManagerCtx) deployment(ctx context.Context, id string) (*appsv1.Deployment, error) {
	if len(id) < 12 {
		return nil, types.ErrRoomNotFound
	}
	list, err := m.deployments(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if strings.HasPrefix(strings.ReplaceAll(string(list[i].UID), "-", ""), id) {
			return &list[i], nil
		}
	}
	return nil, types.ErrRoomNotFound
}

func (m *K8sManagerCtx) GetEntry(ctx context.Context, id string) (*types.RoomEntry, error) {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.toEntry(d)
}

func (m *K8sManagerCtx) GetEntryByName(ctx context.Context, name string) (*types.RoomEntry, error) {
	list, err := m.deployments(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Labels[k8sRoomLabel] == name {
			return m.toEntry(&list[i])
		}
	}
	return nil, types.ErrRoomNotFound
}

func (m *K8sManagerCtx) usedPorts(ctx context.Context) ([]EprPorts, error) {
	list, err := m.deployments(ctx)
	if err != nil {
		return nil, err
	}
	result := []EprPorts{}
	for i := range list {
		labels, err := m.base.extractLabels(list[i].Annotations)
		if err != nil {
			continue
		}
		result = append(result, labels.Epr)
	}
	return result, nil
}

//
// create
//

// k8sAPIVersion guesses the neko api version of an image, images can not be
// inspected from here
func k8sAPIVersion(image string) map[string]string {
	version := "3"
	if strings.HasPrefix(image, "m1k1o/neko:") || strings.HasPrefix(image, "docker.io/m1k1o/neko:") ||
		strings.Contains(image, "neko-apps") || strings.Contains(image, "/neko/") && strings.Contains(image, ":2.") {
		version = "2"
	}
	return map[string]string{"net.m1k1o.neko.api-version": version}
}

func randomName() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func (m *K8sManagerCtx) Create(ctx context.Context, settings types.RoomSettings) (string, error) {
	if settings.Name == "" {
		settings.Name = randomName()
	}
	if !k8sNameRegex.MatchString(m.config.InstanceName+"-"+settings.Name) || len(m.config.InstanceName+"-"+settings.Name) > 63 {
		return "", errors.New("on kubernetes room names may only have lowercase letters, digits and dashes")
	}

	used, err := m.usedPorts(ctx)
	if err != nil {
		return "", err
	}

	spec, err := m.base.buildSpec(ctx, settings, specOptions{
		usedPorts: used,
		imageLabels: func(ctx context.Context, image string) (map[string]string, error) {
			return k8sAPIVersion(image), nil
		},
		proxyHost: func(roomName string) (string, error) {
			return fmt.Sprintf("%s-%s.%s.svc:%d", m.config.InstanceName, roomName, m.ns(), frontendPort), nil
		},
	})
	if err != nil {
		return "", err
	}
	settings = spec.Settings

	stored, err := encodeSettings(settings)
	if err != nil {
		return "", err
	}
	annotations := map[string]string{}
	for k, v := range spec.Labels {
		annotations[k] = v
	}
	annotations[SettingsLabel] = stored

	selector := map[string]string{
		k8sInstanceLabel: m.config.InstanceName,
		k8sRoomLabel:     spec.Name,
	}

	// ports: the web port inside the cluster, webrtc on the node
	ports := []corev1.ContainerPort{{Name: "http", ContainerPort: frontendPort, Protocol: corev1.ProtocolTCP}}
	for port := spec.Epr.Min; port <= spec.Epr.Max; port++ {
		ports = append(ports, corev1.ContainerPort{ContainerPort: int32(port), HostPort: int32(port), Protocol: corev1.ProtocolUDP})
		if m.config.Mux {
			ports = append(ports, corev1.ContainerPort{ContainerPort: int32(port), HostPort: int32(port), Protocol: corev1.ProtocolTCP})
		}
	}

	env := []corev1.EnvVar{}
	for _, kv := range spec.Env {
		k, v, _ := strings.Cut(kv, "=")
		env = append(env, corev1.EnvVar{Name: k, Value: v})
	}

	// mounts are paths on the node
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	for i, mount := range spec.Mounts {
		name := fmt.Sprintf("mount-%d", i)
		hostPathType := corev1.HostPathDirectoryOrCreate
		volumes = append(volumes, corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mount.Source, Type: &hostPathType}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: name, MountPath: mount.Target, ReadOnly: mount.ReadOnly})
	}
	if settings.Resources.ShmSize > 0 {
		size := resource.NewQuantity(settings.Resources.ShmSize, resource.BinarySI)
		volumes = append(volumes, corev1.Volume{
			Name:         "shm",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory, SizeLimit: size}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "shm", MountPath: "/dev/shm"})
	}

	limits := corev1.ResourceList{}
	if settings.Resources.NanoCPUs > 0 {
		limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(settings.Resources.NanoCPUs/1_000_000, resource.DecimalSI)
	}
	if settings.Resources.Memory > 0 {
		limits[corev1.ResourceMemory] = *resource.NewQuantity(settings.Resources.Memory, resource.BinarySI)
	}
	if n := k8sGPUs(settings.Resources.Gpus); n > 0 {
		limits["nvidia.com/gpu"] = *resource.NewQuantity(n, resource.DecimalSI)
	}
	if len(settings.Resources.Devices) > 0 {
		return "", errors.New("devices are not supported on kubernetes, use a device plugin")
	}

	privileged := spec.Privileged
	podSpec := corev1.PodSpec{
		Hostname:     strings.ToLower(strings.ReplaceAll(spec.Hostname, "_", "-")),
		NodeSelector: m.config.K8s.NodeSelector,
		Containers: []corev1.Container{{
			Name:            "neko",
			Image:           settings.NekoImage,
			ImagePullPolicy: corev1.PullPolicy(m.config.K8s.PullPolicy),
			Env:             env,
			Ports:           ports,
			VolumeMounts:    volumeMounts,
			Resources:       corev1.ResourceRequirements{Limits: limits},
			SecurityContext: &corev1.SecurityContext{
				Privileged:   &privileged,
				Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"SYS_ADMIN"}},
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler:     corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(frontendPort)}},
				PeriodSeconds:    3,
				FailureThreshold: 2,
			},
		}},
		Volumes:                       volumes,
		TerminationGracePeriodSeconds: ptrInt64(int64(m.config.StopTimeoutSec)),
	}
	if !k8sNameRegex.MatchString(podSpec.Hostname) || len(podSpec.Hostname) > 63 {
		podSpec.Hostname = ""
	}
	if len(settings.DNS) > 0 {
		podSpec.DNSPolicy = corev1.DNSNone
		podSpec.DNSConfig = &corev1.PodDNSConfig{Nameservers: settings.DNS}
	}

	replicas := int32(0)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.ContainerName,
			Namespace:   m.ns(),
			Labels:      selector,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			// host ports can not be shared, the old pod goes first
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selector},
				Spec:       podSpec,
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.ContainerName,
			Namespace: m.ns(),
			Labels:    selector,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       frontendPort,
				TargetPort: intstr.FromInt32(frontendPort),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}

	if _, err := m.client.CoreV1().Services(m.ns()).Create(ctx, service, metav1.CreateOptions{}); err != nil && !k8serrors.IsAlreadyExists(err) {
		return "", err
	}
	created, err := m.client.AppsV1().Deployments(m.ns()).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		m.client.CoreV1().Services(m.ns()).Delete(ctx, spec.ContainerName, metav1.DeleteOptions{})
		return "", err
	}
	return k8sID(created), nil
}

func ptrInt64(v int64) *int64 { return &v }

// k8sGPUs reads a gpu count from docker style options: all, count=N
func k8sGPUs(opts []string) int64 {
	for _, o := range opts {
		switch {
		case o == "all":
			return 1
		case strings.HasPrefix(o, "count="):
			var n int64
			fmt.Sscanf(strings.TrimPrefix(o, "count="), "%d", &n)
			return n
		}
	}
	if len(opts) > 0 {
		return 1
	}
	return 0
}

func (m *K8sManagerCtx) GetSettings(ctx context.Context, id string) (*types.RoomSettings, error) {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return nil, err
	}
	stored, ok := d.Annotations[SettingsLabel]
	if !ok {
		return nil, errors.New("room settings are missing")
	}
	return decodeSettings(stored)
}

//
// lifecycle
//

func (m *K8sManagerCtx) scale(ctx context.Context, id string, n int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, err := m.deployment(ctx, id)
	if err != nil {
		return err
	}
	d.Spec.Replicas = &n
	_, err = m.client.AppsV1().Deployments(m.ns()).Update(ctx, d, metav1.UpdateOptions{})
	return err
}

func (m *K8sManagerCtx) Start(ctx context.Context, id string) error { return m.scale(ctx, id, 1) }

func (m *K8sManagerCtx) Stop(ctx context.Context, id string) error { return m.scale(ctx, id, 0) }

func (m *K8sManagerCtx) Restart(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, err := m.deployment(ctx, id)
	if err != nil {
		return err
	}
	if d.Spec.Template.Annotations == nil {
		d.Spec.Template.Annotations = map[string]string{}
	}
	d.Spec.Template.Annotations["neko-rooms.m1k1o.net/restarted-at"] = time.Now().UTC().Format(time.RFC3339)
	_, err = m.client.AppsV1().Deployments(m.ns()).Update(ctx, d, metav1.UpdateOptions{})
	return err
}

func (m *K8sManagerCtx) Pause(ctx context.Context, id string) error {
	return errors.New("pausing rooms is not supported on kubernetes, stop them instead")
}

func (m *K8sManagerCtx) Remove(ctx context.Context, id string) error {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return err
	}
	policy := metav1.DeletePropagationForeground
	if err := m.client.AppsV1().Deployments(m.ns()).Delete(ctx, d.Name, metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		return err
	}
	if err := m.client.CoreV1().Services(m.ns()).Delete(ctx, d.Name, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// deleting takes a while, a room recreated under the same name has to
	// wait for the old one to be gone
	for {
		_, errD := m.client.AppsV1().Deployments(m.ns()).Get(ctx, d.Name, metav1.GetOptions{})
		_, errS := m.client.CoreV1().Services(m.ns()).Get(ctx, d.Name, metav1.GetOptions{})
		if k8serrors.IsNotFound(errD) && k8serrors.IsNotFound(errS) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("room is still being removed: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func (m *K8sManagerCtx) ExportAsDockerCompose(ctx context.Context) ([]byte, error) {
	return nil, errors.New("exporting rooms is not supported on kubernetes")
}

//
// neko
//

func (m *K8sManagerCtx) nekoOf(ctx context.Context, id string) (*types.RoomSettings, nekoFetch, *appsv1.Deployment, error) {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	settings, err := decodeSettings(d.Annotations[SettingsLabel])
	if err != nil {
		return nil, nil, nil, err
	}
	addr := d.Annotations[ProxyHostLabel]
	fetch := func(ctx context.Context, path string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+path, nil)
		if err != nil {
			return nil, err
		}
		res, err := m.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("neko answered %s", res.Status)
		}
		return io.ReadAll(io.LimitReader(res.Body, 16<<20))
	}
	return settings, fetch, d, nil
}

func (m *K8sManagerCtx) GetStats(ctx context.Context, id string) (*types.RoomStats, error) {
	settings, fetch, d, err := m.nekoOf(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoStats(ctx, settings.ApiVersion, settings.AdminPass, fetch, d.CreationTimestamp.Time)
}

func (m *K8sManagerCtx) GetScreenshot(ctx context.Context, id string) ([]byte, error) {
	settings, fetch, _, err := m.nekoOf(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoScreenshot(ctx, settings.ApiVersion, settings.AdminPass, fetch)
}

// exec runs a command in the room's pod
func (m *K8sManagerCtx) exec(ctx context.Context, id string, cmd []string) ([]byte, error) {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return nil, err
	}
	pods, err := m.client.CoreV1().Pods(m.ns()).List(ctx, metav1.ListOptions{
		LabelSelector: k8sInstanceLabel + "=" + m.config.InstanceName + "," + k8sRoomLabel + "=" + d.Labels[k8sRoomLabel],
	})
	if err != nil {
		return nil, err
	}
	pod := ""
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			pod = p.Name
			break
		}
	}
	if pod == "" {
		return nil, errors.New("the room is not running")
	}

	req := m.client.CoreV1().RESTClient().Post().
		Resource("pods").Namespace(m.ns()).Name(pod).SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{Container: "neko", Command: cmd, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(m.rest, http.MethodPost, req.URL())
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr}); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (m *K8sManagerCtx) x11(ctx context.Context, id, script string, args ...string) ([]byte, error) {
	cmd := append([]string{"sh", "-c", `DISPLAY="${DISPLAY:-:99.0}"; export DISPLAY; ` + script, "sh"}, args...)
	return m.exec(ctx, id, cmd)
}

func (m *K8sManagerCtx) GetWindowTitle(ctx context.Context, id string) (string, error) {
	out, err := m.x11(ctx, id, `xdotool getactivewindow getwindowname`)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *K8sManagerCtx) OpenURL(ctx context.Context, id string, url string) error {
	_, err := m.x11(ctx, id, `xdotool key --clearmodifiers ctrl+l && sleep 0.2 && xdotool type --delay 4 -- "$1" && sleep 0.1 && xdotool key Return`, url)
	return err
}

// the cluster pulls images, always the newest on start
func (m *K8sManagerCtx) EnsureImage(ctx context.Context, image string) error { return nil }

func (m *K8sManagerCtx) UpdateImage(ctx context.Context, image string) (bool, error) {
	return true, nil
}

//
// events
//

func (m *K8sManagerCtx) EventsLoopStart() { m.poller.Start() }

func (m *K8sManagerCtx) EventsLoopStop() error { return m.poller.Shutdown() }

func (m *K8sManagerCtx) Events(ctx context.Context) (<-chan types.RoomEvent, <-chan error) {
	return m.poller.Events(ctx)
}
