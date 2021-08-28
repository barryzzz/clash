//go:build !linux && !darwin
// +build !linux,!darwin

package dialer

import (
	"net"
	"strconv"
	"syscall"

	"github.com/Dreamacro/clash/component/iface"
)

func lookupTCPAddr(ip net.IP, addrs []*net.IPNet, port string) (*net.TCPAddr, error) {
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

	p, _ := strconv.Atoi(port)
	return &net.TCPAddr{IP: addr.IP, Port: p}, nil
}

func lookupUDPAddr(ip net.IP, addrs []*net.IPNet, port string) (*net.UDPAddr, error) {
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

	p, _ := strconv.Atoi(port)
	return &net.UDPAddr{IP: addr.IP, Port: p}, nil
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, network, address string, destination net.IP) error {
	if !destination.IsGlobalUnicast() {
		return nil
	}

	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return err
	}

	_, port, err := net.SplitHostPort(dialer.LocalAddr.String())
	if err != nil {
		port = "0"
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if addr, err := lookupTCPAddr(destination, ifaceObj.Addrs, port); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	case "udp", "udp4", "udp6":
		if addr, err := lookupUDPAddr(destination, ifaceObj.Addrs, port); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	}

	return nil
}

func bindIfaceToListenConfig(ifaceName string, _ *net.ListenConfig, _, address string) (string, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return "", err
	}

	addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
	if err != nil {
		return "", err
	}

	_, port, err := net.SplitHostPort(address)
	if err != nil {
		port = "0"
	}

	return net.JoinHostPort(addr.IP.String(), port), nil
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
