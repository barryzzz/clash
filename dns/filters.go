package dns

import (
	"net"

	"github.com/Dreamacro/clash/component/mmdb"
	"github.com/Dreamacro/clash/component/trie"
)

type filter interface {
	Match(domain string, ip net.IP) bool
}

type geoipFilter struct{}

type ipcidrFilter struct {
	ipNets []*net.IPNet
}

type domainFilter struct {
	domains *trie.DomainTrie
}

func (gf *geoipFilter) Match(_ string, ip net.IP) bool {
	record, _ := mmdb.Instance().Country(ip)
	return record.Country.IsoCode != "CN" && record.Country.IsoCode != ""
}

func (inf *ipcidrFilter) Match(_ string, ip net.IP) bool {
	if ip == nil {
		return false
	}

	for _, n := range inf.ipNets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func (df *domainFilter) Match(domain string, _ net.IP) bool {
	if domain == "" {
		return false
	}

	return df.domains.Search(domain) != nil
}
