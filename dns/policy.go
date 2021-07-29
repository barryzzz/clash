package dns

import (
	"context"

	DM "golang.org/x/net/dns/dnsmessage"

	D "github.com/Dreamacro/clash/component/dns"
	"github.com/Dreamacro/clash/component/trie"
)

type policy struct {
	policies *trie.DomainTrie
	fallback upstream
}

func (p *policy) ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error) {
	if len(msg.Questions) == 0 {
		return p.fallback.ExchangeContext(ctx, msg)
	}

	if n := p.policies.Search(D.TrimFqdn(msg.Questions[0].Name.String())); n != nil {
		return n.Data.(upstream).ExchangeContext(ctx, msg)
	}

	return p.fallback.ExchangeContext(ctx, msg)
}
