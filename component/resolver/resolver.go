package resolver

import (
	"errors"
	"net"
	"strings"

	trie "github.com/Dreamacro/clash/component/domain-trie"
)

var (
	// DefaultResolver aim to resolve ip
	DefaultResolver Resolver

	// DefaultHosts aim to resolve hosts
	DefaultHosts = trie.New()
)

var (
	ErrIPNotFound = errors.New("couldn't find ip")
	ErrIPVersion  = errors.New("ip version error")
)

type ResolvedIP struct {
	V4 net.IP
	V6 net.IP
}

type Resolver interface {
	ResolveIP(host string) (ip *ResolvedIP, err error)
	ResolveIPv4(host string) (ip *ResolvedIP, err error)
	ResolveIPv6(host string) (ip *ResolvedIP, err error)
}

func (r *ResolvedIP) IPv6Available() bool {
	return len(r.V6) == 16
}

func (r *ResolvedIP) IPv4Available() bool {
	return len(r.V4) == 4
}

func (r *ResolvedIP) SingleIP() net.IP {
	if r.IPv4Available() {
		return r.V4
	}
	if r.IPv6Available() {
		return r.V6
	}
	return nil
}

// ResolveIPv4 with a host, return ipv4
func ResolveIPv4(host string) (*ResolvedIP, error) {
	if node := DefaultHosts.Search(host); node != nil {
		if ip := node.Data.(net.IP).To4(); ip != nil {
			return ResolvedIPFromSingle(ip), nil
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if !strings.Contains(host, ":") {
			return ResolvedIPFromSingle(ip), nil
		}
		return nil, ErrIPVersion
	}

	if DefaultResolver != nil {
		return DefaultResolver.ResolveIPv4(host)
	}

	ipAddrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ipAddrs {
		if ip4 := ip.To4(); ip4 != nil {
			return ResolvedIPFromSingle(ip4), nil
		}
	}

	return nil, ErrIPNotFound
}

// ResolveIPv6 with a host, return ipv6
func ResolveIPv6(host string) (*ResolvedIP, error) {
	if node := DefaultHosts.Search(host); node != nil {
		if ip := node.Data.(net.IP).To16(); ip != nil {
			return ResolvedIPFromSingle(ip), nil
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if strings.Contains(host, ":") {
			return ResolvedIPFromSingle(ip), nil
		}
		return nil, ErrIPVersion
	}

	if DefaultResolver != nil {
		return DefaultResolver.ResolveIPv6(host)
	}

	ipAddrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ipAddrs {
		if ip.To4() == nil {
			return ResolvedIPFromSingle(ip), nil
		}
	}

	return nil, ErrIPNotFound
}

// ResolveIP with a host, return ip
func ResolveIP(host string) (*ResolvedIP, error) {
	if node := DefaultHosts.Search(host); node != nil {
		ip := node.Data.(net.IP)

		return ResolvedIPFromSingle(ip), nil
	}

	if DefaultResolver != nil {
		return DefaultResolver.ResolveIP(host)
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return ResolvedIPFromSingle(ip), nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	return ResolvedIPFromList(ips)
}

func ResolvedIPFromSingle(ip net.IP) *ResolvedIP {
	return &ResolvedIP{
		V4: ip.To4(),
		V6: ip.To16(),
	}
}

func ResolvedIPFromList(ips []net.IP) (*ResolvedIP, error) {
	resolved := &ResolvedIP{}

	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			resolved.V4 = v4
		} else {
			resolved.V6 = ip.To16()
		}
	}

	if len(resolved.V4) == 0 && len(resolved.V6) == 0 {
		return nil, ErrIPNotFound
	}

	return resolved, nil
}
