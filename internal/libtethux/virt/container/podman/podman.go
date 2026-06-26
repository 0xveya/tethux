package podman

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/client"

	"github.com/0xveya/tethux/internal/libtethux/virt"
	"github.com/0xveya/tethux/internal/libtethux/virt/container"
)

// uncomment to check at compile time if methods are missing
// var _ container.ContainerProvider = (*Podman)(nil)

type Option func(*config)

type config struct {
	socketOverride string
}

type Podman struct {
	cli    *client.Client
	socket string
}

func (p *Podman) Socket() string { return p.socket }

func WithSocket(socket string) Option {
	return func(c *config) {
		c.socketOverride = socket
	}
}

type socketCandidate struct {
	label  string
	socket string
}

func New(opts ...Option) (*Podman, error) {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	socket, err := resolveSocket(cfg)
	if err != nil {
		return nil, err
	}

	cli, err := client.New(client.WithHost(socket))
	if err != nil {
		return nil, fmt.Errorf("podman: failed to create client for socket %q: %w", socket, err)
	}

	return &Podman{cli: cli, socket: socket}, nil
}

func resolveSocket(cfg *config) (string, error) {
	if cfg.socketOverride != "" {
		if err := checkSocket(cfg.socketOverride); err != nil {
			return "", fmt.Errorf("podman: override socket %q not accessible: %w", cfg.socketOverride, err)
		}
		return cfg.socketOverride, nil
	}

	for _, env := range []string{"CONTAINER_HOST", "DOCKER_HOST"} {
		if val := os.Getenv(env); val != "" {
			if err := checkSocket(val); err == nil {
				return val, nil
			}
		}
	}

	for _, candidate := range socketCandidates() {
		if err := checkSocket(candidate.socket); err == nil {
			return candidate.socket, nil
		}
	}

	return "", fmt.Errorf("podman: no accessible socket found; tried %v — is podman running? (try: podman system service --time=0)", socketPaths())
}

func socketCandidates() []socketCandidate {
	var candidates []socketCandidate

	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidates = append(candidates, socketCandidate{
			label:  "rootless/XDG_RUNTIME_DIR",
			socket: "unix://" + filepath.Join(xdg, "podman", "podman.sock"),
		})
	}

	if uid := os.Getuid(); uid > 0 {
		candidates = append(candidates, socketCandidate{
			label:  "rootless/run-user",
			socket: fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid),
		})
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, socketCandidate{
			label:  "rootless/podman-machine-qemu",
			socket: "unix://" + filepath.Join(home, ".local", "share", "containers", "podman", "machine", "qemu", "podman.sock"),
		})
		candidates = append(candidates, socketCandidate{
			label:  "rootless/podman-machine-default",
			socket: "unix://" + filepath.Join(home, ".local", "share", "containers", "podman", "machine", "default", "podman.sock"),
		})
	}

	candidates = append(candidates,
		socketCandidate{
			label:  "rootful/run-podman",
			socket: "unix:///run/podman/podman.sock",
		},
		socketCandidate{
			label:  "rootful/var-run-podman",
			socket: "unix:///var/run/podman/podman.sock",
		},
	)

	return candidates
}

func checkSocket(addr string) error {
	path := addr
	if after, ok := strings.CutPrefix(addr, "unix://"); ok {
		path = after
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%q exists but is not a socket", path)
	}
	return nil
}

func socketPaths() []string {
	var paths []string
	for _, c := range socketCandidates() {
		paths = append(paths, c.socket)
	}
	return paths
}

func (p *Podman) Create(cfg virt.NodeConfig) (*virt.Node, error) { return nil, nil }
func (p *Podman) Start(id string) error                          { return nil }
func (p *Podman) Stop(id string) error                           { return nil }
func (p *Podman) Suspend(id string) error                        { return nil }
func (p *Podman) Resume(id string) error                         { return nil }
func (p *Podman) Delete(id string) error                         { return nil }
func (p *Podman) State(id string) (virt.NodeState, error)        { return "", nil }
func (p *Podman) Reload(id string) (*virt.Node, error)           { return nil, nil }
func (p *Podman) List() ([]*virt.Node, error)                    { return nil, nil }

func (p *Podman) CreateContainer(cfg container.ContainerConfig) (*container.ContainerNode, error) {
	return nil, nil
}
func (p *Podman) Pull(image, tag, registry string) error              { return nil }
func (p *Podman) Exec(id string, cmd []string) error                  { return nil }
func (p *Podman) Logs(id string) (io.ReadCloser, error)               { return nil, nil }
func (p *Podman) Inspect(id string) (*container.ContainerNode, error) { return nil, nil }
