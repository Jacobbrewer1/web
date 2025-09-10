package vault

import "fmt"

const (
	// maxInt is the maximum value of an int on the current platform.
	maxInt = int(^uint(0) >> 1)
)

// uintToInt converts a uint to an int, returning an error if the conversion would overflow.
func uintToInt(u uint) (int, error) {
	if u > uint(maxInt) {
		return 0, fmt.Errorf("uint value %d overflows int", u)
	}
	return int(u), nil
}
