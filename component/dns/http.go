package libdns

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"golang.org/x/net/dns/dnsmessage"
)

type HTTPTransport struct {
	Client *http.Client
}

func (t *HTTPTransport) RoundTrip(ctx context.Context, msg *dnsmessage.Message, address string) (*dnsmessage.Message, error) {
	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}

	body, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	reply := &dnsmessage.Message{}

	return reply, reply.Unpack(data)
}
