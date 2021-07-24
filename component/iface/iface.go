package iface

import (
	"errors"
	"net"
	"time"

	"github.com/Dreamacro/clash/common/singledo"
)

type Interface struct {
	Index        int
	Name         string
	Addrs        []*net.IPNet
	HardwareAddr net.HardwareAddr
}

var ErrIfaceNotFound = errors.New("interface not found")
var ErrAddrNotFound = errors.New("addr not found")

var interfaces = singledo.NewSingle(time.Second * 20)

func ResolveInterface(name string) (*Interface, error) {
	value, err, _ := interfaces.Do(func() (interface{}, error) {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}

		r := make(map[string]*Interface)

		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			ipNets := make([]*net.IPNet, 0, len(addrs))
			for _, addr := range addrs {
				ipNet := addr.(*net.IPNet)
				if v4 := ipNet.IP.To4(); v4 != nil {
					ipNet.IP = v4
				}

				ipNets = append(ipNets, ipNet)
			}

			r[iface.Name] = &Interface{
				Index:        iface.Index,
				Name:         iface.Name,
				Addrs:        ipNets,
				HardwareAddr: iface.HardwareAddr,
			}
		}

		return r, nil
	})
	if err != nil {
		return nil, err
	}

	ifaces := value.(map[string]*Interface)
	iface, ok := ifaces[name]
	if !ok {
		return nil, ErrIfaceNotFound
	}

	return iface, nil
}

func PickIPv4Addr(addrs []*net.IPNet) (*net.IPNet, error) {
	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			return addr, nil
		}
	}

	return nil, ErrAddrNotFound
}

func PickIPv6Addr(addrs []*net.IPNet) (*net.IPNet, error) {
	for _, addr := range addrs {
		if addr.IP.To4() == nil {
			return addr, nil
		}
	}

	return nil, ErrAddrNotFound
}

func FlushCache() {
	interfaces.Reset()
}
