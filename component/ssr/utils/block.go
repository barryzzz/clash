package utils

func ForEachBlock(bytes []byte, blockSize int, handler func([]byte) error) error {
	for len(bytes) > 0 {
		current := MinInt(len(bytes), blockSize)

		if err := handler(bytes[:current]); err != nil {
			return err
		}

		bytes = bytes[current:]
	}

	return nil
}
