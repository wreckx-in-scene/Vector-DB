package main

import "strings"

// chunkText mirrors the original C++ chunkText(): split on whitespace,
// then build overlapping windows of chunkWords with overlapWords overlap.
// If the whole text fits in one chunk, returns it unchanged as a single
// element (not re-joined from the split words, to preserve original spacing
// for short texts — matching the C++ behavior of returning {text} as-is).
func chunkText(text string, chunkWords, overlapWords int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) <= chunkWords {
		return []string{text}
	}

	var chunks []string
	step := chunkWords - overlapWords
	for i := 0; i < len(words); i += step {
		end := i + chunkWords
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
