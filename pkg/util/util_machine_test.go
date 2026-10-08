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
}

func TestMachinePointerAlignmentCoversAllocatedPointer(t *testing.T) {
	x := new(uintptr)
	pointer := uintptr(unsafe.Pointer(x))
	mask := uintptr(PointerAlignmentBytes - 1)
	if pointer&mask != 0 {
		t.Fatal("allocated pointer violates reported alignment", pointer, mask)
	}
}
