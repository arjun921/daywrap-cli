package internal

// chunk.go — splits the base64-encoded compressed payload into QR-sized chunks
// prefixed with DW:<index>:<total>:.

import "fmt"

// Chunk splits encoded (a base64url string) into DW-protocol frames.
// Each frame is a deep-linkable URL:
//
//	daywrap://scan?d=DW:<i>:<n>:<payload>
//
// maxPayload controls the payload portion length. Pass the result of
// OptimalChunkSize so frames fit the current terminal.
// Frame sizes are equalised in Display() via equalisedChunks().
func Chunk(encoded string, maxPayload int) []string {
	if maxPayload <= 0 {
		maxPayload = 2460
	}
	n := (len(encoded) + maxPayload - 1) / maxPayload
	if n == 0 {
		n = 1
	}
	chunks := make([]string, 0, n)
	for i := 0; i < n; i++ {
		start := i * maxPayload
		end := start + maxPayload
		if end > len(encoded) {
			end = len(encoded)
		}
		chunks = append(chunks, fmt.Sprintf("daywrap://scan?d=DW:%d:%d:%s", i+1, n, encoded[start:end]))
	}
	return chunks
}

