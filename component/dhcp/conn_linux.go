package dhcp

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"time"

	"github.com/Dreamacro/clash/component/dialer"
	IF "github.com/Dreamacro/clash/component/iface"
	"golang.org/x/net/ipv4"
)

const (
	udpHeaderLength   = 8
	udpProtocolNumber = 17
	dhcpClientPort    = 68
)

type udpPacket []byte

func (p udpPacket) SourcePort() uint16 {
	return binary.BigEndian.Uint16(p[0:2])
}

func (p udpPacket) SetSourcePort(port uint16) {
	binary.BigEndian.PutUint16(p[0:2], port)
}

func (p udpPacket) TargetPort() uint16 {
	return binary.BigEndian.Uint16(p[2:4])
}

func (p udpPacket) SetTargetPort(port uint16) {
	binary.BigEndian.PutUint16(p[2:4], port)
}

func (p udpPacket) Length() uint16 {
	return binary.BigEndian.Uint16(p[4:6])
}

func (p udpPacket) SetLength(length uint16) {
	binary.BigEndian.PutUint16(p[4:6], length)
}

func (p udpPacket) Checksum() uint16 {
	return binary.BigEndian.Uint16(p[6:8])
}

func (p udpPacket) SetChecksum(checksum uint16) {
	binary.BigEndian.PutUint16(p[6:8], checksum)
}

func (p udpPacket) Payload() []byte {
	return p[udpHeaderLength:p.Length()]
}

func (p udpPacket) Valid(sourceAddr, targetAddr net.IP) bool {
	if int(p.Length()) > len(p) {
		return false
	}

	b := p.Checksum()
	p.SetChecksum(0)
	defer p.SetChecksum(b)

	s := uint32(0)

	s += Sum(sourceAddr)
	s += Sum(targetAddr)
	s += uint32(udpProtocolNumber)
	s += uint32(p.Length())

	c := Checksum(s, p[:p.Length()])

	return c == b
}

func (p udpPacket) Verify(sourceAddr, targetAddr net.IP) {
	p.SetChecksum(0)

	s := uint32(0)

	s += Sum(sourceAddr)
	s += Sum(targetAddr)
	s += uint32(udpProtocolNumber)
	s += uint32(p.Length())

	c := Checksum(s, p[:p.Length()])

	p.SetChecksum(c)
}

type udpPacketConn struct {
	rawConn   *ipv4.RawConn
	localAddr *net.UDPAddr
}

func (u *udpPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buf := make([]byte, 65535)

	for {
		header, payload, _, err := u.rawConn.ReadFrom(buf)
		if err != nil {
			return 0, nil, err
		}

		if header.Protocol != udpProtocolNumber {
			continue
		}

		pkt := udpPacket(payload)
		if !pkt.Valid(header.Src.To4(), header.Dst.To4()) {
			continue
		}
		if int(pkt.TargetPort()) != u.localAddr.Port {
			continue
		}

		return copy(p, pkt.Payload()), &net.UDPAddr{IP: header.Src, Port: int(pkt.SourcePort())}, nil
	}
}

func (u *udpPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	uAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, errors.New("invalid addr")
	}

	buf := make([]byte, 65535)

	pkt := udpPacket(buf)

	pkt.SetLength(uint16(len(p)) + udpHeaderLength)
	pkt.SetSourcePort(uint16(u.localAddr.Port))
	pkt.SetTargetPort(uint16(uAddr.Port))

	n = copy(pkt.Payload(), p)

	pkt.Verify(u.localAddr.IP.To4(), uAddr.IP.To4())

	hdr := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0,
		TotalLen: ipv4.HeaderLen + int(pkt.Length()),
		ID:       rand.Int(),
		Flags:    0,
		FragOff:  0,
		TTL:      255,
		Protocol: udpProtocolNumber,
		Checksum: 0,
		Src:      u.localAddr.IP.To4(),
		Dst:      net.IPv4bcast,
		Options:  nil,
	}

	return n, u.rawConn.WriteTo(hdr, pkt[:pkt.Length()], nil)
}

func (u *udpPacketConn) Close() error {
	return u.rawConn.Close()
}

func (u *udpPacketConn) LocalAddr() net.Addr {
	return u.localAddr
}

func (u *udpPacketConn) SetDeadline(t time.Time) error {
	return u.rawConn.SetDeadline(t)
}

func (u *udpPacketConn) SetReadDeadline(t time.Time) error {
	return u.rawConn.SetReadDeadline(t)
}

func (u *udpPacketConn) SetWriteDeadline(t time.Time) error {
	return u.rawConn.SetWriteDeadline(t)
}

func ListenDHCPClient(ifaceName string) (net.PacketConn, error) {
	iface, err := IF.ResolveInterface(ifaceName)
	if err != nil {
		return nil, err
	}

	addr, err := IF.PickIPv4Addr(iface.Addrs)
	if err != nil {
		return nil, err
	}

	conn, err := dialer.ListenPacketWithHook(dialer.ListenPacketWithInterface(ifaceName), "ip4:udp", "")
	if err != nil {
		return nil, err
	}

	rawConn, err := ipv4.NewRawConn(conn)

	return &udpPacketConn{
		rawConn:   rawConn,
		localAddr: &net.UDPAddr{IP: addr.IP, Port: dhcpClientPort},
	}, nil
}

func Sum(b []byte) uint32 {
	var sum uint32

	n := len(b)
	for i := 0; i < n; i = i + 2 {
		sum += uint32(b[i]) << 8
		if i+1 < n {
			sum += uint32(b[i+1])
		}
	}
	return sum
}

func Checksum(sum uint32, b []byte) uint16 {
	sum += Sum(b)
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	sum = ^sum

	return uint16(sum)
}
