package utils

import (
	"io"

	"github.com/Dreamacro/clash/common/pool"
)

func TrimReader(reader io.Reader, n int) error {
	buf := pool.Get(n)
	defer pool.Put(buf)

	_, err := io.ReadFull(reader, buf)
	return err
}
