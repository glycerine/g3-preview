//go:build !g3checked

// Concrete MPFF scalar packing.
package numeral

type ffRef uint32

func ffPack(index, generation uint32, negative bool) ffRef {
	v := index
	if negative {
		v |= 1 << 31
	}
	return ffRef(v)
}
func ffGeneration(r ffRef) uint32 { return 0 }
