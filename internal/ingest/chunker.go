package ingest

import (
	"strings"
)

// ChunkText splits a large text into smaller chunks based on paragraphs.
// In a production environment, you might want to use a more advanced token-based chunker.
func ChunkText(text string, minLength int) []string {
	paragraphs := strings.Split(text, "\n")
	var chunks []string
	var currentChunk strings.Builder

	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if len(trimmed) == 0 {
			continue
		}

		currentChunk.WriteString(trimmed)
		currentChunk.WriteString(" ")

		// If the current accumulated string is large enough, save it as a chunk
		if currentChunk.Len() >= minLength {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
	}

	// Add the last remaining chunk if it's not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}