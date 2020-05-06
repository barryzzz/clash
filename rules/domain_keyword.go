package rules

import (
	"strings"

	C "github.com/Dreamacro/clash/constant"
)

type DomainKeyword struct {
	keyword string
	adapter string
}

func (dk *DomainKeyword) RuleType() C.RuleType {
	return C.DomainKeyword
}

func (dk *DomainKeyword) Match(metadata *C.Metadata) C.RuleMatchResult {
	domain := metadata.Host

	if metadata.AddrType != C.AtypDomainName || !strings.Contains(domain, dk.keyword) {
		return C.NotMatched
	}

	return C.DomainMatched
}

func (dk *DomainKeyword) Adapter() string {
	return dk.adapter
}

func (dk *DomainKeyword) Payload() string {
	return dk.keyword
}

func (dk *DomainKeyword) NoResolveIP() bool {
	return true
}

func NewDomainKeyword(keyword string, adapter string) *DomainKeyword {
	return &DomainKeyword{
		keyword: strings.ToLower(keyword),
		adapter: adapter,
	}
}
