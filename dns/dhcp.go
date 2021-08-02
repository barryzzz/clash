package dns

import (
	"bytes"
	"context"
	"net"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/Dreamacro/clash/common/singledo"
	DH "github.com/Dreamacro/clash/component/dhcp"
	"github.com/Dreamacro/clash/component/dialer"
	D "github.com/Dreamacro/clash/component/dns"
	IF "github.com/Dreamacro/clash/component/iface"
)

const dhcpDefaultTTL = 60

type dhcp struct {
	parallel

	ifaceName string
	singleDo  *singledo.Single

	statusTTL  int
	statusAddr *net.IPNet
}

func (d *dhcp) ExchangeContext(ctx context.Context, msg *dnsmessage.Message) (*dnsmessage.Message, error) {
	_, err, _ := d.singleDo.Do(func() (interface{}, error) {
		iface, err := IF.ResolveInterface(d.ifaceName)
		if err != nil {
			return nil, err
		}

		addr, err := IF.PickIPv4Addr(iface.Addrs)
		if err != nil {
			return nil, err
		}

		if d.statusTTL > 0 && d.statusAddr.IP.Equal(addr.IP) && bytes.Equal(d.statusAddr.Mask, addr.Mask) {
			d.statusTTL--

			return nil, nil
		}

		dns, err := DH.ResolveDNSFromDHCP(ctx, iface.Name)
		if err != nil {
			return nil, err
		}

		var modules []module

		for _, ip := range dns {
			modules = append(modules, &client{
				Client: &D.Client{
					Transport: &D.UDPTransport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return dialer.DialContextWithHook(nil, dialer.DialerWithInterface(d.ifaceName), ctx, network, address)
					}},
				},
				address: net.JoinHostPort(ip.String(), "53"),
			})
		}
		if len(modules) == 0 {
			return nil, DH.ErrNotFound
		}

		d.modules = modules
		d.statusAddr = addr
		d.statusTTL = dhcpDefaultTTL

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	return d.parallel.ExchangeContext(ctx, msg)
}
