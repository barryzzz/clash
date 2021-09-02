//go:build !linux && !darwin
// +build !linux,!darwin

package dialer

import (
	"net"
	"strconv"
	"strings"

	"github.com/Dreamacro/clash/component/iface"
)

func lookupLocalAddr(ifaceName string, network string, destination net.IP, port int) (net.Addr, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	var addr *net.IPNet
	switch network {
	case "udp4", "tcp4":
		addr, err = iface.PickIPv4Addr(ifaceObj.Addrs, destination)
	case "tcp6", "udp6":
		addr, err = iface.PickIPv6Addr(ifaceObj.Addrs, destination)
	default:
		if destination != nil {
			if destination.To4() != nil {
				addr, err = iface.PickIPv4Addr(ifaceObj.Addrs, destination)
			} else {
				addr, err = iface.PickIPv6Addr(ifaceObj.Addrs, destination)
			}
		} else {
			addr, err = iface.PickIPv4Addr(ifaceObj.Addrs, destination)
		}
	}

	if strings.HasPrefix(network, "tcp") {
		return &net.TCPAddr{
			IP:   addr.IP,
			Port: port,
		}, nil
	} else if strings.HasPrefix(network, "udp") {
		return &net.UDPAddr{
			IP:   addr.IP,
			Port: port,
		}, nil
	}

	return nil, iface.ErrAddrNotFound
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, network string, destination net.IP) error {
	if !destination.IsGlobalUnicast() {
		return nil
	}

	local := 0
	if dialer.LocalAddr != nil {
		_, port, err := net.SplitHostPort(dialer.LocalAddr.String())
		if err == nil {
			local, _ = strconv.Atoi(port)
		}
	}

	addr, err := lookupLocalAddr(ifaceName, network, destination, local)
	if err != nil {
		return err
	}

	dialer.LocalAddr = addr

	return nil
}

func bindIfaceToListenConfig(ifaceName string, _ *net.ListenConfig, network, address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		port = "0"
	}

	local, _ := strconv.Atoi(port)

	addr, err := lookupLocalAddr(ifaceName, network, nil, local)
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}
