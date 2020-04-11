package outboundgroup

import (
	"encoding/json"
	"github.com/Dreamacro/clash/adapters/outbound"
	"github.com/Dreamacro/clash/adapters/provider"
	"github.com/Dreamacro/clash/common/singledo"
	C "github.com/Dreamacro/clash/constant"
)

type Relay struct {
	*outbound.Base
	single    *singledo.Single
	providers []provider.ProxyProvider
}


func (r *Relay) Dialer(dialer C.ProxyDialer) C.ProxyDialer {
	proxies := r.rawProxies()

	for _, p := range proxies {
		dialer = p.Dialer(dialer)
	}

	return newGroupDialer(r, dialer)
}

func (r *Relay) MarshalJSON() ([]byte, error) {
	var all []string
	for _, proxy := range r.rawProxies() {
		all = append(all, proxy.Name())
	}
	return json.Marshal(map[string]interface{}{
		"type": r.Type().String(),
		"all":  all,
	})
}

func (r *Relay) rawProxies() []C.Proxy {
	elm, _, _ := r.single.Do(func() (interface{}, error) {
		return getProvidersProxies(r.providers), nil
	})

	return elm.([]C.Proxy)
}

func NewRelay(name string, providers []provider.ProxyProvider) *Relay {
	return &Relay{
		Base:      outbound.NewBase(name, "", C.Relay, false),
		single:    singledo.NewSingle(defaultGetProxiesDuration),
		providers: providers,
	}
}
