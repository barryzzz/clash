package libdns

import (
	"context"
	"net"
)

type dialFunc = func (ctx context.Context, network, address string) (net.Conn, error)

func dialWith(dial dialFunc, ctx context.Context, network, address string) (net.Conn, error) {
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}

	return dial(ctx, network, address)
}