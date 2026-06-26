package container

import (
	"io"

	"github.com/0xveya/tethux/internal/libtethux/virt"
)

type ContainerProvider interface {
	virt.Provider

	CreateContainer(cfg ContainerConfig) (*ContainerNode, error)
	Pull(image, tag, registry string) error
	Exec(id string, cmd []string) error
	Logs(id string) (io.ReadCloser, error)
	Inspect(id string) (*ContainerNode, error)
}

type ContainerConfig struct {
	virt.NodeConfig

	Registry string
	Tag      string
	Digest   string

	Entrypoint  []string
	Cmd         []string
	Env         []string
	Volumes     []VolumeMount
	CapAdd      []string
	CapDrop     []string
	Privileged  bool
	NetworkMode string

	Hostname   string
	DNS        []string
	ExtraHosts []string

	Labels map[string]string
}

type VolumeMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

type ContainerNode struct {
	virt.Node

	ImageID   string
	ImageName string
	Labels    map[string]string
	Networks  []string
}
