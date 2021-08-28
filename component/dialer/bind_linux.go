package dialer

import (
	"net"
	"syscall"
)

type controlFn = func(network, address string, c syscall.RawConn) error

func bindControl(ifaceName string, chain controlFn) controlFn {
	return func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		ipStr, _, err := net.SplitHostPort(address)
		if err == nil {
			ip := net.ParseIP(ipStr)
			if ip != nil && !ip.IsGlobalUnicast() {
				return
			}
		}

		return c.Control(func(fd uintptr) {
			syscall.BindToDevice(int(fd), ifaceName)
		})
	}
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, _, _ string, _ net.IP) error {
	dialer.Control = bindControl(ifaceName, dialer.Control)

	return nil
}

func bindIfaceToListenConfig(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	lc.Control = bindControl(ifaceName, lc.Control)

	return address, nil
}

func addrReuseToListenConfig(lc *net.ListenConfig) {
	chain := lc.Control

	lc.Control = func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		return c.Control(func(fd uintptr) {
			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		})
	}
}
