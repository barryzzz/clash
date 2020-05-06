package rules

import (
	"github.com/Dreamacro/clash/component/mmdb"
	C "github.com/Dreamacro/clash/constant"
)

type GEOIP struct {
	country     string
	adapter     string
	noResolveIP bool
}

func (g *GEOIP) RuleType() C.RuleType {
	return C.GEOIP
}

func (g *GEOIP) Match(metadata *C.Metadata) C.RuleMatchResult {
	ip := metadata.DstIP
	if ip == nil {
		return C.NotMatched
	}

	record4, _ := mmdb.Instance().Country(ip.V4)
	record6, _ := mmdb.Instance().Country(ip.V6)

	matched4 := record4.Country.IsoCode == g.country
	matched6 := record6.Country.IsoCode == g.country

	if matched4 && matched6 {
		return C.IPMatched
	} else if matched4 {
		return C.IPv4Matched
	} else if matched6 {
		return C.IPv6Matched
	} else {
		return C.NotMatched
	}
}

func (g *GEOIP) Adapter() string {
	return g.adapter
}

func (g *GEOIP) Payload() string {
	return g.country
}

func (g *GEOIP) NoResolveIP() bool {
	return g.noResolveIP
}

func NewGEOIP(country string, adapter string, noResolveIP bool) *GEOIP {
	geoip := &GEOIP{
		country:     country,
		adapter:     adapter,
		noResolveIP: noResolveIP,
	}

	return geoip
}
