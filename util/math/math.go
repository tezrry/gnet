package math

// IsPowerOfTwo reports whether given integer is a power of two.
func IsPowerOfTwo(n uint64) bool {
	return n&(n-1) == 0
}

// CeilToPowerOfTwo returns the least power of two integer value greater than
// or equal to n.
func CeilToPowerOfTwo(n uint64) uint64 {
	if n <= 2 {
		return n
	}
	n--
	n = formatBits(n)
	n++
	return n
}

// FloorToPowerOfTwo returns the greatest power of two integer value less than
// or equal to n.
func FloorToPowerOfTwo(n uint64) uint64 {
	if n <= 2 {
		return n
	}
	n = formatBits(n)
	n >>= 1
	n++
	return n
}

func formatBits(n uint64) uint64 {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n
}
