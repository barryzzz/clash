package dhcp

import (
	"context"
	"errors"
	"net"

	IF "github.com/Dreamacro/clash/component/iface"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

var ErrNotResponding = errors.New("not responding")
var ErrNotFound = errors.New("not found")

func ResolveDNSFromDHCP(context context.Context, ifaceName string) (net.IP, error) {
	iface, err := IF.ResolveInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	conn, err := ListenDHCPClient(ifaceName)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	result := make(chan net.IP, 1)

	addr, err := IF.PickIPv4Addr(iface.Addrs)
	if err != nil {
		return nil, err
	}

	inform, err := dhcpv4.NewInform(iface.HardwareAddr, addr.IP, dhcpv4.WithBroadcast(false), dhcpv4.WithRequestedOptions(dhcpv4.OptionDomainNameServer))
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

func receiveAck(conn net.PacketConn, id dhcpv4.TransactionID, result chan<- net.IP) {
	defer close(result)

	buf := make([]byte, 65535)

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

		result <- dns[0]

		return
	}
}
