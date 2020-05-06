package constant

// Rule Type
const (
	Domain RuleType = iota
	DomainSuffix
	DomainKeyword
	GEOIP
	IPCIDR
	SrcIPCIDR
	SrcPort
	DstPort
	MATCH
)

// Rule Match Result
const (
	DomainMatched RuleMatchResult = iota
	IPMatched
	IPv4Matched
	IPv6Matched
	PortMatched
	Matched
	NotMatched
)

type RuleType int
type RuleMatchResult int

func (rt RuleType) String() string {
	switch rt {
	case Domain:
		return "Domain"
	case DomainSuffix:
		return "DomainSuffix"
	case DomainKeyword:
		return "DomainKeyword"
	case GEOIP:
		return "GeoIP"
	case IPCIDR:
		return "IPCIDR"
	case SrcIPCIDR:
		return "SrcIPCIDR"
	case SrcPort:
		return "SrcPort"
	case DstPort:
		return "DstPort"
	case MATCH:
		return "Match"
	default:
		return "Unknown"
	}
}

type Rule interface {
	RuleType() RuleType
	Match(metadata *Metadata) RuleMatchResult
	Adapter() string
	Payload() string
	NoResolveIP() bool
}
