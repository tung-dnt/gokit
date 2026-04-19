package chunker

import "strings"

// Chunker splits text into overlapping word-based segments.
type Chunker struct {
	chunkSize int
	overlap   int
}

// New creates a Chunker with the given chunk size and overlap (in words).
func New(chunkSize, overlap int) *Chunker {
	return &Chunker{chunkSize: chunkSize, overlap: overlap}
}

// Chunk splits text into overlapping segments and returns them as strings.
func (c *Chunker) Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var segments []string
	for start := 0; start < len(words); start += c.chunkSize - c.overlap {
		end := min(start+c.chunkSize, len(words))

		segments = append(segments, strings.Join(words[start:end], " "))

		if end == len(words) {
			break
		}
	}

	return segments
}
