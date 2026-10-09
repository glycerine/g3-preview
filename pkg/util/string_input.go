// Source: src/util/symbol.cpp; src/util/params.cpp.
package util

import "strings"

func nulStringLength(text string) int {
	if i := strings.IndexByte(text, 0); i >= 0 {
		return i
	}
	return len(text)
}
