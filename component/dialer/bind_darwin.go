package dialer

import (
	"net"
	"syscall"

	"github.com/Dreamacro/clash/component/iface"
)

type controlFn = func(network, address string, c syscall.RawConn) error

func bindControl(ifaceIdx int, chain controlFn) controlFn {
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
			switch network {
			case "tcp4", "udp4":
				syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_BOUND_IF, ifaceIdx)
			case "tcp6", "udp6":
				syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_BOUND_IF, ifaceIdx)
			}
		})
	}
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, _, _ string, _ net.IP) error {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return err
	}

	dialer.Control = bindControl(ifaceObj.Index, dialer.Control)
	return nil
}

func bindIfaceToListenConfig(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return "", err
	}

	lc.Control = bindControl(ifaceObj.Index, lc.Control)
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
