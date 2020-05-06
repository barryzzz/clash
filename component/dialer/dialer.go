package dialer

import (
	"context"
	"errors"
	"net"

	"github.com/Dreamacro/clash/component/resolver"
)

func Dialer() (*net.Dialer, error) {
	dialer := &net.Dialer{}
	if DialerHook != nil {
		if err := DialerHook(dialer); err != nil {
			return nil, err
		}
	}

	return dialer, nil
}

func ListenConfig() (*net.ListenConfig, error) {
	cfg := &net.ListenConfig{}
	if ListenConfigHook != nil {
		if err := ListenConfigHook(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func Dial(ctx context.Context, network, address string) (net.Conn, error) {
	return DialContext(context.Background(), network, address)
}

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	dest, err := resolver.ResolveIP(host)
	if err != nil {
		return nil, err
	}

	return DialContextResolved(ctx, network, dest, port)
}

func DialContextResolved(ctx context.Context, network string, dest *resolver.ResolvedIP, port string) (net.Conn, error) {
	destination := *dest

	switch network {
	case "tcp4", "udp4":
		destination.V6 = nil
	case "tcp6", "udp6":
		destination.V4 = nil
	case "tcp", "udp":
		break
	default:
		return nil, errors.New("network invalid")
	}

	return dualStackDailContext(ctx, network, &destination, port)
}

func ListenPacket(network, address string) (net.PacketConn, error) {
	lc, err := ListenConfig()
	if err != nil {
		return nil, err
	}

	if ListenPacketHook != nil && address == "" {
		ip, err := ListenPacketHook()
		if err != nil {
			return nil, err
		}
		address = net.JoinHostPort(ip.String(), "0")
	}
	return lc.ListenPacket(context.Background(), network, address)
}

func dualStackDailContext(ctx context.Context, network string, dest *resolver.ResolvedIP, port string) (net.Conn, error) {
	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		ipv6 bool
		done bool
	}
	results := make(chan dialResult)
	var primary, fallback dialResult

	startRacer := func(ctx context.Context, network string, ip net.IP, ipv6 bool) {
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

		dialer, err := Dialer()
		if err != nil {
			result.error = err
			return
		}

		if DialHook != nil {
			if result.error = DialHook(dialer, network, ip); result.error != nil {
				return
			}
		}
		result.Conn, result.error = dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}

	if dest.IPv4Available() {
		go startRacer(ctx, network, dest.V4, false)
	}
	if dest.IPv6Available() {
		go startRacer(ctx, network, dest.V6, true)
	}

	for {
		select {
		case res := <-results:
			if res.error == nil {
				return res.Conn, nil
			}

			if !res.ipv6 {
				primary = res
			} else {
				fallback = res
			}

			if primary.done && fallback.done {
				if primary.done {
					return nil, primary.error
				} else if fallback.done {
					return nil, fallback.error
				} else {
					return nil, primary.error
				}
			}
		}
	}
}
