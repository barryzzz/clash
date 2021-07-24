package dialer

import (
	"errors"
	"net"

	IF "github.com/Dreamacro/clash/component/iface"
)

var (
	errPlatformNotSupport = errors.New("unsupport platform")
)

func lookupTCPAddr(ip net.IP, addrs []*net.IPNet) (*net.TCPAddr, error) {
	var addr *net.IPNet
	var err error

	if ip.To4() != nil {
		addr, err = IF.PickIPv4Addr(addrs)
	} else {
		addr, err = IF.PickIPv6Addr(addrs)
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
		addr, err = IF.PickIPv4Addr(addrs)
	} else {
		addr, err = IF.PickIPv6Addr(addrs)
	}
	if err != nil {
		return nil, err
	}

	return &net.UDPAddr{IP: addr.IP, Port: 0}, nil
}

func fallbackBindToDialer(dialer *net.Dialer, network string, ip net.IP, name string) error {
	if !ip.IsGlobalUnicast() {
		return nil
	}

	iface, err := IF.ResolveInterface(name)
	if err != nil {
		return err
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if addr, err := lookupTCPAddr(ip, iface.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	case "udp", "udp4", "udp6":
		if addr, err := lookupUDPAddr(ip, iface.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	}

	return nil
}

func fallbackBindToListenConfig(name string) (string, error) {
	iface, err := IF.ResolveInterface(name)
	if err != nil {
		return "", err
	}

	for _, addr := range iface.Addrs {
		return net.JoinHostPort(addr.IP.String(), "0"), nil
	}

	return "", ErrAddrNotFound
}
