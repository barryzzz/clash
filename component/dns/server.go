package dns

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/Dreamacro/clash/common/pool"
)

var ErrDuplicateWrite = errors.New("duplicate write")

type ResponseWriter interface {
	WriteMessage(msg *dnsmessage.Message) error
}

type Handler interface {
	ServeDNS(w ResponseWriter, msg *dnsmessage.Message)
}

type HandleFunc func(w ResponseWriter, msg *dnsmessage.Message)

type Server struct {
	Handler      Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type responseWriter struct {
	writeBack func(msg *dnsmessage.Message) error
	once      sync.Once
}

func (r *responseWriter) WriteMessage(msg *dnsmessage.Message) error {
	err := ErrDuplicateWrite

	r.once.Do(func() {
		err = r.writeBack(msg)
	})

	return err
}

func (h HandleFunc) ServeDNS(w ResponseWriter, msg *dnsmessage.Message) {
	h(w, msg)
}

func (s *Server) ServePacket(pc net.PacketConn) error {
	buf := make([]byte, LargeUDPDNSMessageSize)

	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return err
		}

		msg := &dnsmessage.Message{}

		if err = msg.Unpack(buf[:n]); err != nil {
			continue
		}

		writer := &responseWriter{writeBack: func(msg *dnsmessage.Message) error {
			buf, err := msg.Pack()
			if err != nil {
				return err
			}

			_, err = pc.WriteTo(buf, addr)
			return err
		}}

		go s.handleMessage(writer, msg)
	}
}

func (s *Server) ServeStream(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	buf := pool.Get(pool.RelayBufferSize)
	defer pool.Put(buf)

	for {
		err := conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		if err != nil {
			return
		}

		_, err = io.ReadFull(conn, buf[:2])
		if err != nil {
			return
		}

		length := int(binary.BigEndian.Uint16(buf[:2]))
		if length > cap(buf) {
			return
		}

		n, err := io.ReadFull(conn, buf[:length])
		if err != nil {
			return
		}

		msg := &dnsmessage.Message{}

		err = msg.Unpack(buf[:n])
		if err != nil {
			continue
		}

		conn.SetReadDeadline(time.Time{})

		writer := &responseWriter{writeBack: func(msg *dnsmessage.Message) error {
			pkt, err := msg.AppendPack(buf[:0])
			if err != nil {
				return err
			}

			conn.SetWriteDeadline(time.Now().Add(s.WriteTimeout))
			defer conn.SetWriteDeadline(time.Time{})

			_, err = conn.Write(pkt)
			return err
		}}

		s.handleMessage(writer, msg)
	}
}

func (s *Server) handleMessage(writer *responseWriter, msg *dnsmessage.Message) {
	defer func() {
		writer.once.Do(func() {
			msg := &dnsmessage.Message{
				Header: dnsmessage.Header{
					ID:                 msg.ID,
					Response:           true,
					RecursionAvailable: true,
					RCode:              dnsmessage.RCodeServerFailure,
				},
				Questions: msg.Questions,
			}

			writer.writeBack(msg)
		})
	}()

	handler := s.Handler
	if handler == nil {

		return
	}

	handler.ServeDNS(writer, msg)
}
