package inbound

import (
	"net"

	"github.com/Dreamacro/clash/component/socks5"
	C "github.com/Dreamacro/clash/constant"
)

// SocketAdapter is a adapter for socks and redir connection
type SocketAdapter struct {
	conn     net.Conn
	metadata *C.Metadata
}

// Conn return net.Conn of connection
func (s *SocketAdapter) Conn() net.Conn {
	return s.conn
}

// Metadata return destination metadata
func (s *SocketAdapter) Metadata() *C.Metadata {
	return s.metadata
}

// Close close underlying resources
func (s *SocketAdapter) Close() {
	_ = s.conn.Close()
}

// NewSocket is SocketAdapter generator
func NewSocket(target socks5.Addr, conn net.Conn, source C.Type) *SocketAdapter {
	metadata := parseSocksAddr(target)
	metadata.NetWork = C.TCP
	metadata.Type = source
	if ip, port, err := parseAddr(conn.RemoteAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}

	return &SocketAdapter{
		conn:     conn,
		metadata: metadata,
	}
}
