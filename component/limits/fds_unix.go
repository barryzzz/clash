// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package limits

import "golang.org/x/sys/unix"

var FdLimits = DefaultFdsLimit

func init() {
	limits := &unix.Rlimit{}

	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, limits); err != nil {
		return
	}

	if limits.Cur == unix.RLIM_INFINITY {
		FdLimits = InfiniteFdsLimit
	} else {
		FdLimits = int(limits.Cur)
	}

	if limits.Cur == limits.Max {
		return
	}

	limits.Cur = limits.Max
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, limits); err != nil {
		return
	}

	if limits.Cur == unix.RLIM_INFINITY {
		FdLimits = InfiniteFdsLimit
	} else {
		FdLimits = int(limits.Cur)
	}
}
