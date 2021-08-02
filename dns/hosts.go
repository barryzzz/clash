package dns

import (
	"context"
	"net"

	DM "golang.org/x/net/dns/dnsmessage"

	D "github.com/Dreamacro/clash/component/dns"
	"github.com/Dreamacro/clash/component/resolver"
	"github.com/Dreamacro/clash/component/trie"
)

type hosts struct {
	hosts    *trie.DomainTrie
	fallback module
}

func (h *hosts) ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error) {
	if !D.IsIPRequest(msg) {
		return h.fallback.ExchangeContext(ctx, msg)
	}

	domain := D.TrimFqdn(msg.Questions[0].Name.String())
	if node := h.hosts.Search(domain); node != nil {
		ip := node.Data.(net.IP)

		var resourceType DM.Type
		var resourceBody DM.ResourceBody

		if v4 := ip.To4(); v4 != nil {
			resourceType = DM.TypeA
			resourceBody = &DM.AResource{}

			copy(resourceBody.(*DM.AResource).A[:], v4)
		} else {
			resourceType = DM.TypeAAAA
			resourceBody = &DM.AAAAResource{}

			copy(resourceBody.(*DM.AAAAResource).AAAA[:], ip.To16())
		}

		if resourceType != msg.Questions[0].Type {
			return nil, resolver.ErrIPVersion
		}

		return &DM.Message{
			Header: DM.Header{
				ID:                 msg.ID,
				Response:           true,
				RecursionAvailable: true,
				RCode:              DM.RCodeSuccess,
			},
			Questions: msg.Questions,
			Answers: []DM.Resource{{
				Header: DM.ResourceHeader{
					Name:  msg.Questions[0].Name,
					Type:  resourceType,
					Class: DM.ClassINET,
					TTL:   dnsDefaultTTL,
				},
				Body: resourceBody,
			}},
		}, nil
	}

	return h.fallback.ExchangeContext(ctx, msg)
}
