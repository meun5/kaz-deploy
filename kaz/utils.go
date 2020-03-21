package kaz

func ChunkSlice(slice []Client, size int) [][]Client {
	var divided [][]Client

	for i := 0; i < len(slice); i += size {
		end := i + size

		if end > len(slice) {
			end = len(slice)
		}

		divided = append(divided, slice[i:end])
	}

	return divided
}
