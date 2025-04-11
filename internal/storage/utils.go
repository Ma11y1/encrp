package storage

func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
