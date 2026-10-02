//go:build linux

package transport

import "syscall"

// bindControl returns a net.Dialer Control hook that forces the socket to egress
// the named interface via SO_BINDTODEVICE. This overrides the kernel's
// route-by-metric choice, which on a host with two interfaces on the same subnet
// would otherwise pick the wrong egress regardless of source address.
//
// Binding requires CAP_NET_RAW (or root); without it the dial fails with EPERM.
func bindControl(iface string) func(network, address string, c syscall.RawConn) error {
	return func(_, _ string, c syscall.RawConn) error {
		var sockErr error
		if err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, iface)
		}); err != nil {
			return err
		}
		return sockErr
	}
}
