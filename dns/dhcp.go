package dns

import (
	"bytes"
	"context"
	"net"
	"time"

	D "github.com/miekg/dns"

	"github.com/Dreamacro/clash/common/singledo"
	"github.com/Dreamacro/clash/component/dhcp"
	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/iface"
	"github.com/Dreamacro/clash/component/resolver"
)

const defaultCacheTTL = 60

type dhcpClient struct {
	*D.Client

	ifaceName string
	dnsAddr   *singledo.Single

	cacheDNS  net.IP
	cacheAddr *net.IPNet
	cacheTTL  int
}

func (d *dhcpClient) Exchange(m *D.Msg) (msg *D.Msg, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
	defer cancel()

	return d.ExchangeContext(ctx, m)
}

func (d *dhcpClient) ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	value, err, _ := d.dnsAddr.Do(func() (interface{}, error) {
		ifaceObj, err := iface.ResolveInterface(d.ifaceName)
		if err != nil {
			return nil, err
		}

		addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
		if err != nil {
			return nil, err
		}

		if d.cacheTTL > 0 && d.cacheAddr.IP.Equal(addr.IP) && bytes.Equal(d.cacheAddr.Mask, addr.Mask) {
			d.cacheTTL--

			return d.cacheDNS, nil
		}

		dns, err := dhcp.ResolveDNSFromDHCP(ctx, ifaceObj.Name)
		if err != nil {
			return nil, err
		}

		d.cacheTTL = defaultCacheTTL
		d.cacheDNS = dns
		d.cacheAddr = addr

		return dns, nil
	})
	if err != nil {
		return nil, err
	}

	dns := value.(net.IP)

	conn, err := dialer.DialContext(ctx, "udp", net.JoinHostPort(dns.String(), "53"), dialer.WithInterface(d.ifaceName))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	r := make(chan *D.Msg, 1)
	e := make(chan error, 1)

	go d.exchangeWithConn(conn, m, r, e)

	select {
	case r := <-r:
		return r, nil
	case e := <-e:
		return nil, e
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (d *dhcpClient) exchangeWithConn(conn net.Conn, m *D.Msg, r chan<- *D.Msg, e chan<- error) {
	defer conn.Close()

	m, _, err := d.Client.ExchangeWithConn(m, &D.Conn{
		Conn:         conn,
		UDPSize:      D.MinMsgSize,
		TsigSecret:   nil,
		TsigProvider: nil,
	})
	if err != nil {
		e <- err
		return
	}

	r <- m
}

func newDHCPClient(ifaceName string) *dhcpClient {
	return &dhcpClient{
		Client: &D.Client{
			Net:     "udp",
			UDPSize: D.MinMsgSize,
		},
		ifaceName: ifaceName,
		dnsAddr:   singledo.NewSingle(time.Second * 20),
	}
}
