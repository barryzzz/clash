package rules

import (
	"strings"

	C "github.com/Dreamacro/clash/constant"
)

type DomainSuffix struct {
	suffix  string
	adapter string
}

func (ds *DomainSuffix) RuleType() C.RuleType {
	return C.DomainSuffix
}

func (ds *DomainSuffix) Match(metadata *C.Metadata) C.RuleMatchResult {
	domain := metadata.Host

	if metadata.AddrType != C.AtypDomainName || !(strings.HasSuffix(domain, "."+ds.suffix) || domain == ds.suffix) {
		return C.NotMatched
	}

	return C.DomainMatched
}

func (ds *DomainSuffix) Adapter() string {
	return ds.adapter
}

func (ds *DomainSuffix) Payload() string {
	return ds.suffix
}

func (ds *DomainSuffix) NoResolveIP() bool {
	return true
}

func NewDomainSuffix(suffix string, adapter string) *DomainSuffix {
	return &DomainSuffix{
		suffix:  strings.ToLower(suffix),
		adapter: adapter,
	}
}
