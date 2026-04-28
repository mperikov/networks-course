package checksum

func BytesToChecksum(data []byte) uint16 {
	var sum uint16 = 0
	for i, b := range data {
		if i&1 == 0 {
			sum += uint16(b) << 8
		} else {
			sum += uint16(b)
		}
	}
	return ^sum
}

func CheckSum(data []byte, checksum uint16) bool {
	for i, b := range data {
		if i&1 == 0 {
			checksum += uint16(b) << 8
		} else {
			checksum += uint16(b)
		}
	}
	return checksum == 0xffff
}
