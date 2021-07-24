// +build !linux

package dhcp

import (
	"net"

	"github.com/Dreamacro/clash/component/dialer"
)

const ListenAddr = "0.0.0.0:68"

func ListenDHCPClient(ifaceName string) (net.PacketConn, error) {
	return dialer.ListenPacketWithHook(dialer.ListenPacketWithInterface(ifaceName), "udp4", ListenAddr)
}
