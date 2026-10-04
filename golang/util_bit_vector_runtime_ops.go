// Source: src/util/bit_vector.cpp:82-235; src/util/bit_vector.h:116-239.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package g3

import (
	"github.com/glycerine/g3/golang/utilhash"
	"io"
)

func (v *RuntimeBitVector) ShiftRight(k uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if k == 0 {
		return nil
	}
	newBits := v.numBits + k
	oldWords := bitVectorNumWords(v.numBits)
	newWords := bitVectorNumWords(newBits)
	if e := v.Resize(newBits, false); e != nil {
		return e
	}
	bs, ws := k%32, k/32
	if ws > 0 {
		j, i := oldWords, oldWords+ws
		for j > 0 {
			j--
			i--
			x, e := v.read(j)
			if e != nil {
				return e
			}
			if e = v.write(i, x); e != nil {
				return e
			}
		}
		for i > 0 {
			i--
			if e := v.write(i, 0); e != nil {
				return e
			}
		}
	}
	if bs > 0 {
		if satStorageSourceDebug {
			for i := uint32(0); i < ws; i++ {
				x, e := v.read(i)
				if e != nil {
					return e
				}
				if x != 0 {
					return ErrSATAssertion
				}
			}
		}
		prev := uint32(0)
		for i := ws; i < newWords; i++ {
			x, e := v.read(i)
			if e != nil {
				return e
			}
			np := x >> (32 - bs)
			if e = v.write(i, x<<bs); e != nil {
				return e
			}
			x, e = v.read(i)
			if e != nil {
				return e
			}
			if e = v.write(i, x|prev); e != nil {
				return e
			}
			prev = np
		}
	}
	return nil
}
func (v *RuntimeBitVector) Equal(src *RuntimeBitVector) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	if e := src.open(); e != nil {
		return false, e
	}
	if v.numBits != src.numBits {
		return false, nil
	}
	n := bitVectorNumWords(v.numBits)
	if n == 0 {
		return true, nil
	}
	for i := uint32(0); i < n-1; i++ {
		a, e := v.read(i)
		if e != nil {
			return false, e
		}
		b, e := src.read(i)
		if e != nil {
			return false, e
		}
		if a != b {
			return false, nil
		}
	}
	a, e := v.read(n - 1)
	if e != nil {
		return false, e
	}
	b, e := src.read(n - 1)
	if e != nil {
		return false, e
	}
	mask := (uint32(1) << (src.numBits % 32)) - 1
	if mask == 0 {
		mask = ^uint32(0)
	}
	return a&mask == b&mask, nil
}
func (v *RuntimeBitVector) Union(src *RuntimeBitVector) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v.numBits < src.numBits {
		if e := v.Resize(src.numBits, false); e != nil {
			return e
		}
	}
	n := bitVectorNumWords(src.numBits)
	if satStorageSourceDebug && n > bitVectorNumWords(v.numBits) {
		return ErrSATAssertion
	}
	rest := src.numBits % 32
	end := n
	if rest != 0 {
		end = n - 1
	}
	for i := uint32(0); i < end; i++ {
		a, e := v.read(i)
		if e != nil {
			return e
		}
		b, e := src.read(i)
		if e != nil {
			return e
		}
		if e = v.write(i, a|b); e != nil {
			return e
		}
	}
	if rest != 0 {
		a, e := v.read(end)
		if e != nil {
			return e
		}
		b, e := src.read(end)
		if e != nil {
			return e
		}
		return v.write(end, a|b&((uint32(1)<<rest)-1))
	}
	return nil
}
func (v *RuntimeBitVector) Intersect(src *RuntimeBitVector) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	n1, n2 := bitVectorNumWords(v.numBits), bitVectorNumWords(src.numBits)
	if n1 == 0 {
		return nil
	}
	end := n1
	rest := uint32(0)
	if n2 <= n1 {
		end = n2
		rest = src.numBits % 32
		if rest != 0 {
			end = n2 - 1
		}
	}
	for i := uint32(0); i < end; i++ {
		a, e := v.read(i)
		if e != nil {
			return e
		}
		b, e := src.read(i)
		if e != nil {
			return e
		}
		if e = v.write(i, a&b); e != nil {
			return e
		}
	}
	if n2 <= n1 {
		if rest != 0 {
			a, e := v.read(end)
			if e != nil {
				return e
			}
			b, e := src.read(end)
			if e != nil {
				return e
			}
			if e = v.write(end, a&(b&((uint32(1)<<rest)-1))); e != nil {
				return e
			}
		}
		return v.zero(n2, n1, 0)
	}
	return nil
}
func (v *RuntimeBitVector) Contains(src *RuntimeBitVector) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	if e := src.open(); e != nil {
		return false, e
	}
	n := bitVectorNumWords(v.numBits)
	if n == 0 {
		return true, nil
	}
	for i := uint32(0); i < n-1; i++ {
		a, e := v.read(i)
		if e != nil {
			return false, e
		}
		b, e := src.read(i)
		if e != nil {
			return false, e
		}
		b2, e := src.read(i)
		if e != nil {
			return false, e
		}
		if a&b != b2 {
			return false, nil
		}
	}
	mask := (uint32(1) << (v.numBits % 32)) - 1
	if mask == 0 {
		mask = ^uint32(0)
	}
	other, e := src.read(n - 1)
	if e != nil {
		return false, e
	}
	other &= mask
	a, e := v.read(n - 1)
	return a&other == other, e
}
func (v *RuntimeBitVector) Neg() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, n := uint32(0), bitVectorNumWords(v.numBits); i < n; i++ {
		w, e := v.read(i)
		if e != nil {
			return e
		}
		if e = v.write(i, ^w); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeBitVector) Hash() (uint32, error) {
	if e := v.scalarOpen(); e != nil {
		return 0, e
	}
	return utilhash.RuntimeBytes(v.runtime, v.block, 0, v.numBits/8, 0)
}
func (v *RuntimeBitVector) Display(out io.Writer) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := v.numBits; i > 0; {
		i--
		b, e := v.Get(i)
		if e != nil {
			return e
		}
		x := []byte{'0'}
		if b {
			x[0] = '1'
		}
		n, e := out.Write(x)
		if e != nil {
			return e
		}
		if n != 1 {
			return io.ErrShortWrite
		}
	}
	return nil
}
