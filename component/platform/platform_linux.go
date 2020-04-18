package platform

import "syscall"

func init() {
	l := int64(0)

	_, err := syscall.Splice(-1, &l, -1, &l, 0, 0)

	IsZeroCopySupported = err != syscall.ENOSYS
}
