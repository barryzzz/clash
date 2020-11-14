package resolver

import (
	"errors"
	"net"
	"strings"

	"github.com/Dreamacro/clash/component/trie"
)

var (
	// DefaultResolver aim to resolve ip
	DefaultResolver Resolver

	// DisableIPv6 means don't resolve ipv6 host
	// default value is true
	DisableIPv6 = true

	// DefaultHosts aim to resolve hosts
	DefaultHosts = trie.New()
)

var (
	ErrIPNotFound   = errors.New("couldn't find ip")
	ErrIPVersion    = errors.New("ip version error")
	ErrIPv6Disabled = errors.New("ipv6 disabled")
)

type Resolver interface {
	ResolveIPs(host string, v4, v6 bool) (ip []net.IP, err error)
}

// ResolveIPv4 with a host, return ipv4
func ResolveIPv4(host string) (net.IP, error) {
	if node := DefaultHosts.Search(host); node != nil {
		if ip := node.Data.(net.IP).To4(); ip != nil {
			return ip, nil
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if !strings.Contains(host, ":") {
			return ip, nil
		}
		return nil, ErrIPVersion
	}

	var addrs []net.IP
	var err error

	if DefaultResolver != nil {
		addrs, err = DefaultResolver.ResolveIPs(host, true, false)
	} else {
		addrs, err = net.LookupIP(host)
	}

	if err != nil {
		return nil, err
	}

	for _, ip := range addrs {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, nil
		}
	}

	return nil, ErrIPNotFound
}

// ResolveIPv6 with a host, return ipv6
func ResolveIPv6(host string) (net.IP, error) {
	if DisableIPv6 {
		return nil, ErrIPv6Disabled
	}

	if node := DefaultHosts.Search(host); node != nil {
		if ip := node.Data.(net.IP).To16(); ip != nil {
			return ip, nil
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if strings.Contains(host, ":") {
			return ip, nil
		}
		return nil, ErrIPVersion
	}

	var addrs []net.IP
	var err error

	if DefaultResolver != nil {
		addrs, err = DefaultResolver.ResolveIPs(host, false, true)
	} else {
		addrs, err = net.LookupIP(host)
	}

	if err != nil {
		return nil, err
	}

	for _, ip := range addrs {
		if ip16 := ip.To16(); ip16 != nil {
			return ip16, nil
		}
	}

	return nil, ErrIPNotFound
}

// ResolveIP with a host, return ip
func ResolveIP(host string) (net.IP, error) {
	if node := DefaultHosts.Search(host); node != nil {
		return node.Data.(net.IP), nil
	}

	var addrs []net.IP
	var err error

	if DefaultResolver != nil {
		addrs, err = DefaultResolver.ResolveIPs(host, true, !DisableIPv6)
	} else {
		addrs, err = net.LookupIP(host)
	}

	if err != nil {
		return nil, err
	}

	if DisableIPv6 {
		for _, ip := range addrs {
			if ip4 := ip.To4(); ip4 != nil {
				return ip4, nil
			}
		}
	} else if len(addrs) > 0 {
		return addrs[0], nil
	}

	return nil, ErrIPNotFound
}

func ResolveIPs(host string) ([]net.IP, error) {
	if DefaultResolver != nil {
		return DefaultResolver.ResolveIPs(host, true, !DisableIPv6)
	}

	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	if DisableIPv6 {
		filtered := make([]net.IP, 0, len(addrs))

		for _, ip := range addrs {
			if ip4 := ip.To4(); ip4 != nil {
				filtered = append(filtered, ip4)
			}
		}

		addrs = filtered
	}

	if len(addrs) > 0 {
		return addrs, nil
	}

	return nil, ErrIPNotFound
}
