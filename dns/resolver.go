package dns

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"

	DM "golang.org/x/net/dns/dnsmessage"
	"golang.org/x/sync/singleflight"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/component/dialer"
	D "github.com/Dreamacro/clash/component/dns"
	"github.com/Dreamacro/clash/component/fakeip"
	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/component/trie"
)

type module interface {
	ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error)
}

type Resolver struct {
	ipv6   bool
	group  singleflight.Group
	cache  *cache.LruCache
	client module
}

// ResolveIP request with TypeA and TypeAAAA, priority return TypeA
func (r *Resolver) ResolveIP(host string) (ip net.IP, err error) {
	ch := make(chan net.IP, 1)
	go func() {
		defer close(ch)
		ip, err := r.resolveIP(host, DM.TypeAAAA)
		if err != nil {
			return
		}
		ch <- ip
	}()

	ip, err = r.resolveIP(host, DM.TypeAAAA)
	if err == nil {
		return
	}

	ip, open := <-ch
	if !open {
		return nil, resolver.ErrIPNotFound
	}

	return ip, nil
}

// ResolveIPv4 request with TypeA
func (r *Resolver) ResolveIPv4(host string) (ip net.IP, err error) {
	return r.resolveIP(host, DM.TypeA)
}

// ResolveIPv6 request with TypeAAAA
func (r *Resolver) ResolveIPv6(host string) (ip net.IP, err error) {
	return r.resolveIP(host, DM.TypeAAAA)
}

// Exchange a batch of dns request, and it use cache
func (r *Resolver) Exchange(msg *DM.Message) (*DM.Message, error) {
	if len(msg.Questions) == 0 {
		return nil, errors.New("should have one question at least")
	}
	if !r.ipv6 && msg.Questions[0].Type == DM.TypeAAAA {
		return &DM.Message{
			Header: DM.Header{
				ID:                 msg.Header.ID,
				Response:           true,
				RecursionAvailable: true,
				RCode:              DM.RCodeSuccess,
			},
			Questions: msg.Questions,
		}, nil
	}

	q := msg.Questions[0]
	cached, expireTime, hit := r.cache.GetWithExpire(q.GoString())
	if hit {
		now := time.Now()
		reply := D.ShallowCloneMessage(cached.(*DM.Message))
		reply.ID = msg.ID

		if expireTime.Before(now) {
			D.OverrideTTLOfMessage(reply, uint32(1)) // Continue fetch
			go r.exchange(reply)
		} else {
			D.OverrideTTLOfMessage(reply, uint32(time.Until(expireTime).Seconds()))
		}

		return reply, nil
	}
	return r.exchange(msg)
}

func (r *Resolver) exchange(msg *DM.Message) (*DM.Message, error) {
	ret, err, shared := r.group.Do(msg.Questions[0].GoString(), func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
		defer cancel()

		reply, err := r.client.ExchangeContext(ctx, msg)
		if err == nil {
			r.putCache(reply)
		}
		return reply, err
	})
	if err != nil {
		return nil, err
	}

	reply := ret.(*DM.Message)
	if shared {
		reply = D.ShallowCloneMessage(reply)

		reply.ID = msg.ID
	}

	return reply, nil
}

func (r *Resolver) resolveIP(host string, dnsType DM.Type) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		isIPv4 := ip.To4() != nil
		if dnsType == DM.TypeAAAA && !isIPv4 {
			return ip, nil
		} else if dnsType == DM.TypeA && isIPv4 {
			return ip, nil
		} else {
			return nil, resolver.ErrIPVersion
		}
	}

	if ip := resolver.LookupHosts(host); ip != nil {
		if v4 := ip.To4(); v4 != nil && dnsType == DM.TypeA {
			return v4, nil
		} else if v6 := ip.To16(); v6 != nil && dnsType == DM.TypeAAAA {
			return v6, nil
		}

		return nil, resolver.ErrIPVersion
	}

	name, err := DM.NewName(D.Fqdn(host))
	if err != nil {
		return nil, err
	}

	msg := &DM.Message{
		Header: DM.Header{
			ID:               uint16(rand.Uint32()),
			RecursionDesired: true,
		},
		Questions: []DM.Question{{
			Name:  name,
			Type:  dnsType,
			Class: DM.ClassINET,
		}},
	}

	reply, err := r.Exchange(msg)
	if err != nil {
		return nil, err
	}

	ips := D.ExtractIPsFromMessage(reply)
	if len(ips) == 0 {
		return nil, resolver.ErrIPNotFound
	}

	return ips[rand.Intn(len(ips))], nil
}

func (r *Resolver) putCache(msg *DM.Message) {
	var ttl = dnsDefaultTTL

	switch {
	case len(msg.Answers) != 0:
		ttl = msg.Answers[0].Header.TTL
	case len(msg.Authorities) != 0:
		ttl = msg.Authorities[0].Header.TTL
	case len(msg.Additionals) != 0:
		ttl = msg.Additionals[0].Header.TTL
	}

	r.cache.SetWithExpire(msg.Questions[0].GoString(), D.ShallowCloneMessage(msg), time.Now().Add(time.Second*time.Duration(ttl)))
}

type NameServer struct {
	Net  string
	Addr string
}

type FallbackFilter struct {
	GeoIP  bool
	IPCIDR []*net.IPNet
	Domain []string
}

type Config struct {
	Main, Fallback []NameServer
	Default        []NameServer
	IPv6           bool
	EnhancedMode   EnhancedMode
	FallbackFilter FallbackFilter
	Pool           *fakeip.Pool
	Hosts          *trie.DomainTrie
	Policy         map[string]NameServer
}

func NewResolver(config Config) *Resolver {
	defaultResolver := &Resolver{
		ipv6:   config.IPv6,
		cache:  cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true)),
		client: &parallel{modules: transformClients(config.Default, dialer.DialContext)},
	}

	dialWithResolver := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		ip := net.ParseIP(host)
		if ip == nil {
			ip, err := defaultResolver.ResolveIP(host)
			if err != nil {
				return nil, err
			}

			host = ip.String()
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
	}

	var up module = &parallel{modules: transformClients(config.Main, dialWithResolver)}

	if len(config.Fallback) != 0 {
		var filters []filter
		if config.FallbackFilter.GeoIP {
			filters = append(filters, &geoipFilter{})
		}
		if len(config.FallbackFilter.IPCIDR) != 0 {
			filters = append(filters, &ipcidrFilter{ipNets: config.FallbackFilter.IPCIDR})
		}
		if len(config.FallbackFilter.Domain) != 0 {
			domains := trie.New()
			for _, domain := range config.FallbackFilter.Domain {
				domains.Insert(domain, "")
			}

			filters = append(filters, &domainFilter{domains: domains})
		}

		up = &fallback{
			main:     up,
			fallback: &parallel{transformClients(config.Fallback, dialWithResolver)},
			filters:  filters,
		}
	}

	if len(config.Policy) != 0 {
		policies := trie.New()
		for domain, nameserver := range config.Policy {
			policies.Insert(domain, transformClients([]NameServer{nameserver}, dialWithResolver)[0])
		}

		up = &policy{
			policies: policies,
			fallback: up,
		}
	}

	return &Resolver{
		ipv6:   config.IPv6,
		cache:  cache.NewLRUCache(cache.WithSize(4096), cache.WithStale(true)),
		client: up,
	}
}
