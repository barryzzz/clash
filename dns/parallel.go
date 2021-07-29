package dns

import (
	"context"
	"errors"
	"fmt"

	DM "golang.org/x/net/dns/dnsmessage"

	"github.com/Dreamacro/clash/common/picker"
	"github.com/Dreamacro/clash/component/resolver"
)

var ErrAllDNSRequestFailed = errors.New("all DNS requests failed")

type parallel struct {
	upstreams []upstream
}

func (p *parallel) ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error) {
	fast, ctx := picker.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
	for _, client := range p.upstreams {
		c := client
		fast.Go(func() (interface{}, error) {
			return c.ExchangeContext(ctx, msg)
		})
	}

	elm := fast.Wait()
	if elm == nil {
		err := ErrAllDNSRequestFailed
		if fErr := fast.Error(); fErr != nil {
			err = fmt.Errorf("%w, first error: %s", err, fErr.Error())
		}
		return nil, err
	}

	return elm.(*DM.Message), nil
}
