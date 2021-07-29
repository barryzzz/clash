package dns

import (
	"context"
	"errors"

	DM "golang.org/x/net/dns/dnsmessage"

	D "github.com/Dreamacro/clash/component/dns"
)

var (
	ErrServerFailure = errors.New("server failure")
)

type client struct {
	*D.Client

	address string
}

func (c *client) ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error) {
	reply, err := c.Client.ExchangeContext(ctx, msg, c.address)
	if err != nil {
		return nil, err
	}

	if reply.RCode == DM.RCodeServerFailure || reply.RCode == DM.RCodeRefused {
		return nil, ErrServerFailure
	}

	return reply, nil
}
