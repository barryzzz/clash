package rules

import (
	"github.com/Dreamacro/clash/component/resolver"
	"net"

	C "github.com/Dreamacro/clash/constant"
)

type IPCIDROption func(*IPCIDR)

func WithIPCIDRSourceIP(b bool) IPCIDROption {
	return func(i *IPCIDR) {
		i.isSourceIP = b
	}
}

func WithIPCIDRNoResolve(noResolve bool) IPCIDROption {
	return func(i *IPCIDR) {
		i.noResolveIP = noResolve
	}
}

type IPCIDR struct {
	ipnet       *net.IPNet
	adapter     string
	isSourceIP  bool
	noResolveIP bool
}

func (i *IPCIDR) RuleType() C.RuleType {
	if i.isSourceIP {
		return C.SrcIPCIDR
	}
	return C.IPCIDR
}

func (i *IPCIDR) Match(metadata *C.Metadata) C.RuleMatchResult {
	ip := metadata.DstIP
	if i.isSourceIP {
		ip = resolver.ResolvedIPFromSingle(metadata.SrcIP)
	}

	if ip.IPv4Available() && i.ipnet.Contains(ip.V4) {
		return C.IPv4Matched
	} else if ip.IPv6Available() && i.ipnet.Contains(ip.V6) {
		return C.IPv6Matched
	} else {
		return C.NotMatched
	}
}

func (i *IPCIDR) Adapter() string {
	return i.adapter
}

func (i *IPCIDR) Payload() string {
	return i.ipnet.String()
}

func (i *IPCIDR) NoResolveIP() bool {
	return i.noResolveIP
}

func NewIPCIDR(s string, adapter string, opts ...IPCIDROption) (*IPCIDR, error) {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, errPayload
	}

	ipcidr := &IPCIDR{
		ipnet:   ipnet,
		adapter: adapter,
	}

	for _, o := range opts {
		o(ipcidr)
	}

	return ipcidr, nil
}
