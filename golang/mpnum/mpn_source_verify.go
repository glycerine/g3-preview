// Licensed under the MIT license.
// Source: src/util/mpn.cpp:206-217 and actual Algorithm M/A bodies.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// sourceVerifyDivision is the source Z3DEBUG lexical temp, after both outputs.
// The embedded charged holder retains a failed lexical construction/destruction
// only for explicit Go owning cleanup. Successful calls destroy it immediately;
// its allocation is never reused as source manager scratch.
func (m *NaturalManager) sourceVerifyDivision(a, b, q, r limbView) error {
	if !modularSourceDebug {
		return nil
	}
	// Guard reuse of this retained lexical holder only when the source reaches
	// its constructor, after the quotient and remainder have been published.
	if m.verificationActive {
		return arena.ErrInvalidSlot
	}
	m.verificationActive = true
	temp := &m.verification
	if e := temp.resize(m.runtime, a.size+1); e != nil {
		return e
	}
	out := limbView{runtime: m.runtime, source: temp, size: a.size + 1}
	// mpn_manager::mul, Knuth M: initialize lnga cells, then every column.
	for i := uint32(0); i < q.size; i++ {
		out.write(i, 0)
	}
	for j := uint32(0); j < b.size; j++ {
		digit := b.read(j)
		if digit == 0 {
			out.write(j+q.size, 0)
			continue
		}
		var carry uint32
		for i := uint32(0); i < q.size; i++ {
			value := uint64(q.read(i))*uint64(digit) + uint64(out.read(i+j)) + uint64(carry)
			out.write(i+j, uint32(value))
			carry = uint32(value >> 32)
		}
		out.write(j+q.size, carry)
	}
	// Source aliasing add reads temp at each position before writing that cell.
	var carry uint32
	for i := uint32(0); i < a.size; i++ {
		u := out.read(i)
		var v uint32
		if i < r.size {
			v = r.read(i)
		}
		sum := u + v
		c1 := sum < u
		value := sum + carry
		c2 := value < sum
		out.write(i, value)
		carry = 0
		if c1 || c2 {
			carry = 1
		}
	}
	out.write(a.size, carry)
	// The source add also walks the normalized size even though the caller does
	// not consume it. Preserve these reads rather than replacing the algorithm.
	realSize := a.size + 1
	for realSize > 1 && out.read(realSize-1) == 0 {
		realSize--
	}
	ok := true
	for i := uint32(0); i < a.size && ok; i++ {
		if out.read(i) != a.read(i) {
			ok = false
		}
	}
	if out.read(a.size) != 0 {
		ok = false
	}
	if !ok {
		return ErrIntegerAssertion
	}
	if e := temp.finalize(m.runtime); e != nil {
		return e
	}
	m.verificationActive = false
	return nil
}
