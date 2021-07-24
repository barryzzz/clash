package dialer

var (
	DefaultOptions []Option
)

func WithInterface(name string) Option {
	return func(opt *Config) error {
		if opt.Dialer != nil {
			err := bindIfaceToDialer(opt.Dialer, name)
			if err == errPlatformNotSupport {
				err = fallbackBindToDialer(opt.Dialer, opt.Address, opt.IP, name)
			}

			return err
		}
		if opt.ListenConfig != nil {
			err := bindIfaceToListenConfig(opt.ListenConfig, name)
			if err == errPlatformNotSupport {
				opt.Address, err = fallbackBindToListenConfig(name)
			}

			return err
		}

		return nil
	}
}
