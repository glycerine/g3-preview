// Source: src/util/file_path.h:23-35.
package util

// GetExtension returns a borrowed suffix after the last dot before the first
// NUL byte (or the end of the slice). Nil means no dot or a null input; a
// non-nil empty slice represents a trailing dot. Dots in directory names
// count too, as in the source. Treat the returned bytes as read-only.
func GetExtension(fileName []byte) []byte {
	lastDot := -1
	end := 0
	for end < len(fileName) && fileName[end] != 0 {
		if fileName[end] == '.' {
			lastDot = end + 1
		}
		end++
	}
	if lastDot < 0 {
		return nil
	}
	return fileName[lastDot:end]
}
