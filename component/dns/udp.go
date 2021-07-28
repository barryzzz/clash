package libdns

import (
	"context"
	"io"
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

const LargeUDPDNSMessageSize = 1332

type UDPTransport struct {
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func (t *UDPTransport) RoundTrip(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error) {
	conn, err := dialWith(t.DialContext, ctx, "udp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resCh := make(chan *dnsmessage.Message, 1)
	errCh := make(chan error, 1)

	go func() {
		if err := writeMsg(conn, msg); err != nil {
			errCh <- err

			return
		}

		for {
			reply, err := readMsg(conn)
			if err != nil {
				errCh <- err

				return
			}

			if reply.ID != msg.ID {
				continue
			}

			resCh <- reply

			return
		}
	}()

	select {
	case res := <-resCh:
		return res, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func writeMsg(conn net.Conn, msg *dnsmessage.Message) error {
	data, err := msg.Pack()
	if err != nil {
		return err
	}

	n, err := conn.Write(data)
	if err != nil {
		return err
	}

	if n != len(data) {
		return io.ErrShortWrite
	}

	return nil
}

func readMsg(conn net.Conn) (*dnsmessage.Message, error) {
	data := make([]byte, LargeUDPDNSMessageSize)

	n, err := conn.Read(data)
	if err != nil {
		return nil, err
	}

	msg := &dnsmessage.Message{}

	return msg, msg.Unpack(data[:n])
}
