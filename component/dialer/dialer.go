package dialer

import (
	"context"
	"errors"
	"net"

	"github.com/Dreamacro/clash/component/resolver"
)

type Config struct {
	Dialer       *net.Dialer
	ListenConfig *net.ListenConfig
	Context      context.Context
	Network      string
	Address      string
	IP           net.IP
}

type Option func(opt *Config) error

func (d *Config) DialContext() (net.Conn, error) {
	return d.Dialer.DialContext(d.Context, d.Network, d.Address)
}

func (l *Config) ListenPacket() (net.PacketConn, error) {
	return l.ListenConfig.ListenPacket(l.Context, l.Network, l.Address)
}

func DialContext(ctx context.Context, network, address string, options ...Option) (net.Conn, error) {
	switch network {
	case "tcp4", "tcp6", "udp4", "udp6":
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		var ip net.IP
		switch network {
		case "tcp4", "udp4":
			ip, err = resolver.ResolveIPv4(host)
		default:
			ip, err = resolver.ResolveIPv6(host)
		}
		if err != nil {
			return nil, err
		}

		return dialContext(ctx, network, net.JoinHostPort(ip.String(), port), ip, options)
	case "tcp", "udp":
		return dualStackDialContext(ctx, network, address, options)
	default:
		return nil, errors.New("network invalid")
	}
}

func ListenPacket(ctx context.Context, network, address string, options ...Option) (net.PacketConn, error) {
	cfg := &Config{
		ListenConfig: &net.ListenConfig{},
		Context:      ctx,
		Network:      network,
		Address:      address,
	}

	for _, o := range DefaultOptions {
		if err := o(cfg); err != nil {
			return nil, err
		}
	}

	for _, o := range options {
		if err := o(cfg); err != nil {
			return nil, err
		}
	}

	return cfg.ListenPacket()
}

func dialContext(ctx context.Context, network, address string, ip net.IP, options []Option) (net.Conn, error) {
	opt := &Config{
		Dialer:  &net.Dialer{},
		Context: ctx,
		Network: network,
		Address: address,
		IP:      ip,
	}

	for _, o := range DefaultOptions {
		if err := o(opt); err != nil {
			return nil, err
		}
	}

	for _, o := range options {
		if err := o(opt); err != nil {
			return nil, err
		}
	}

	return opt.DialContext()
}

func dualStackDialContext(ctx context.Context, network, address string, options []Option) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		resolved bool
		ipv6     bool
		done     bool
	}
	results := make(chan dialResult)
	var primary, fallback dialResult

	startRacer := func(ctx context.Context, network, host string, ipv6 bool) {
		result := dialResult{ipv6: ipv6, done: true}
		defer func() {
			select {
			case results <- result:
			case <-returned:
				if result.Conn != nil {
					result.Conn.Close()
				}
			}
		}()

		var ip net.IP
		if ipv6 {
			ip, result.error = resolver.ResolveIPv6(host)
		} else {
			ip, result.error = resolver.ResolveIPv4(host)
		}
		if result.error != nil {
			return
		}
		result.resolved = true

		result.Conn, result.error = dialContext(ctx, network, net.JoinHostPort(ip.String(), port), ip, options)
	}

	go startRacer(ctx, network+"4", host, false)
	go startRacer(ctx, network+"6", host, true)

	for res := range results {
		if res.error == nil {
			return res.Conn, nil
		}

		if !res.ipv6 {
			primary = res
		} else {
			fallback = res
		}

		if primary.done && fallback.done {
			if primary.resolved {
				return nil, primary.error
			} else if fallback.resolved {
				return nil, fallback.error
			} else {
				return nil, primary.error
			}
		}
	}

	return nil, errors.New("never touched")
}
