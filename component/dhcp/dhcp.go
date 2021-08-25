package dhcp

import (
	"context"
	"errors"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"

	"github.com/Dreamacro/clash/component/iface"
)

var ErrNotResponding = errors.New("DHCP not responding")
var ErrNotFound = errors.New("DNS option not found")

func ResolveDNSFromDHCP(context context.Context, ifaceName string) ([]net.IP, error) {
	ifaceObj, err := iface.ResolveInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	conn, err := ListenDHCPClient(context, ifaceName)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	result := make(chan []net.IP, 1)

	addr, err := iface.PickIPv4Addr(ifaceObj.Addrs)
	if err != nil {
		return nil, err
	}

	inform, err := dhcpv4.NewInform(ifaceObj.HardwareAddr, addr.IP, dhcpv4.WithBroadcast(false), dhcpv4.WithRequestedOptions(dhcpv4.OptionDomainNameServer))
	if err != nil {
		return nil, err
	}

	go receiveAck(conn, inform.TransactionID, result)

	_, err = conn.WriteTo(inform.ToBytes(), &net.UDPAddr{IP: net.IPv4bcast, Port: 67})
	if err != nil {
		return nil, err
	}

	select {
	case r, ok := <-result:
		if !ok {
			return nil, ErrNotFound
		}
		return r, nil
	case <-context.Done():
		return nil, ErrNotResponding
	}
}

func receiveAck(conn net.PacketConn, id dhcpv4.TransactionID, result chan<- []net.IP) {
	defer close(result)

	buf := make([]byte, dhcpv4.MaxMessageSize)

	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}

		pkt, err := dhcpv4.FromBytes(buf[:n])
		if err != nil {
			continue
		}

		if pkt.MessageType() != dhcpv4.MessageTypeAck {
			continue
		}

		if pkt.TransactionID != id {
			continue
		}

		dns := pkt.DNS()
		if len(dns) == 0 {
			return
		}

		result <- dns

		return
	}
}
