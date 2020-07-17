package ssr

import (
	"fmt"
	"reflect"

	"github.com/Dreamacro/clash/component/ssr/protocol"

	"github.com/Dreamacro/go-shadowsocks2/core"
)

type Option struct {
	Cipher   core.Cipher
	Protocol string
	Host     string
	Port     int
	UserID   uint32
	UserKey  string
}

type Plugin struct {
	Protocol protocol.Protocol
}

func New(option *Option) (plugin *Plugin, err error) {
	cipher, ok := option.Cipher.(*core.StreamCipher)
	if !ok {
		return nil, fmt.Errorf("unsupported cipher %s", reflect.TypeOf(option.Cipher).Name())
	}

	plugin = &Plugin{}

	if option.Protocol != "" {
		plugin.Protocol, err = protocol.NewProtocol(option.Protocol, cipher.Key, option.UserID, option.UserKey)
		if err != nil {
			return nil, err
		}
	}

	return
}
