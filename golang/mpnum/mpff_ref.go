//go:build !g3checked

// Copyright (c) 2012 Microsoft Corporation. Concrete MPFF scalar packing.
package mpnum

type ffRef uint32

func ffPack(index, generation uint32, negative bool) ffRef {
	v := index
	if negative {
		v |= 1 << 31
	}
	return ffRef(v)
}
func ffGeneration(r ffRef) uint32 { return 0 }
