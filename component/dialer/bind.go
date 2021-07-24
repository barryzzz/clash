package dialer

import (
	"errors"
	"net"

	"github.com/Dreamacro/clash/component/iface"
)

var (
	errPlatformNotSupport = errors.New("unsupport platform")
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

func fallbackBindToDialer(dialer *net.Dialer, network string, ip net.IP, name string) error {
	if !ip.IsGlobalUnicast() {
		return nil
	}

	ifaceObj, err := iface.ResolveInterface(name)
	if err != nil {
		return err
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		if addr, err := lookupTCPAddr(ip, ifaceObj.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	case "udp", "udp4", "udp6":
		if addr, err := lookupUDPAddr(ip, ifaceObj.Addrs); err == nil {
			dialer.LocalAddr = addr
		} else {
			return err
		}
	}

	return nil
}

func fallbackBindToListenConfig(name string) (string, error) {
	ifaceObj, err := iface.ResolveInterface(name)
	if err != nil {
		return "", err
	}

	addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(addr.IP.String(), "0"), nil
}
