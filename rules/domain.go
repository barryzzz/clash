package rules

import (
	"strings"

	C "github.com/Dreamacro/clash/constant"
)

type Domain struct {
	domain  string
	adapter string
}

func (d *Domain) RuleType() C.RuleType {
	return C.Domain
}

func (d *Domain) Match(metadata *C.Metadata) C.RuleMatchResult {
	if metadata.AddrType != C.AtypDomainName || metadata.Host != d.domain {
		return C.NotMatched
	}

	return C.DomainMatched
}

func (d *Domain) Adapter() string {
	return d.adapter
}

func (d *Domain) Payload() string {
	return d.domain
}

func (d *Domain) NoResolveIP() bool {
	return true
}

func NewDomain(domain string, adapter string) *Domain {
	return &Domain{
		domain:  strings.ToLower(domain),
		adapter: adapter,
	}
}
