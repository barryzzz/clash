package dns

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/common/picker"
	"github.com/Dreamacro/clash/component/fakeip"
	"github.com/Dreamacro/clash/component/resolver"

	D "github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

var (
	globalSessionCache = tls.NewLRUClientSessionCache(64)
)

type dnsClient interface {
	Exchange(m *D.Msg) (msg *D.Msg, err error)
	ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error)
}

type result struct {
	Msg   *D.Msg
	Error error
}

type Resolver struct {
	ipv6            bool
	mapping         bool
	fakeip          bool
	pool            *fakeip.Pool
	main            []dnsClient
	fallback        []dnsClient
	fallbackFilters []fallbackFilter
	group           singleflight.Group
	cache           *cache.Cache
}

// ResolveIP request with TypeA and TypeAAAA, priority return TypeA
func (r *Resolver) ResolveIP(host string) (*resolver.ResolvedIP, error) {
	resolved := &resolver.ResolvedIP{}

	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ip, err := r.resolveIP(host, D.TypeAAAA)
		if err == nil {
			resolved.V6 = ip.V6
		}
		ch <- err
	}()

	ip, err := r.resolveIP(host, D.TypeA)
	if err != nil {
		return nil, err
	} else {
		resolved.V4 = ip.V4
	}

	err = <-ch
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

// ResolveIPv4 request with TypeA
func (r *Resolver) ResolveIPv4(host string) (ip *resolver.ResolvedIP, err error) {
	return r.resolveIP(host, D.TypeA)
}

// ResolveIPv6 request with TypeAAAA
func (r *Resolver) ResolveIPv6(host string) (ip *resolver.ResolvedIP, err error) {
	return r.resolveIP(host, D.TypeAAAA)
}

func (r *Resolver) shouldFallback(ip net.IP) bool {
	for _, filter := range r.fallbackFilters {
		if filter.Match(ip) {
			return true
		}
	}
	return false
}

// Exchange a batch of dns request, and it use cache
func (r *Resolver) Exchange(m *D.Msg) (msg *D.Msg, err error) {
	if len(m.Question) == 0 {
		return nil, errors.New("should have one question at least")
	}

	q := m.Question[0]
	cache, expireTime := r.cache.GetWithExpire(q.String())
	if cache != nil {
		msg = cache.(*D.Msg).Copy()
		setMsgTTL(msg, uint32(expireTime.Sub(time.Now()).Seconds()))
		return
	}
	defer func() {
		if msg == nil {
			return
		}

		putMsgToCache(r.cache, q.String(), msg)
		if r.mapping {
			ips := r.msgToIP(msg)
			for _, ip := range ips {
				putMsgToCache(r.cache, ip.String(), msg)
			}
		}
	}()

	ret, err, shared := r.group.Do(q.String(), func() (interface{}, error) {
		isIPReq := isIPRequest(q)
		if isIPReq {
			return r.fallbackExchange(m)
		}

		return r.batchExchange(r.main, m)
	})

	if err == nil {
		msg = ret.(*D.Msg)
		if shared {
			msg = msg.Copy()
		}
	}

	return
}

// IPToHost return fake-ip or redir-host mapping host
func (r *Resolver) IPToHost(ip *resolver.ResolvedIP) (string, bool) {
	if r.fakeip {
		if ip.IPv4Available() {
			cached4, exists := r.pool.LookBack(ip.V4)
			if exists {
				return cached4, true
			}
		}

		if ip.IPv6Available() {
			cached6, exists := r.pool.LookBack(ip.V6)
			if exists {
				return cached6, true
			}
		}
	}

	var cached interface{}
	if ip.IPv4Available() {
		cached = r.cache.Get(ip.V4.String())
	}
	if cached == nil && ip.IPv6Available() {
		cached = r.cache.Get(ip.V6.String())
	}

	if cached == nil {
		return "", false
	}

	fqdn := cached.(*D.Msg).Question[0].Name
	return strings.TrimRight(fqdn, "."), true
}

func (r *Resolver) IsMapping() bool {
	return r.mapping
}

// FakeIPEnabled returns if fake-ip is enabled
func (r *Resolver) FakeIPEnabled() bool {
	return r.fakeip
}

// IsFakeIP determine if given ip is a fake-ip
func (r *Resolver) IsFakeIP(ip *resolver.ResolvedIP) bool {
	if r.FakeIPEnabled() {
		return r.pool.Exist(ip.V4) || r.pool.Exist(ip.V6)
	}
	return false
}

func (r *Resolver) batchExchange(clients []dnsClient, m *D.Msg) (msg *D.Msg, err error) {
	fast, ctx := picker.WithTimeout(context.Background(), time.Second*5)
	for _, client := range clients {
		r := client
		fast.Go(func() (interface{}, error) {
			m, err := r.ExchangeContext(ctx, m)
			if err != nil {
				return nil, err
			} else if m.Rcode == D.RcodeServerFailure || m.Rcode == D.RcodeRefused {
				return nil, errors.New("server failure")
			}
			return m, nil
		})
	}

	elm := fast.Wait()
	if elm == nil {
		err := errors.New("All DNS requests failed")
		if fErr := fast.Error(); fErr != nil {
			err = fmt.Errorf("%w, first error: %s", err, fErr.Error())
		}
		return nil, err
	}

	msg = elm.(*D.Msg)
	return
}

func (r *Resolver) fallbackExchange(m *D.Msg) (msg *D.Msg, err error) {
	msgCh := r.asyncExchange(r.main, m)
	if r.fallback == nil {
		res := <-msgCh
		msg, err = res.Msg, res.Error
		return
	}
	fallbackMsg := r.asyncExchange(r.fallback, m)
	res := <-msgCh
	if res.Error == nil {
		if ips := r.msgToIP(res.Msg); len(ips) != 0 {
			if !r.shouldFallback(ips[0]) {
				msg = res.Msg
				err = res.Error
				return msg, err
			}
		}
	}

	res = <-fallbackMsg
	msg, err = res.Msg, res.Error
	return
}

func (r *Resolver) resolveIP(host string, dnsType uint16) (*resolver.ResolvedIP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		isIPv4 := ip.To4() != nil
		if dnsType == D.TypeAAAA && !isIPv4 {
			return resolver.ResolvedIPFromSingle(ip), nil
		} else if dnsType == D.TypeA && isIPv4 {
			return resolver.ResolvedIPFromSingle(ip), nil
		} else {
			return nil, resolver.ErrIPVersion
		}
	}

	query := &D.Msg{}
	query.SetQuestion(D.Fqdn(host), dnsType)

	msg, err := r.Exchange(query)
	if err != nil {
		return nil, err
	}

	ips := r.msgToIP(msg)
	ipLength := len(ips)
	if ipLength == 0 {
		return nil, resolver.ErrIPNotFound
	}

	ip = ips[rand.Intn(ipLength)]

	return resolver.ResolvedIPFromSingle(ip), nil
}

func (r *Resolver) msgToIP(msg *D.Msg) []net.IP {
	ips := []net.IP{}

	for _, answer := range msg.Answer {
		switch ans := answer.(type) {
		case *D.AAAA:
			ips = append(ips, ans.AAAA)
		case *D.A:
			ips = append(ips, ans.A)
		}
	}

	return ips
}

func (r *Resolver) asyncExchange(client []dnsClient, msg *D.Msg) <-chan *result {
	ch := make(chan *result, 1)
	go func() {
		res, err := r.batchExchange(client, msg)
		ch <- &result{Msg: res, Error: err}
	}()
	return ch
}

type NameServer struct {
	Net  string
	Addr string
}

type FallbackFilter struct {
	GeoIP  bool
	IPCIDR []*net.IPNet
}

type Config struct {
	Main, Fallback []NameServer
	Default        []NameServer
	IPv6           bool
	EnhancedMode   EnhancedMode
	FallbackFilter FallbackFilter
	Pool           *fakeip.Pool
}

func New(config Config) *Resolver {
	defaultResolver := &Resolver{
		main:  transform(config.Default, nil),
		cache: cache.New(time.Second * 60),
	}

	r := &Resolver{
		ipv6:    config.IPv6,
		main:    transform(config.Main, defaultResolver),
		cache:   cache.New(time.Second * 60),
		mapping: config.EnhancedMode == MAPPING,
		fakeip:  config.EnhancedMode == FAKEIP,
		pool:    config.Pool,
	}

	if len(config.Fallback) != 0 {
		r.fallback = transform(config.Fallback, defaultResolver)
	}

	fallbackFilters := []fallbackFilter{}
	if config.FallbackFilter.GeoIP {
		fallbackFilters = append(fallbackFilters, &geoipFilter{})
	}
	for _, ipnet := range config.FallbackFilter.IPCIDR {
		fallbackFilters = append(fallbackFilters, &ipnetFilter{ipnet: ipnet})
	}
	r.fallbackFilters = fallbackFilters

	return r
}
