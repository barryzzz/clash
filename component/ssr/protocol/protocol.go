package protocol

import (
	"errors"
	"net"
	"strings"
)

var (
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	ErrUnsupportedCipher   = errors.New("unsupported cipher")
	ErrNotImplement        = errors.New("not implement")
	ErrAuthFailure         = errors.New("auth failure")
)

type Protocol interface {
	StreamConn(net.Conn) (net.Conn, error)
	PacketConn(net.PacketConn) (net.PacketConn, error)
}

func NewProtocol(protocol string, passwordKey []byte, userId uint32, userKey string) (Protocol, error) {
	protocol = strings.ReplaceAll(strings.ToLower(protocol), "-", "_")

	switch protocol {
	case authAes128Md5:
		return NewAuthAes128Md5(passwordKey, userId, userKey), nil
	case authAes128Sha1:
		return NewAuthAes128Sha1(passwordKey, userId, userKey), nil
	}

	return nil, ErrUnsupportedProtocol
}
