// Source: src/util/machine.h:20-29.
package util

import (
	"math/bits"
	"unsafe"
)

// PointerAlignmentBytes is the alignment Go assigns to uintptr on this target.
// It replaces the source compiler's AMD64/non-AMD64 preprocessor split.
const PointerAlignmentBytes = uint(unsafe.Alignof(uintptr(0)))

// PointerAlignmentShift is the number of low zero bits guaranteed by the host
// pointer alignment. It is the Go-target translation of PTR_ALIGNMENT.
func PointerAlignmentShift() uint {
	return uint(bits.TrailingZeros(PointerAlignmentBytes))
}
