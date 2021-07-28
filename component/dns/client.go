package libdns

import (
	"context"

	"golang.org/x/net/dns/dnsmessage"
)

type RoundTripper interface {
	RoundTrip(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error)
}

type Client struct {
	Transport RoundTripper

	DisableTruncatedRetry bool
}

func (c *Client) ExchangeContext(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error) {
	tp := c.Transport
	if tp == nil {
		tp = &UDPTransport{}
	}

	reply, err := tp.RoundTrip(ctx, msg, address)
	if err != nil {
		return nil, err
	}

	if reply.Truncated && !c.DisableTruncatedRetry {
		if u, ok := tp.(*UDPTransport); ok {
			tp = &TCPTransport{DialContext: u.DialContext}

			return tp.RoundTrip(ctx, msg, address)
		}
	}

	return reply, nil
}
