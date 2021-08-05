package dns

import (
	"net"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/Dreamacro/clash/common/sockopt"
	D "github.com/Dreamacro/clash/component/dns"
	R "github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/component/trie"
	"github.com/Dreamacro/clash/context"
	"github.com/Dreamacro/clash/log"
)

var (
	address string
	server  = &Server{}

	dnsDefaultTTL uint32 = 600
)

type Server struct {
	packetConn net.PacketConn
	listener   net.Listener
	handler    handler
}

// ServeDNS implement D.Handler ServeDNS
func (s *Server) ServeDNS(w D.ResponseWriter, msg *dnsmessage.Message) {
	if len(msg.Questions) == 0 {
		return
	}

	handler := s.handler
	if handler == nil {
		return
	}

	reply, err := handler(context.NewDNSContext(msg), msg)
	if err != nil {
		reply = &dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:                 msg.ID,
				Response:           true,
				RecursionAvailable: true,
				RCode:              dnsmessage.RCodeRefused,
			},
			Questions: msg.Questions,
		}
	}

	w.WriteMessage(reply)
}

// Close implement io.Closer
func (s *Server) Close() error {
	if s.packetConn != nil {
		s.packetConn.Close()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	return nil
}

func (s *Server) setHandler(handler handler) {
	s.handler = handler
}

func ReCreateServer(addr string, resolver *Resolver, mapper *ResolverEnhancer, hosts *trie.DomainTrie) error {
	if addr == address && resolver != nil {
		handler := newHandler(resolver, mapper, hosts)
		server.setHandler(handler)
		return nil
	}

	if server != nil {
		server.Close()
		server = &Server{}
		address = ""
	}

	_, port, err := net.SplitHostPort(addr)
	if port == "0" || port == "" || err != nil {
		return nil
	}

	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	tl, err := net.Listen("tcp", addr)
	if err != nil {
		log.Warnln("Failed to Listen TCP Address: %s", err)
	}

	err = sockopt.UDPReuseaddr(pc.(*net.UDPConn))
	if err != nil {
		log.Warnln("Failed to Reuse UDP Address: %s", err)
	}

	address = addr
	handler := newHandler(resolver, mapper, hosts)
	server = &Server{
		packetConn: pc,
		listener:   tl,
		handler:    handler,
	}
	dServer := &D.Server{
		Handler:      server,
		ReadTimeout:  R.DefaultDNSTimeout,
		WriteTimeout: R.DefaultDNSTimeout,
	}

	go func() {
		dServer.ServePacket(pc)
	}()

	if tl != nil {
		go func() {
			dServer.ServeStream(tl)
		}()
	}

	return nil
}
