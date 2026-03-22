package nvvf

// indexOfByte finds the first occurrence of a byte in a slice.
// Used by GetGPUName() to find the null terminator in the GPU name buffer.
func indexOfByte(buf []byte, b byte) int {
	for i, v := range buf {
		if v == b {
			return i
		}
	}
	return -1
}
