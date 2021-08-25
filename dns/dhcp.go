package dns

import (
	"bytes"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	D "github.com/miekg/dns"

	"github.com/Dreamacro/clash/component/dhcp"
	"github.com/Dreamacro/clash/component/iface"
	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/log"
)

const (
	IfaceTTL = time.Second * 20
	DHCPTTL  = time.Minute * 10
)

var (
	ErrDHCPUnavailable = errors.New("dhcp unavailable")
)

type dhcpClient struct {
	ifaceName string

	lock            sync.Mutex
	ifaceInvalidate time.Time
	dnsInvalidate   time.Time

	ifaceAddr *net.IPNet
	resolver  *Resolver
}

func (d *dhcpClient) Exchange(m *D.Msg) (msg *D.Msg, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
	defer cancel()

	return d.ExchangeContext(ctx, m)
}

func (d *dhcpClient) ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	res := d.resolve(ctx)
	if res == nil {
		return nil, ErrDHCPUnavailable
	}

	return res.ExchangeContext(ctx, m)
}

func (d *dhcpClient) resolve(ctx context.Context) *Resolver {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.ifaceInvalidate.Before(time.Now()) {
		return d.resolver
	}

	var err error
	defer func() {
		if err != nil {
			log.Debugln("Resolve DNS from DHCP: %v", err)

			d.resolver = nil
		}
	}()

	d.ifaceInvalidate = time.Now().Add(IfaceTTL)

	ifaceObj, err := iface.ResolveInterface(d.ifaceName)
	if err != nil {
		return nil
	}

	addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
	if err != nil {
		return nil
	}

	if time.Now().Before(d.dnsInvalidate) && d.resolver != nil && d.ifaceAddr.IP.Equal(addr.IP) && bytes.Equal(d.ifaceAddr.Mask, addr.Mask) {
		return d.resolver
	}

	d.dnsInvalidate = time.Now().Add(DHCPTTL)

	dns, err := dhcp.ResolveDNSFromDHCP(ctx, d.ifaceName)
	if err != nil {
		return nil
	}

	nameserver := make([]NameServer, 0, len(dns))
	for _, d := range dns {
		nameserver = append(nameserver, NameServer{Addr: net.JoinHostPort(d.String(), "53")})
	}

	d.resolver = NewResolver(Config{
		Main: nameserver,
	})

	return d.resolver
}

func newDHCPClient(ifaceName string) *dhcpClient {
	return &dhcpClient{ifaceName: ifaceName}
}
