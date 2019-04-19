package arena

func calculateRequiredPadding(o Offset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}
