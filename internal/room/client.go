package room

import (
	"net/http"
	"os"
	"strings"

	"github.com/docker/cli/cli/connhelper"
	dockerClient "github.com/docker/docker/client"
)

// NewDockerClient connects to a docker engine: the local socket, a remote one
// over TCP (DOCKER_TLS_VERIFY and DOCKER_CERT_PATH set up TLS) or over SSH,
// e.g. ssh://user@host, using the ssh client and its keys.
func NewDockerClient(host string) (*dockerClient.Client, error) {
	if host == "" {
		host = os.Getenv("DOCKER_HOST")
	}

	opts := []dockerClient.Opt{
		dockerClient.FromEnv,
		dockerClient.WithAPIVersionNegotiation(),
	}

	switch {
	case strings.HasPrefix(host, "ssh://"):
		helper, err := connhelper.GetConnectionHelperWithSSHOpts(host, []string{
			// a fresh container has never seen the host key
			"-o", "StrictHostKeyChecking=accept-new",
		})
		if err != nil {
			return nil, err
		}
		opts = append(opts,
			dockerClient.WithHTTPClient(&http.Client{Transport: &http.Transport{DialContext: helper.Dialer}}),
			dockerClient.WithHost(helper.Host),
			dockerClient.WithDialContext(helper.Dialer),
		)
	case host != "":
		opts = append(opts, dockerClient.WithHost(host))
	}

	return dockerClient.NewClientWithOpts(opts...)
}
