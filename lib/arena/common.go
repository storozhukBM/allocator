package arena

func calculateRequiredPadding(o Offset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}

func clearBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	buf[0] = 0
	for bufPart := 1; bufPart < len(buf); bufPart *= 2 {
		copy(buf[bufPart:], buf[:bufPart])
	}
}
