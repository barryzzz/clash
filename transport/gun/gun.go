// Modified from: https://github.com/Qv2ray/gun-lite
// License: MIT

package gun

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/http2"
)

var (
	ErrInvalidLength = errors.New("invalid length")

	ErrClientOnly = errors.New("stream client only")
)

var (
	defaultHeader = http.Header{
		"content-type": []string{"application/grpc"},
		"user-agent":   []string{"grpc-go/1.36.0"},
	}
)

type Config struct {
	ServiceName string
	Host        string
}

type Gun struct {
	Config
	transport *http2.Transport
}

type Trunk struct {
	conn   net.Conn
	client *http2.ClientConn
	gun    *Gun
}

type conn struct {
	trunk       *Trunk
	lAddr       net.Addr
	rAddr       net.Addr
	writer      *bufio.Writer
	reader      *bufio.Reader
	readCloser  io.Closer
	writeCloser io.Closer
	remain      int
}

func (g *Gun) NewTrunk(conn net.Conn) (*Trunk, error) {
	cc := tls.Client(conn, g.transport.TLSClientConfig)
	if err := cc.Handshake(); err != nil {
		return nil, err
	}

	if state := cc.ConnectionState(); state.NegotiatedProtocol != http2.NextProtoTLS {
		return nil, fmt.Errorf("http2: unexpected ALPN protocol %s, want %s", state.NegotiatedProtocol, http2.NextProtoTLS)
	}

	conn = cc

	client, err := g.transport.NewClientConn(cc)
	if err != nil {
		return nil, err
	}

	return &Trunk{
		conn:   conn,
		client: client,
		gun:    g,
	}, nil
}

func (t *Trunk) NewConn(ctx context.Context) (net.Conn, error) {
	serviceName := "GunService"
	if t.gun.ServiceName != "" {
		serviceName = t.gun.ServiceName
	}

	reader, writer := io.Pipe()
	request := &http.Request{
		Method: http.MethodPost,
		Body:   reader,
		URL: &url.URL{
			Scheme: "https",
			Host:   t.gun.Host,
			Path:   fmt.Sprintf("/%s/Tun", serviceName),
		},
		Proto:      "HTTP/2",
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     defaultHeader,
	}
	request = request.WithContext(ctx)

	resp, err := t.client.RoundTrip(request)
	if err != nil {
		t.client.Close()
		t.conn.Close()

		return nil, err
	}

	return &conn{
		trunk:       t,
		lAddr:       t.conn.LocalAddr(),
		rAddr:       t.conn.RemoteAddr(),
		writer:      bufio.NewWriter(writer),
		reader:      bufio.NewReader(resp.Body),
		readCloser:  resp.Body,
		writeCloser: writer,
	}, nil
}

func (t *Trunk) Close() error {
	t.client.Close()
	return t.conn.Close()
}

func (c *conn) Read(b []byte) (n int, err error) {
	if c.remain > 0 {
		size := c.remain
		if len(b) < size {
			size = len(b)
		}

		n, err = io.ReadFull(c.reader, b[:size])
		c.remain -= n
		return
	}

	// 0x00 grpclength(uint32) 0x0A uleb128 payload
	_, err = c.reader.Discard(6)
	if err != nil {
		return 0, err
	}

	protobufPayloadLen, err := binary.ReadUvarint(c.reader)
	if err != nil {
		return 0, ErrInvalidLength
	}

	size := int(protobufPayloadLen)
	if len(b) < size {
		size = len(b)
	}

	n, err = io.ReadFull(c.reader, b[:size])
	if err != nil {
		return
	}

	remain := int(protobufPayloadLen) - n
	if remain > 0 {
		c.remain = remain
	}

	return n, nil
}

func (c *conn) Write(b []byte) (int, error) {
	protobufHeader := [binary.MaxVarintLen64 + 1]byte{0x0A}
	variantSize := binary.PutUvarint(protobufHeader[1:], uint64(len(b)))
	grpcHeader := make([]byte, 5)
	grpcPayloadLen := uint32(variantSize + 1 + len(b))
	binary.BigEndian.PutUint32(grpcHeader[1:5], grpcPayloadLen)

	buffers := net.Buffers{grpcHeader, protobufHeader[:variantSize+1], b}
	n, err := buffers.WriteTo(c.writer)
	if err != nil {
		return 0, err
	}

	return int(n), c.writer.Flush()
}

func (c *conn) Close() error {
	c.readCloser.Close()
	c.writeCloser.Close()

	return nil
}

func (c *conn) LocalAddr() net.Addr {
	return c.lAddr
}

func (c *conn) RemoteAddr() net.Addr {
	return c.rAddr
}

func (c *conn) SetDeadline(t time.Time) error {
	return c.Close()
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return c.Close()
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return c.Close()
}

func New(tlsCfg *tls.Config, cfg Config) *Gun {
	return &Gun{
		Config: cfg,
		transport: &http2.Transport{
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return nil, ErrClientOnly
			},
			TLSClientConfig: tlsCfg,
		},
	}
}
