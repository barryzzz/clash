package context

import (
	"github.com/gofrs/uuid"
	"golang.org/x/net/dns/dnsmessage"
)

const (
	DNSTypeHost   = "host"
	DNSTypeFakeIP = "fakeip"
	DNSTypeRaw    = "raw"
)

type DNSContext struct {
	id  uuid.UUID
	msg *dnsmessage.Message
	tp  string
}

func NewDNSContext(msg *dnsmessage.Message) *DNSContext {
	id, _ := uuid.NewV4()
	return &DNSContext{
		id:  id,
		msg: msg,
	}
}

// ID implement C.PlainContext ID
func (c *DNSContext) ID() uuid.UUID {
	return c.id
}

// SetType set type of response
func (c *DNSContext) SetType(tp string) {
	c.tp = tp
}

// Type return type of response
func (c *DNSContext) Type() string {
	return c.tp
}
