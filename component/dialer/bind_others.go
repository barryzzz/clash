//go:build !linux && !darwin
// +build !linux,!darwin

package dialer

import (
	"net"
	"syscall"

	"github.com/Dreamacro/clash/component/iface"
	C "github.com/Dreamacro/clash/constant"
)

func lookupTCPAddr(ip net.IP, addrs []*net.IPNet) (*net.TCPAddr, error) {
	var addr *net.IPNet
	var err error

	if ip.To4() != nil {
		addr, err = iface.PickIPv4Addr(addrs)
	} else {
		addr, err = iface.PickIPv6Addr(addrs)
	}
	if err != nil {
		return nil, err
	}

	return &net.TCPAddr{IP: addr.IP, Port: 0}, nil
}

func lookupUDPAddr(ip net.IP, addrs []*net.IPNet) (*net.UDPAddr, error) {
	var addr *net.IPNet
	var err error

	if ip.To4() != nil {
		addr, err = iface.PickIPv4Addr(addrs)
	} else {
		addr, err = iface.PickIPv6Addr(addrs)
	}
	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{IP: addr.IP, Port: 0}, nil
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, network, _ string, destination net.IP) error {
	if !destination.IsGlobalUnicast() {
		return nil
	}

	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return err
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if addr, err := lookupTCPAddr(destination, ifaceObj.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	case "udp", "udp4", "udp6":
		if addr, err := lookupUDPAddr(destination, ifaceObj.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	}

	return nil
}

func bindIfaceToListenConfig(ifaceName string, _ *net.ListenConfig, _, _ string) (string, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return "", err
	}

	addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(addr.IP.String(), "0"), nil
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
			syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		})
	}
}
