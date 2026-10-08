// Source: src/util/machine.h:20-29.
package util

import (
	"math/bits"
	"testing"
	"unsafe"
)

func TestMachinePointerAlignmentFromGoTarget(t *testing.T) {
	want := uint(unsafe.Alignof(uintptr(0)))
	if PointerAlignmentBytes != want || PointerAlignmentBytes == 0 || PointerAlignmentBytes&(PointerAlignmentBytes-1) != 0 {
		t.Fatal("pointer alignment", PointerAlignmentBytes, want)
	}
	if PointerAlignmentShift() != uint(bits.TrailingZeros(want)) || uint(1)<<PointerAlignmentShift() != want {
		t.Fatal("pointer alignment shift", PointerAlignmentShift(), want)
	}
	if PointerTagMask() != uintptr(want-1) {
		t.Fatal("pointer tag mask", PointerTagMask(), want-1)
	}
}

func TestMachinePointerAlignmentCoversAllocatedPointer(t *testing.T) {
	x := new(uintptr)
	pointer := uintptr(unsafe.Pointer(x))
	if pointer&PointerTagMask() != 0 {
		t.Fatal("allocated pointer violates reported alignment", pointer, PointerTagMask())
	}
}
