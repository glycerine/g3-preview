// Licensed under the MIT license.
// Translation of random_gen and shuffle in src/util/util.h.
package utilhash

const RandomMax uint32 = 0x7fff

// Random owns source random_gen state; its zero value has seed zero.
// Concurrent callers must synchronize owner state.
type Random struct{ data uint32 }

func NewRandom(seed uint32) *Random   { return &Random{data: seed} }
func (r *Random) SetSeed(seed uint32) { r.data = seed }
func (r *Random) Next() uint32        { r.data = r.data*214013 + 2531011; return (r.data >> 16) & RandomMax }

// Bounded requires a nonzero bound, like the source.
func (r *Random) Bounded(u uint32) uint32 { return r.Next() % u }

// ShuffleUint32 translates shuffle for unsigned 32-bit source-domain values.
func ShuffleUint32(values []uint32, r *Random) {
	for n := len(values) - 1; n > 0; n-- {
		k := int(r.Next()) % (n + 1)
		values[n], values[k] = values[k], values[n]
	}
}

// ShuffleInt32 translates shuffle for signed 32-bit source-domain values.
func ShuffleInt32(values []int32, r *Random) {
	for n := len(values) - 1; n > 0; n-- {
		k := int(r.Next()) % (n + 1)
		values[n], values[k] = values[k], values[n]
	}
}
