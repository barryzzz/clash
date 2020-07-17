package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/common/structure"
	"github.com/Dreamacro/clash/component/dialer"
	obfs "github.com/Dreamacro/clash/component/simple-obfs"
	"github.com/Dreamacro/clash/component/socks5"
	"github.com/Dreamacro/clash/component/ssr"
	v2rayObfs "github.com/Dreamacro/clash/component/v2ray-plugin"
	C "github.com/Dreamacro/clash/constant"

	"github.com/Dreamacro/go-shadowsocks2/core"
)

type ShadowSocks struct {
	*Base
	cipher core.Cipher
	plugin interface{}
}

type ShadowSocksOption struct {
	Name       string                 `proxy:"name"`
	Server     string                 `proxy:"server"`
	Port       int                    `proxy:"port"`
	Password   string                 `proxy:"password"`
	Cipher     string                 `proxy:"cipher"`
	UDP        bool                   `proxy:"udp,omitempty"`
	Plugin     string                 `proxy:"plugin,omitempty"`
	PluginOpts map[string]interface{} `proxy:"plugin-opts,omitempty"`
}

type simpleObfsOption struct {
	Mode string `obfs:"mode"`
	Host string `obfs:"host,omitempty"`
}

type v2rayObfsOption struct {
	Mode           string            `obfs:"mode"`
	Host           string            `obfs:"host,omitempty"`
	Path           string            `obfs:"path,omitempty"`
	TLS            bool              `obfs:"tls,omitempty"`
	Headers        map[string]string `obfs:"headers,omitempty"`
	SkipCertVerify bool              `obfs:"skip-cert-verify,omitempty"`
	Mux            bool              `obfs:"mux,omitempty"`
}

type ssrOption struct {
	Protocol struct {
		Mode    string `obfs:"mode"`
		UserID  int    `obfs:"user-id"`
		UserKey string `obfs:"user-key"`
	} `obfs:"protocol"`
}

func (ss *ShadowSocks) StreamConn(c net.Conn, metadata *C.Metadata) (conn net.Conn, err error) {
	defer func() {
		if err != nil {
			return
		}
		if _, e := conn.Write(serializesSocksAddr(metadata)); e != nil {
			conn = nil
			err = e
		}
	}()

	switch opts := ss.plugin.(type) {
	case *simpleObfsOption:
		if opts.Mode == "tls" {
			return ss.cipher.StreamConn(obfs.NewTLSObfs(c, opts.Host)), nil
		} else if opts.Mode == "http" {
			_, port, _ := net.SplitHostPort(ss.addr)
			return ss.cipher.StreamConn(obfs.NewHTTPObfs(c, opts.Host, port)), nil
		}
	case *v2rayObfs.Option:
		if conn, err := v2rayObfs.NewV2rayObfs(c, opts); err != nil {
			return nil, err
		} else {
			return ss.cipher.StreamConn(conn), nil
		}
	case *ssr.Plugin:
		c = ss.cipher.StreamConn(c)

		if opts.Protocol != nil {
			return opts.Protocol.StreamConn(c)
		}

		return c, nil
	}

	return ss.cipher.StreamConn(c), nil
}

func (ss *ShadowSocks) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, error) {
	c, err := dialer.DialContext(ctx, "tcp", ss.addr)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", ss.addr, err)
	}
	tcpKeepAlive(c)

	c, err = ss.StreamConn(c, metadata)
	return NewConn(c, ss), err
}

func (ss *ShadowSocks) DialUDP(metadata *C.Metadata) (C.PacketConn, error) {
	pc, err := dialer.ListenPacket("udp", "")
	if err != nil {
		return nil, err
	}

	addr, err := resolveUDPAddr("udp", ss.addr)
	if err != nil {
		return nil, err
	}

	pc = ss.cipher.PacketConn(pc)
	return newPacketConn(&ssPacketConn{PacketConn: pc, rAddr: addr}, ss), nil
}

func (ss *ShadowSocks) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type": ss.Type().String(),
	})
}

func NewShadowSocks(option ShadowSocksOption) (*ShadowSocks, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	cipher := option.Cipher
	password := option.Password
	ciph, err := core.PickCipher(cipher, nil, password)
	if err != nil {
		return nil, fmt.Errorf("ss %s initialize error: %w", addr, err)
	}

	var plugin interface{}

	decoder := structure.NewDecoder(structure.Option{TagName: "obfs", WeaklyTypedInput: true})
	if option.Plugin == "obfs" {
		opts := simpleObfsOption{Host: "bing.com"}
		if err := decoder.Decode(option.PluginOpts, &opts); err != nil {
			return nil, fmt.Errorf("ss %s initialize obfs error: %w", addr, err)
		}

		if opts.Mode != "tls" && opts.Mode != "http" {
			return nil, fmt.Errorf("ss %s obfs mode error: %s", addr, opts.Mode)
		}

		plugin = &opts
	} else if option.Plugin == "v2ray-plugin" {
		opts := v2rayObfsOption{Host: "bing.com", Mux: true}
		if err := decoder.Decode(option.PluginOpts, &opts); err != nil {
			return nil, fmt.Errorf("ss %s initialize v2ray-plugin error: %w", addr, err)
		}

		if opts.Mode != "websocket" {
			return nil, fmt.Errorf("ss %s obfs mode error: %s", addr, opts.Mode)
		}

		v2rayOption := &v2rayObfs.Option{
			Host:    opts.Host,
			Path:    opts.Path,
			Headers: opts.Headers,
			Mux:     opts.Mux,
		}

		if opts.TLS {
			v2rayOption.TLS = true
			v2rayOption.SkipCertVerify = opts.SkipCertVerify
			v2rayOption.SessionCache = getClientSessionCache()
		}

		plugin = v2rayOption
	} else if option.Plugin == "ssr" {
		opts := ssrOption{}
		if err := decoder.Decode(option.PluginOpts, &opts); err != nil {
			return nil, fmt.Errorf("ss %s ssr error: %w", addr, err)
		}

		ssrOption := &ssr.Option{
			Cipher:   ciph,
			Protocol: opts.Protocol.Mode,
			Host:     option.Server,
			Port:     option.Port,
			UserID:   uint32(opts.Protocol.UserID),
			UserKey:  opts.Protocol.UserKey,
		}

		ssrPlugin, err := ssr.New(ssrOption)
		if err != nil {
			return nil, fmt.Errorf("ss %s ssr plugin error: %w", addr, err)
		}

		plugin = ssrPlugin
	}

	return &ShadowSocks{
		Base: &Base{
			name: option.Name,
			addr: addr,
			tp:   C.Shadowsocks,
			udp:  option.UDP,
		},
		cipher: ciph,
		plugin: plugin,
	}, nil
}

type ssPacketConn struct {
	net.PacketConn
	rAddr net.Addr
}

func (spc *ssPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	packet, err := socks5.EncodeUDPPacket(socks5.ParseAddrToSocksAddr(addr), b)
	if err != nil {
		return
	}
	return spc.PacketConn.WriteTo(packet[3:], spc.rAddr)
}

func (spc *ssPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, _, e := spc.PacketConn.ReadFrom(b)
	if e != nil {
		return 0, nil, e
	}

	addr := socks5.SplitAddr(b[:n])
	if addr == nil {
		return 0, nil, errors.New("parse addr error")
	}

	udpAddr := addr.UDPAddr()
	if udpAddr == nil {
		return 0, nil, errors.New("parse addr error")
	}

	copy(b, b[len(addr):])
	return n - len(addr), udpAddr, e
}
