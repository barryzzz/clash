// +build !linux

package feature

func HasConnZeroCopy() bool {
	return false
}
