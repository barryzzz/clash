package dns

import (
	"context"
	"net"

	DM "golang.org/x/net/dns/dnsmessage"

	D "github.com/Dreamacro/clash/component/dns"
)

type fallback struct {
	main     upstream
	fallback upstream
	filters  []filter
}

func (f *fallback) ExchangeContext(ctx context.Context, msg *DM.Message) (*DM.Message, error) {
	if !D.IsIPRequest(msg) {
		return f.main.ExchangeContext(ctx, msg)
	}

	question := &msg.Questions[0]
	if f.shouldUseFallback(question.Name.String(), nil) {
		return f.fallback.ExchangeContext(ctx, msg)
	}

	fallback := &struct {
		reply *DM.Message
		err   error
	}{}

	fallbackCh := make(chan struct{})

	go func() {
		fallback.reply, fallback.err = f.fallback.ExchangeContext(ctx, msg)
		close(fallbackCh)
	}()

	reply, err := f.main.ExchangeContext(ctx, msg)
	ips := D.ExtractIPsFromMessage(reply)
	if err != nil || len(ips) == 0 || f.shouldUseFallback("", ips[0]) {
		<-fallbackCh

		return fallback.reply, fallback.err
	}

	return reply, err
}

func (f *fallback) shouldUseFallback(domain string, ip net.IP) bool {
	for _, f := range f.filters {
		if f.Match(domain, ip) {
			return true
		}
	}

	return false
}
