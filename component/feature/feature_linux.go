package feature

import "syscall"

var zeroCopySupport = false

func init() {
	_, err := syscall.Splice(-1, nil, 0, nil, 0, 0)

	zeroCopySupport = err != syscall.ENOSYS
}

func HasConnZeroCopy() bool {
	return zeroCopySupport
}
