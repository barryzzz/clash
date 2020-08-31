package resolver

import (
	"net"
)

var DefaultHostMapper HostMapper

type HostMapper interface {
	FakeIPEnabled() bool
	MappingEnabled() bool
	IsFakeIP(net.IP) bool
	ResolveHost(net.IP) (string, bool)
}

func FakeIPEnabled() bool {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.FakeIPEnabled()
	}

	return false
}

func MappingEnabled() bool {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.MappingEnabled()
	}

	return false
}

func IsFakeIP(ip net.IP) bool {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.IsFakeIP(ip)
	}

	return false
}

func ResolveHost(ip net.IP) (string, bool) {
	if mapper := DefaultHostMapper; mapper != nil {
		return mapper.ResolveHost(ip)
	}

	return "", false
}
