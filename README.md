# sme

`sme` contains a small Ethernet switch implementation in `internal/libsnb` and a Cobra CLI in `cmd/snb` for exercising it.

## Library sketch

```go
sw := libsnb.NewSwitch(libsnb.SwitchOptions{})

left, _ := libsnb.NewPort(libsnb.RawScheme, libsnb.PortOptions{
	ID:        "left",
	Interface: "vethA-host",
	MTU:       1500,
})
right, _ := libsnb.NewPort(libsnb.RawScheme, libsnb.PortOptions{
	ID:        "right",
	Interface: "vethB-host",
	MTU:       1500,
})

_ = sw.AttachPort(left)
_ = sw.AttachPort(right)
_ = sw.Start()
defer sw.Stop()
```

Linux namespace setup is still available as a helper:

```go
libsnb.AttachVethToNamespace(pid, "vethA-host", "eth0", 1500)
```

## CLI

Show the command tree:

```bash
go run ./cmd/snb --help
```

Run the automated tests:

```bash
go test ./...
```

If you want just the switch behavior tests:

```bash
go test ./internal/libsnb -run TestSwitch -v
```

### Usermode test flow

This mode stays in userspace and uses UDP sockets instead of raw sockets or namespace setup.

Start a three-port bridge:

```bash
go run ./cmd/snb bridge udp \
  --port left:127.0.0.1:10001:127.0.0.1:11001 \
  --port right:127.0.0.1:10002:127.0.0.1:11002 \
  --port tap:127.0.0.1:10003:127.0.0.1:11003
```

Listen for one forwarded frame:

```bash
go run ./cmd/snb frame listen --listen 127.0.0.1:11002 --count 1
```

Inject one test frame into the left side:

```bash
go run ./cmd/snb frame send \
  --to 127.0.0.1:10001 \
  --src 02:00:00:00:00:01 \
  --dst ff:ff:ff:ff:ff:ff \
  --payload hello
```

If you want a more explicit emulator-style topology, use the generic port form:

```bash
go run ./cmd/snb bridge ports \
  --port id=left,scheme=udp,listen=127.0.0.1:10001,remote=127.0.0.1:11001 \
  --port id=right,scheme=udp,listen=127.0.0.1:10002,remote=127.0.0.1:11002 \
  --port id=uplink,scheme=udp,listen=0.0.0.0:12000,remote=198.51.100.10:13000
```

That lets you expose a local UDP ingress port and forward traffic to a remote emulator endpoint or host.

### Mixed transport flow

The generic `bridge ports` command can mix `udp`, `raw`, and `pcap` ports in one switch:

```bash
sudo go run ./cmd/snb bridge ports \
  --port id=uplink,scheme=raw,if=eth0 \
  --port id=mirror,scheme=pcap,if=eth1,immediate=true \
  --port id=emulator,scheme=udp,listen=127.0.0.1:10001,remote=127.0.0.1:11001
```

Use this when you want to validate that raw sockets and pcap ports can both participate in the same switching graph while a UDP endpoint stands in for a remote emulator process.

You can also run the packaged no-sudo demo script:

```bash
./scripts/usermode-demo.sh
```

That script:

- starts a three-port `snb bridge udp`
- starts listeners for each egress side
- sends one broadcast frame and one learned unicast frame
- exits after the listeners receive their expected frames

### Namespace test flow

This mode needs root privileges and creates veth pairs for two Linux namespaces or containers.

Start two containers:

```bash
sudo podman run --name a --rm -it --cap-add=NET_ADMIN --net=none alpine
sudo podman run --name b --rm -it --cap-add=NET_ADMIN --net=none alpine
```

Bridge them:

```bash
sudo go run ./cmd/snb bridge namespace \
  "$(sudo podman inspect -f '{{.State.Pid}}' a)" \
  "$(sudo podman inspect -f '{{.State.Pid}}' b)"
```

Inside each container:

```bash
ip addr add 10.0.0.1/24 dev eth0
ip addr add 10.0.0.2/24 dev eth0
```

Useful flags:

- `--pcap` uses pcap instead of raw sockets.
- `--container-if` changes the namespace interface name.
- `--host-a` and `--host-b` change the host-side veth names.
