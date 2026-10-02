//go:build !linux

package transport

import (
	"fmt"
	"syscall"

	"github.com/tj-smith47/shelly-go/types"
)

// bindControl returns a Control hook that fails every dial: there is no portable
// way to force a socket's egress interface outside Linux, and silently using the
// default route would defeat the caller's request.
func bindControl(iface string) func(network, address string, c syscall.RawConn) error {
	return func(_, _ string, _ syscall.RawConn) error {
		return fmt.Errorf("%w: binding to interface %q requires Linux", types.ErrNotSupported, iface)
	}
}
