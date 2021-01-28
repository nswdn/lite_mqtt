package calc

func Bytes2Bits(data byte) []uint8 {
	dst := make([]uint8, 0)
	for i := 0; i < 8; i++ {
		move := uint(7 - i)
		dst = append(dst, (data>>move)&1)
	}
	return dst
}
