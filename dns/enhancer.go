package dns

import (
	"net"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/component/fakeip"
)

var globalEnhancerMapping = cache.NewLRUCache(cache.WithSize(4096))
var globalEnhancerFakeIPMapping = fakeip.NewCache(4096)

type ResolverEnhancer struct {
	mode     EnhancedMode
	fakePool *fakeip.Pool
	mapping  *cache.LruCache
}

func (h *ResolverEnhancer) FakeIPEnabled() bool {
	return h.mode == FAKEIP
}

func (h *ResolverEnhancer) MappingEnabled() bool {
	return h.mode == FAKEIP || h.mode == MAPPING
}

func (h *ResolverEnhancer) IsFakeIP(ip net.IP) bool {
	if !h.FakeIPEnabled() {
		return false
	}

	if pool := h.fakePool; pool != nil {
		return pool.Exist(ip)
	}

	return false
}

func (h *ResolverEnhancer) FindHostByIP(ip net.IP) (string, bool) {
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

func NewEnhancer(cfg Config) *ResolverEnhancer {
	var mapping *cache.LruCache
	var fakePool *fakeip.Pool

	if cfg.EnhancedMode != NORMAL {
		mapping = globalEnhancerMapping
	}

	if cfg.EnhancedMode == FAKEIP {
		fakePool = fakeip.New(cfg.FakeIPRange, cfg.FakeIPFilter, globalEnhancerFakeIPMapping)
	}

	return &ResolverEnhancer{
		mode:     cfg.EnhancedMode,
		fakePool: fakePool,
		mapping:  mapping,
	}
}
