package podman

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	moby "github.com/moby/moby/api/types/container"

	"github.com/moby/moby/client"

	"github.com/0xveya/tethux/internal/libtethux/virt"
	"github.com/0xveya/tethux/internal/libtethux/virt/container"
	"github.com/0xveya/tethux/internal/libtethux/virt/container/errs"
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
		return nil, fmt.Errorf("podman: %w: %q: %w", errs.ErrFailedToCreateClent, socket, err)
	}

	return &Podman{cli: cli, socket: socket}, nil
}

func resolveSocket(cfg *config) (string, error) {
	if cfg.socketOverride != "" {
		if err := checkSocket(cfg.socketOverride); err != nil {
			return "", fmt.Errorf("podman: %w: %q err: %w", errs.ErrOverrideSocketNotAccessible, cfg.socketOverride, err)
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

	return "", fmt.Errorf("podman: %w; tried %v — is podman running? (try: podman system service --time=0)", errs.ErrNoSockerFound, socketPaths())
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
		candidates = append(candidates,
			socketCandidate{
				label:  "rootless/podman-machine-qemu",
				socket: "unix://" + filepath.Join(home, ".local", "share", "containers", "podman", "machine", "qemu", "podman.sock"),
			},
			socketCandidate{
				label:  "rootless/podman-machine-default",
				socket: "unix://" + filepath.Join(home, ".local", "share", "containers", "podman", "machine", "default", "podman.sock"),
			},
		)
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
		return fmt.Errorf("%q exists but is %w", path, errs.ErrNotASocket)
	}
	return nil
}

func socketPaths() []string {
	candidates := socketCandidates()
	paths := make([]string, 0, len(candidates))
	for _, c := range candidates {
		paths = append(paths, c.socket)
	}
	return paths
}

// all of these funcs should not disragard things some may be useful i just didnt read into the docs enough yet
// blabalba aobve prolyl missinfo bseides maybe nicer pull

func (p *Podman) StartContainer(ctx context.Context, id string, opts *client.ContainerStartOptions) error {
	o := client.ContainerStartOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerStart(ctx, id, o)
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToStartContainer, id, err)
	}
	return nil
}

func (p *Podman) StopContainer(ctx context.Context, id string, opts *client.ContainerStopOptions) error {
	o := client.ContainerStopOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerStop(ctx, id, o)
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToStopContainer, id, err)
	}
	return nil
}

func (p *Podman) SuspendContainer(ctx context.Context, id string, opts *client.ContainerPauseOptions) error {
	o := client.ContainerPauseOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerPause(ctx, id, o)
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToSuspendContainer, id, err)
	}
	return nil
}

func (p *Podman) ResumeContainer(ctx context.Context, id string, opts *client.ContainerUnpauseOptions) error {
	o := client.ContainerUnpauseOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerUnpause(ctx, id, o)
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToResumeContainer, id, err)
	}
	return nil
}

func (p *Podman) DeleteContainer(ctx context.Context, id string, opts *client.ContainerRemoveOptions) error {
	o := client.ContainerRemoveOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerRemove(ctx, id, o)
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToDeleteContainer, id, err)
	}
	return nil
}

func (p *Podman) State(ctx context.Context, id string) (virt.NodeState, error) {
	resp, err := p.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToInspectContainer, id, err)
	}

	s := resp.Container.State
	switch {
	case s.Paused:
		return virt.NodeSuspended, nil
	case s.Restarting:
		return virt.NodeStarting, nil
	case s.Status == "running":
		return virt.NodeRunning, nil
	case s.Status == "created", s.Status == "exited", s.Status == "dead":
		return virt.NodeStopped, nil
	case s.Status == "removing":
		return virt.NodeStopping, nil
	default:
		return virt.NodeStopped, nil
	}
}

func (p *Podman) RestartContainer(ctx context.Context, id string, opts *client.ContainerRestartOptions) error {
	o := client.ContainerRestartOptions{}
	if opts != nil {
		o = *opts
	}
	_, err := p.cli.ContainerRestart(ctx, id, o)
	return err
}

func (p *Podman) CreateContainer(ctx context.Context, cfg *container.ContainerConfig) (*container.ContainerNode, error) {
	binds := make([]string, len(cfg.Volumes))
	for i, v := range cfg.Volumes {
		bind := v.Source + ":" + v.Target
		if v.ReadOnly {
			bind += ":ro"
		}
		binds[i] = bind
	}

	resp, err := p.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: cfg.Name,
		Config: &moby.Config{
			Image:      cfg.Image,
			Cmd:        cfg.Cmd,
			Entrypoint: cfg.Entrypoint,
			Env:        cfg.Env,
			Labels:     cfg.Labels,
			Hostname:   cfg.Hostname,
		},
		HostConfig: &moby.HostConfig{
			Binds:       binds,
			CapAdd:      cfg.CapAdd,
			CapDrop:     cfg.CapDrop,
			Privileged:  cfg.Privileged,
			NetworkMode: moby.NetworkMode(cfg.NetworkMode),
			DNS:         cfg.DNS,
			ExtraHosts:  cfg.ExtraHosts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToCreateContainer, cfg.Name, err)
	}

	return &container.ContainerNode{
		Node: virt.Node{
			ID:    resp.ID,
			Name:  cfg.Name,
			State: virt.NodeStopped,
		},
		ImageName: cfg.Image,
		Labels:    cfg.Labels,
	}, nil
}

func (p *Podman) Pull(ctx context.Context, ref string, opts *client.ImagePullOptions) error {
	o := client.ImagePullOptions{}
	if opts != nil {
		o = *opts
	}
	// some form of progress tracking would be cool and epic
	// and also setting identity which would be global tho
	// just make the client.ImagePullOptions configurable with a param
	resp, err := p.cli.ImagePull(ctx, ref, client.ImagePullOptions{
		All:           o.All,
		RegistryAuth:  o.RegistryAuth,
		PrivilegeFunc: o.PrivilegeFunc,
		Platforms:     o.Platforms,
	})
	if err != nil {
		return fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToPullImage, ref, err)
	}
	return resp.Wait(ctx)
}

func (p *Podman) Exec(ctx context.Context, id string, cmd []string, execOpts *client.ExecCreateOptions, attachOpts *client.ExecAttachOptions) (stdout, stderr []byte, err error) {
	eo := client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	if execOpts != nil {
		execOpts.Cmd = cmd
		execOpts.AttachStdout = true
		execOpts.AttachStderr = true
		eo = *execOpts
	}

	ao := client.ExecAttachOptions{}
	if attachOpts != nil {
		ao = *attachOpts
	}

	exec, err := p.cli.ExecCreate(ctx, id, eo)
	if err != nil {
		return nil, nil, fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToCreateExec, id, err)
	}

	resp, err := p.cli.ExecAttach(ctx, exec.ID, ao)
	if err != nil {
		return nil, nil, fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToAttachExec, id, err)
	}
	defer resp.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, resp.Conn); err != nil {
		return nil, nil, fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToStdCopy, id, err)
	}

	stdout = stdoutBuf.Bytes()
	stderr = stderrBuf.Bytes()

	return stdout, stderr, nil
}

func (p *Podman) Logs(ctx context.Context, id string, opts *client.ContainerLogsOptions) (io.ReadCloser, error) {
	o := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	if opts != nil {
		opts.ShowStdout = true
		opts.ShowStderr = true
		o = *opts
	}
	resp, err := p.cli.ContainerLogs(ctx, id, o)
	if err != nil {
		return nil, fmt.Errorf("podman: logs %q: %w", id, err)
	}
	return resp, nil
}

func (p *Podman) Inspect(ctx context.Context, id string, opts *client.ContainerInspectOptions) (*container.ContainerNode, error) {
	o := client.ContainerInspectOptions{}
	if opts != nil {
		o = *opts
	}
	resp, err := p.cli.ContainerInspect(ctx, id, o)
	if err != nil {
		return nil, fmt.Errorf("podman: %w %q: %w", errs.ErrFailedToInspectContainer, id, err)
	}

	var networks []string
	for name := range resp.Container.NetworkSettings.Networks {
		networks = append(networks, name)
	}

	return &container.ContainerNode{
		Node: virt.Node{
			ID:    resp.Container.ID,
			Name:  strings.TrimPrefix(resp.Container.Name, "/"),
			State: mapState(resp.Container.State),
		},
		ImageID:   resp.Container.Image,
		ImageName: resp.Container.Config.Image,
		Labels:    resp.Container.Config.Labels,
		Networks:  networks,
	}, nil
}

type stateInput interface {
	*moby.State | moby.ContainerState
}

func mapState[T stateInput](s T) virt.NodeState {
	switch v := any(s).(type) {
	case *moby.State:
		switch {
		case v.Paused:
			return virt.NodeSuspended
		case v.Restarting:
			return virt.NodeStarting
		case v.Running:
			return virt.NodeRunning
		case v.Status == moby.StateRemoving:
			return virt.NodeStopping
		default:
			return virt.NodeStopped
		}
	case moby.ContainerState:
		switch v {
		case moby.StateRunning:
			return virt.NodeRunning
		case moby.StatePaused:
			return virt.NodeSuspended
		case moby.StateRestarting:
			return virt.NodeStarting
		case moby.StateRemoving:
			return virt.NodeStopping
		default:
			return virt.NodeStopped
		}
	}
	return virt.NodeStopped
}

// for virt.Provider

func (p *Podman) Create(ctx context.Context, cfg *virt.NodeConfig) (*virt.Node, error) {
	if cfg == nil {
		return nil, fmt.Errorf("podman: create: cfg is nil")
	}
	node, err := p.CreateContainer(ctx, &container.ContainerConfig{
		NodeConfig: *cfg,
	})
	if err != nil {
		return nil, err
	}
	return &node.Node, nil
}

func (p *Podman) Start(ctx context.Context, id string) error {
	return p.StartContainer(ctx, id, nil)
}

func (p *Podman) Stop(ctx context.Context, id string) error {
	return p.StopContainer(ctx, id, nil)
}

func (p *Podman) Suspend(ctx context.Context, id string) error {
	return p.SuspendContainer(ctx, id, nil)
}

func (p *Podman) Resume(ctx context.Context, id string) error {
	return p.ResumeContainer(ctx, id, nil)
}

func (p *Podman) Delete(ctx context.Context, id string) error {
	return p.DeleteContainer(ctx, id, nil)
}

func (p *Podman) Restart(ctx context.Context, id string) error {
	return p.RestartContainer(ctx, id, nil)
}

func (p *Podman) Name() string {
	return "podman"
}

func (p *Podman) List(ctx context.Context) ([]*virt.Node, error) {
	result, err := p.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		// idk if i should have here a err type form the contianer spakcage bc this is a method of the virt provider enot the container provider
		return nil, fmt.Errorf("podman: list: %w", err)
	}
	var nodes []*virt.Node
	for i := range result.Items {
		name := ""
		if len(result.Items[i].Names) > 0 {
			name = strings.TrimPrefix(result.Items[i].Names[0], "/")
		}
		nodes = append(nodes, &virt.Node{
			ID:    result.Items[i].ID,
			Name:  name,
			State: mapState(result.Items[i].State),
		})
	}
	return nodes, nil
}

func (p *Podman) Reload(ctx context.Context, id string) (*virt.Node, error) {
	node, err := p.Inspect(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return &node.Node, nil
}
