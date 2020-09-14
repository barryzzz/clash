package dns

import (
	"net"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/component/fakeip"
)

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

func (h *ResolverEnhancer) Equals(o *ResolverEnhancer) bool {
	// check reusable

	if h.fakePool == o.fakePool {
		return true
	}

	if h.fakePool == nil || o.fakePool == nil {
		return false
	}

	return h.fakePool.EqualsIgnoreHosts(o.fakePool)
}

func (h *ResolverEnhancer) Patch(o *ResolverEnhancer) {
	if h.fakePool == nil || o.fakePool == nil {
		return
	}

	h.fakePool.PatchHosts(o.fakePool)
}

func NewEnhancer(cfg Config) *ResolverEnhancer {
	var fakePool *fakeip.Pool
	var mapping *cache.LruCache

	if cfg.EnhancedMode != NORMAL {
		fakePool = cfg.Pool
		mapping = cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true))
	}

	return &ResolverEnhancer{
		mode:     cfg.EnhancedMode,
		fakePool: fakePool,
		mapping:  mapping,
	}
}
