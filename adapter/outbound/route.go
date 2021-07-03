package outbound

import (
	"net"

	C "github.com/Dreamacro/clash/constant"
)

type RouteAddr struct {
	net.Addr
	C.Route
	name string
}

type routeConn struct {
	net.Conn
	name string
}

type routePacketConn struct {
	net.PacketConn
	stream net.Conn
	name   string
}

func (c *routePacketConn) RemoteAddr() net.Addr {
	panic("implement me")
}

func (addr *RouteAddr) Hops() C.Hops {
	if addr.Route == nil {
		return append(make(C.Hops, 0, 32), addr.name)
	}

	return append(addr.Route.Hops(), addr.name)
}

func (c *routeConn) LocalAddr() net.Addr {
	parent, _ := c.Conn.LocalAddr().(*RouteAddr)

	return &RouteAddr{
		Addr:  c.Conn.LocalAddr(),
		Route: parent,
		name:  c.name,
	}
}

func (c *routePacketConn) LocalAddr() net.Addr {
	parent, _ := c.PacketConn.LocalAddr().(*RouteAddr)

	return &RouteAddr{
		Addr:  c.PacketConn.LocalAddr(),
		Route: parent,
		name:  c.name,
	}
}

func (c *routePacketConn) Read(b []byte) (n int, err error) {
	return c.stream.Read(b)
}

func (c *routePacketConn) Write(b []byte) (n int, err error) {
	return c.stream.Write(b)
}

func WithRouteHop(c net.Conn, p C.ProxyAdapter) net.Conn {
	if pc, ok := c.(net.PacketConn); ok {
		return &routePacketConn{
			PacketConn: pc,
			stream:     c,
			name:       p.Name(),
		}
	}

	return &routeConn{
		Conn: c,
		name: p.Name(),
	}
}
