package libdns

import (
	"context"
	"encoding/binary"
	"io"
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

type TCPTransport struct {
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func (t *TCPTransport) RoundTrip(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error) {
	conn, err := dialWith(t.DialContext, ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	resCh := make(chan *dnsmessage.Message, 1)
	errCh := make(chan error, 1)

	go func() {
		if err := writeMsgWithLength(conn, msg); err != nil {
			errCh <- err

			return
		}

		for {
			reply, err := readMsgWithLength(conn)
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
	case r := <-resCh:
		return r, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func writeMsgWithLength(conn net.Conn, msg *dnsmessage.Message) error {
	data, err := msg.Pack()
	if err != nil {
		return err
	}

	length := [2]byte{}

	binary.BigEndian.PutUint16(length[:], uint16(len(data)))

	buffers := net.Buffers{length[:], data}

	_, err = buffers.WriteTo(conn)

	return err
}

func readMsgWithLength(conn net.Conn) (*dnsmessage.Message, error) {
	length := [2]byte{}

	_, err := io.ReadFull(conn, length[:])
	if err != nil {
		return nil, err
	}

	data := make([]byte, int(binary.BigEndian.Uint16(length[:])))

	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil, err
	}

	msg := &dnsmessage.Message{}

	return msg, msg.Unpack(data)
}
