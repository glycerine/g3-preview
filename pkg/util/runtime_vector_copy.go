// Source: src/util/vector.h:191-216.
package util

// CopyRuntimeVectorColumn implements the source vector copy constructor.
// Callers bind dst to the same Runtime as src and retain failed-copy roots.
func CopyRuntimeVectorColumn(dst, src *RuntimeVectorColumn, stride uint32) error {
	if src.Block == (Block{}) {
		dst.Block = Block{}
		return nil
	}
	n, e := src.Size()
	if e != nil {
		return e
	}
	c, e := src.Capacity()
	if e != nil {
		return e
	}
	h, e := src.Runtime.Allocate(cliqueCopyRequest(c, stride, src.Runtime.SizeTBits()))
	dst.Pending = h
	if e != nil {
		return e
	}
	if e = WriteRuntimeWord(src.Runtime, h, 0, c); e != nil {
		return e
	}
	if e = WriteRuntimeWord(src.Runtime, h, 4, n); e != nil {
		return e
	}
	dst.Block = h
	dst.Pending = Block{}
	for i := uint32(0); i < n; i++ {
		for k := uint32(0); k < stride; k++ {
			b, e := src.Runtime.Read(src.Block, 8+uint64(i)*uint64(stride)+uint64(k))
			if e != nil {
				return e
			}
			if e = dst.Runtime.Write(h, 8+uint64(i)*uint64(stride)+uint64(k), b); e != nil {
				return e
			}
		}
	}
	return nil
}

func cliqueCopyRequest(capacity, stride uint32, bits uint8) uint64 {
	n := uint64(capacity)*uint64(stride) + 8
	if bits == 32 {
		return uint64(uint32(n))
	}
	return n
}
