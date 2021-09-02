package dialer

var (
	DefaultOptions []Option
)

type Config struct {
	SkipDefault   bool
	InterfaceName string
	AddrReuse     bool
}

type Option func(opt *Config)

func WithInterface(name string) Option {
	return func(opt *Config) {
		opt.InterfaceName = name
	}
}

func WithAddrReuse(reuse bool) Option {
	return func(opt *Config) {
		opt.AddrReuse = reuse
	}
}

func WithSkipDefault(skip bool) Option {
	return func(opt *Config) {
		opt.SkipDefault = skip
	}
}
