package dialer

import (
	"net"
	"syscall"
)

func addrReuseToListenConfig(lc *net.ListenConfig) {
	chain := lc.Control

	lc.Control = func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		return c.Control(func(fd uintptr) {
			syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		})
	}
}
