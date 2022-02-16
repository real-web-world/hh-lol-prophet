package bdk

func ArrChunkStr(arr []string, size int) [][]string {
	chunks := make([][]string, 0)
	chunk := make([]string, 0, size)
	for i := 0; i < len(arr); i++ {
		chunk = append(chunk, arr[i])
		if len(chunk) >= size {
			chunks = append(chunks, chunk)
			chunk = make([]string, 0, size)
		}
	}
	return chunks
}
func ArrChunkI64(arr []int64, size int) [][]int64 {
	chunks := make([][]int64, 0)
	chunk := make([]int64, 0, size)
	for i := 0; i < len(arr); i++ {
		chunk = append(chunk, arr[i])
		if len(chunk) >= size {
			chunks = append(chunks, chunk)
			chunk = make([]int64, 0, size)
		}
	}
	return chunks
}
