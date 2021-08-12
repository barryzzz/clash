package dns

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/Dreamacro/clash/common/singledo"
	D "github.com/Dreamacro/clash/component/dns"
)

var (
	// EnhancedModeMapping is a mapping for EnhancedMode enum
	EnhancedModeMapping = map[string]EnhancedMode{
		NORMAL.String():  NORMAL,
		FAKEIP.String():  FAKEIP,
		MAPPING.String(): MAPPING,
	}
)

const (
	NORMAL EnhancedMode = iota
	FAKEIP
	MAPPING
)

type EnhancedMode int

// UnmarshalYAML deserialize EnhancedMode with yaml
func (e *EnhancedMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	mode, exist := EnhancedModeMapping[tp]
	if !exist {
		return errors.New("invalid mode")
	}
	*e = mode
	return nil
}

// MarshalYAML serialize EnhancedMode with yaml
func (e EnhancedMode) MarshalYAML() (interface{}, error) {
	return e.String(), nil
}

// UnmarshalJSON deserialize EnhancedMode with json
func (e *EnhancedMode) UnmarshalJSON(data []byte) error {
	var tp string
	json.Unmarshal(data, &tp)
	mode, exist := EnhancedModeMapping[tp]
	if !exist {
		return errors.New("invalid mode")
	}
	*e = mode
	return nil
}

// MarshalJSON serialize EnhancedMode with json
func (e EnhancedMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e EnhancedMode) String() string {
	switch e {
	case NORMAL:
		return "normal"
	case FAKEIP:
		return "fake-ip"
	case MAPPING:
		return "redir-host"
	default:
		return "unknown"
	}
}

func transformClients(servers []NameServer, dial D.DialContextFunc) []module {
	modules := make([]module, 0, len(servers))

	for _, s := range servers {
		switch s.Net {
		case "udp":
			modules = append(modules, &client{
				Client:  &D.Client{Transport: &D.UDPTransport{DialContext: dial}},
				address: s.Addr,
			})
		case "tcp":
			modules = append(modules, &client{
				Client:  &D.Client{Transport: &D.TCPTransport{DialContext: dial}},
				address: s.Addr,
			})
		case "tls":
			host, _, _ := net.SplitHostPort(s.Addr)
			modules = append(modules, &client{
				Client: &D.Client{Transport: &D.TLSTransport{
					Config: &tls.Config{
						// alpn identifier, see https://tools.ietf.org/html/draft-hoffman-dprive-dns-tls-alpn-00#page-6
						NextProtos: []string{"dns"},
						ServerName: host,
					},
					DialContext: dial,
				}},
				address: s.Addr,
			})
		case "https":
			modules = append(modules, &client{
				Client: &D.Client{Transport: &D.HTTPTransport{Client: &http.Client{
					Transport: &http.Transport{
						ForceAttemptHTTP2: true,
						DialContext:       dial,
					},
				}}},
				address: s.Addr,
			})
		case "dhcp":
			modules = append(modules, &dhcp{
				ifaceName: s.Addr,
				singleDo:  singledo.NewSingle(time.Second * 20),
			})
		}
	}

	return modules
}
