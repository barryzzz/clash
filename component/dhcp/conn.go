// +build !linux

package dhcp

import (
	"context"
	"net"

	"github.com/Dreamacro/clash/component/dialer"
)

const ListenAddr = "0.0.0.0:68"

func ListenDHCPClient(ctx context.Context, ifaceName string) (net.PacketConn, error) {
	return dialer.ListenPacket(ctx, "udp4", ListenAddr, dialer.WithInterface(ifaceName))
}
