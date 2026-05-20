package libsnb

import "syscall"

type RawSocketPort struct {
	id     string
	mtu    int
	fd     int
	ifName string
}

func (r *RawSocketPort) ID() string {
	return r.id
}

func (r *RawSocketPort) MTU() int {
	return r.mtu
}

func (r *RawSocketPort) ReadFrame() (Frame, error) {
	buf := make([]byte, 65536)
	n, _, err := syscall.Recvfrom(r.fd, buf, 0)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (r *RawSocketPort) WriteFrame(frame Frame) error {
	return syscall.Sendto(r.fd, frame, 0, nil)
}

func (r *RawSocketPort) Close() error {
	return syscall.Close(r.fd)
}
