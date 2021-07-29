package dns

import (
	"net"
	"time"

	DM "golang.org/x/net/dns/dnsmessage"

	"github.com/Dreamacro/clash/common/cache"
	"github.com/Dreamacro/clash/component/dns"
	"github.com/Dreamacro/clash/component/fakeip"
	"github.com/Dreamacro/clash/context"
	"github.com/Dreamacro/clash/log"
)

type handler func(ctx *context.DNSContext, msg *DM.Message) (*DM.Message, error)
type middleware func(next handler) handler

func withMapping(mapping *cache.LruCache) middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, msg *DM.Message) (*DM.Message, error) {
			if !dns.IsIPRequest(msg) {
				return next(ctx, msg)
			}

			q := msg.Questions[0]

			reply, err := next(ctx, msg)
			if err != nil {
				return nil, err
			}

			host := dns.TrimFqdn(q.Name.String())

			for _, ans := range reply.Answers {
				var ip net.IP

				switch a := ans.Body.(type) {
				case *DM.AResource:
					ip = a.A[:]
				case *DM.AAAAResource:
					ip = a.AAAA[:]
				default:
					continue
				}

				mapping.SetWithExpire(ip.String(), host, time.Now().Add(time.Second*time.Duration(ans.Header.TTL)))
			}

			return reply, nil
		}
	}
}

func withFakeIP(fakePool *fakeip.Pool) middleware {
	return func(next handler) handler {
		return func(ctx *context.DNSContext, msg *DM.Message) (*DM.Message, error) {
			if !dns.IsIPRequest(msg) {
				return next(ctx, msg)
			}

			q := msg.Questions[0]

			host := dns.TrimFqdn(q.Name.String())
			if fakePool.LookupHost(host) {
				return next(ctx, msg)
			}

			switch q.Type {
			case DM.TypeAAAA, dns.TypeSVCB, dns.TypeHTTPS:
				return dns.ReplyWithEmptyAnswer(msg), nil
			}

			if q.Type != DM.TypeA {
				return next(ctx, msg)
			}

			ctx.SetType(context.DNSTypeFakeIP)

			ip := fakePool.Lookup(host)
			resourceBody := &DM.AResource{}
			copy(resourceBody.A[:], ip.To4())

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
						Name:  q.Name,
						Type:  DM.TypeA,
						Class: DM.ClassINET,
						TTL:   1,
					},
					Body: resourceBody,
				}},
			}, nil
		}
	}
}

func withResolver(resolver *Resolver) handler {
	return func(ctx *context.DNSContext, msg *DM.Message) (*DM.Message, error) {
		ctx.SetType(context.DNSTypeRaw)

		reply, err := resolver.Exchange(msg)
		if err != nil {
			if len(msg.Questions) != 0 {
				log.Debugln("[DNS Server] Exchange %s failed: %v", msg.Questions[0].GoString(), err)
			}

			return reply, err
		}
		msg.RecursionAvailable = true

		return reply, nil
	}
}

func compose(middlewares []middleware, endpoint handler) handler {
	length := len(middlewares)
	h := endpoint
	for i := length - 1; i >= 0; i-- {
		middleware := middlewares[i]
		h = middleware(h)
	}

	return h
}

func newHandler(resolver *Resolver, mapper *ResolverEnhancer) handler {
	middlewares := []middleware{}

	if mapper.mode == FAKEIP {
		middlewares = append(middlewares, withFakeIP(mapper.fakePool))
	}

	if mapper.mode != NORMAL {
		middlewares = append(middlewares, withMapping(mapper.mapping))
	}

	return compose(middlewares, withResolver(resolver))
}
