package dns

import (
	"context"
	"crypto/tls"
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

type TLSTransport struct {
	Config *tls.Config

	DialContext DialContextFunc
}

func (t *TLSTransport) RoundTrip(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error) {
	tcp := &TCPTransport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := dialWith(t.DialContext, ctx, network, address)
			if err != nil {
				return nil, err
			}

			tc := tls.Client(conn, t.Config)
			errCh := make(chan error, 1)

			go func() {
				errCh <- tc.Handshake()
			}()

			select {
			case err := <-errCh:
				if err != nil {
					tc.Close()

					return nil, err
				}

				return tc, nil
			case <-ctx.Done():
				tc.Close()

				return nil, ctx.Err()
			}
		},
	}

	return tcp.RoundTrip(ctx, msg, address)
}
