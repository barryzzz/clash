package dns

import (
	"net"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/component/fakeip"
)

type HostMapper struct {
	mode     EnhancedMode
	fakePool *fakeip.Pool
	mapping  *cache.LruCache
}

func (h *HostMapper) FakeIPEnabled() bool {
	return h.mode == FAKEIP
}

func (h *HostMapper) MappingEnabled() bool {
	return h.mode == FAKEIP || h.mode == MAPPING
}

func (h *HostMapper) IsFakeIP(ip net.IP) bool {
	if !h.FakeIPEnabled() {
		return false
	}

	if pool := h.fakePool; pool != nil {
		return pool.Exist(ip)
	}

	return false
}

func (h *HostMapper) ResolveHost(ip net.IP) (string, bool) {
	if pool := h.fakePool; pool != nil {
		if host, existed := pool.LookBack(ip); existed {
			return host, true
		}
	}

	if mapping := h.mapping; mapping != nil {
		if host, existed := h.mapping.Get(ip.String()); existed {
			return host.(string), true
		}
	}

	return "", false
}

func (h *HostMapper) Equals(o *HostMapper) bool {
	// check reusable

	if h.FakeIPEnabled() == o.FakeIPEnabled() {
		if h.FakeIPEnabled() {
			return h.fakePool.Gateway().Equal(o.fakePool.Gateway())
		}

		return true
	}

	return false
}

func NewHostMapper(cfg Config) *HostMapper {
	var fakePool *fakeip.Pool
	var mapping *cache.LruCache

	if cfg.EnhancedMode != NORMAL {
		fakePool = cfg.Pool
		mapping = cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true))
	}

	return &HostMapper{
		mode:     cfg.EnhancedMode,
		fakePool: fakePool,
		mapping:  mapping,
	}
}
