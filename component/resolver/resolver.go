package resolver

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"strings"
	"time"

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

	// DefaultDNSTimeout defined the default dns request timeout
	DefaultDNSTimeout = time.Second * 5
)

var (
	ErrIPNotFound   = errors.New("couldn't find ip")
	ErrIPVersion    = errors.New("ip version error")
	ErrIPv6Disabled = errors.New("ipv6 disabled")
)

type Resolver interface {
	ResolveIP(host string) (ip net.IP, err error)
	ResolveIPv4(host string) (ip net.IP, err error)
	ResolveIPv6(host string) (ip net.IP, err error)
}

// ResolveIPv4 with a host, return ipv4
func ResolveIPv4(host string) (net.IP, error) {
	if r := DefaultResolver; r != nil {
		return r.ResolveIPv4(host)
	}

	if ip := LookupHosts(host); ip != nil {
		if ip = ip.To4(); ip != nil {
			return ip, nil
		}

		return nil, ErrIPVersion
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if !strings.Contains(host, ":") {
			return ip, nil
		}
		return nil, ErrIPVersion
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultDNSTimeout)
	defer cancel()
	ipAddrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	} else if len(ipAddrs) == 0 {
		return nil, ErrIPNotFound
	}

	return ipAddrs[rand.Intn(len(ipAddrs))], nil
}

// ResolveIPv6 with a host, return ipv6
func ResolveIPv6(host string) (net.IP, error) {
	if DisableIPv6 {
		return nil, ErrIPv6Disabled
	}

	if ip := LookupHosts(host); ip != nil {
		if ip = ip.To16(); ip != nil {
			return ip, nil
		}

		return nil, ErrIPVersion
	}

	if node := DefaultHosts.Search(host); node != nil {
		if ip := node.Data.(net.IP).To16(); ip != nil {
			return ip, nil
		}

		return nil, ErrIPVersion
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if strings.Contains(host, ":") {
			return ip, nil
		}
		return nil, ErrIPVersion
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultDNSTimeout)
	defer cancel()
	ipAddrs, err := net.DefaultResolver.LookupIP(ctx, "ip6", host)
	if err != nil {
		return nil, err
	} else if len(ipAddrs) == 0 {
		return nil, ErrIPNotFound
	}

	return ipAddrs[rand.Intn(len(ipAddrs))], nil
}

// ResolveIP with a host, return ip
func ResolveIP(host string) (net.IP, error) {
	if r := DefaultResolver; r != nil {
		if DisableIPv6 {
			return r.ResolveIPv4(host)
		}
		return r.ResolveIP(host)
	}

	if DisableIPv6 {
		return ResolveIPv4(host)
	}

	if node := DefaultHosts.Search(host); node != nil {
		return node.Data.(net.IP), nil
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return ip, nil
	}

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, err
	}

	return ipAddr.IP, nil
}

// LookupHosts return ip with host from DefaultHosts
func LookupHosts(host string) net.IP {
	node := DefaultHosts.Search(host)
	if node == nil {
		return nil
	}

	return node.Data.(net.IP)
}
