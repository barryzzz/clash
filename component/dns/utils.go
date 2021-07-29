package dns

import (
	"context"
	"net"
	"strings"

	"golang.org/x/net/dns/dnsmessage"
)

type dialFunc = func(ctx context.Context, network, address string) (net.Conn, error)

func ShallowCloneMessage(msg *dnsmessage.Message) *dnsmessage.Message {
	return &dnsmessage.Message{
		Header:      msg.Header,
		Questions:   append([]dnsmessage.Question(nil), msg.Questions...),
		Answers:     append([]dnsmessage.Resource(nil), msg.Answers...),
		Authorities: append([]dnsmessage.Resource(nil), msg.Authorities...),
		Additionals: append([]dnsmessage.Resource(nil), msg.Additionals...),
	}
}

func ExtractIPsFromMessage(msg *dnsmessage.Message) []net.IP {
	res := make([]net.IP, 0, len(msg.Answers))

	for _, ans := range msg.Answers {
		switch ans.Header.Type {
		case dnsmessage.TypeA:
			res = append(res, ans.Body.(*dnsmessage.AResource).A[:])
		case dnsmessage.TypeAAAA:
			res = append(res, ans.Body.(*dnsmessage.AAAAResource).AAAA[:])
		}
	}

	return res
}

func IsIPRequest(msg *dnsmessage.Message) bool {
	if msg.Response {
		return false
	}

	if len(msg.Questions) == 0 {
		return false
	}

	q := msg.Questions[0]

	return q.Class == dnsmessage.ClassINET && (q.Type == dnsmessage.TypeA || q.Type == dnsmessage.TypeAAAA)
}

func OverrideTTLOfMessage(msg *dnsmessage.Message, ttl uint32) {
	for _, ans := range msg.Answers {
		ans.Header.TTL = ttl
	}
	for _, ns := range msg.Authorities {
		ns.Header.TTL = ttl
	}
	for _, add := range msg.Additionals {
		add.Header.TTL = ttl
	}
}

func TrimFqdn(fqdm string) string {
	return strings.TrimRight(fqdm, ".")
}

func Fqdn(domain string) string {
	if len(domain) == 0 {
		return domain
	}

	if domain[len(domain)-1] == '.' {
		return domain
	}

	return domain + "."
}

func ReplyWithEmptyAnswer(msg *dnsmessage.Message) *dnsmessage.Message {
	return &dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:                 msg.ID,
			Response:           true,
			RecursionAvailable: true,
			RCode:              dnsmessage.RCodeSuccess,
		},
		Questions: msg.Questions,
	}
}

func dialWith(dial dialFunc, ctx context.Context, network, address string) (net.Conn, error) {
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}

	return dial(ctx, network, address)
}
