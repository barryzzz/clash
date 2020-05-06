package outbound

import (
	"context"
	"net"

	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/resolver"
	C "github.com/Dreamacro/clash/constant"
)

type Direct struct {
	*Base
}

func (d *Direct) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, error) {
	dest := metadata.DstIP

	if !metadata.Resolved() {
		var err error
		dest, err = resolver.ResolveIP(metadata.Host)
		if err != nil {
			return nil, err
		}
	}

	c, err := dialer.DialContextResolved(ctx, "tcp", dest, metadata.DstPort)
	if err != nil {
		return nil, err
	}
	tcpKeepAlive(c)
	return NewConn(c, d), nil
}

func (d *Direct) DialUDP(metadata *C.Metadata) (C.PacketConn, error) {
	pc, err := dialer.ListenPacket("udp", "")
	if err != nil {
		return nil, err
	}
	return newPacketConn(&directPacketConn{pc}, d), nil
}

type directPacketConn struct {
	net.PacketConn
}

func (dp *directPacketConn) WriteWithMetadata(p []byte, metadata *C.Metadata) (n int, err error) {
	if !metadata.Resolved() {
		ip, err := resolver.ResolveIP(metadata.Host)
		if err != nil {
			return 0, err
		}
		metadata.DstIP = ip
	}
	return dp.WriteTo(p, metadata.UDPAddr())
}

func NewDirect() *Direct {
	return &Direct{
		Base: &Base{
			name: "DIRECT",
			tp:   C.Direct,
			udp:  true,
		},
	}
}
