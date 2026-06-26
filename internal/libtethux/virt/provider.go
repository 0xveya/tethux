package virt

type ConsoleType string

const (
	ConsoleNone   ConsoleType = "none"
	ConsoleTelnet ConsoleType = "telnet"
	ConsoleSerial ConsoleType = "serial"
	ConsoleVNC    ConsoleType = "vnc"
	ConsoleSpice  ConsoleType = "spice"
	ConsoleAux    ConsoleType = "aux"
)

type Console struct {
	Type ConsoleType
	Host string
	Port uint16
}

type NodeState string

const (
	NodeStopped   NodeState = "stopped"
	NodeStarting  NodeState = "starting"
	NodeRunning   NodeState = "running"
	NodeStopping  NodeState = "stopping"
	NodeSuspended NodeState = "suspended"
)

type NodeConfig struct {
	ID             string
	Name           string
	Image          string
	CPUs           int
	MemoryMB       int
	ConsoleType    ConsoleType
	AuxConsoleType ConsoleType
	Meta           map[string]string
}

type Node struct {
	ID      string
	Name    string
	State   NodeState
	Console Console
	Aux     *Console // nil if no auxiliary console
}

type Provider interface {
	Name() string

	Create(cfg NodeConfig) (*Node, error)
	Start(id string) error
	Stop(id string) error
	Suspend(id string) error
	Resume(id string) error
	Delete(id string) error

	State(id string) (NodeState, error)
	Reload(id string) (*Node, error)

	List() ([]*Node, error)
}

type ServerProvider interface {
	Provider

	StartServer() error
	StopServer() error
	ServerRunning() bool
	ServerAddr() string
}
