// Licensed under the MIT license.
// Concrete source util/prime_generator.{h,cpp}, caller-confined owner profile.
package util

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrPrimeCapacity = errors.New("mpnum: prime generator capacity exceeded")
var ErrPrimeNontermination = errors.New("mpnum: source prime growth loop cannot terminate")

// PrimeGenerator owns its scalar cache. Implicit owning Go copies are rejected.
// Explicit owner lifetime replaces the source global generator/finalizer;
// callers may share this owner across borrowed iterators under confinement.
type PrimeGenerator struct {
	self   *PrimeGenerator
	budget *arena.Budget
	primes primeUnsigned64Vector
	header uint64
	Closed bool
}
type primeUnsigned64Vector struct {
	values []uint64
	size   uint32
}
type PrimeGeneratorStats struct {
	Count, Capacity                      uint32
	OwnerBytes, BackingBytes, TotalBytes uint64
}

func primeSourceCapacity(old uint32) (n, oldBytes, newBytes uint32, e error) {
	if old == 0 {
		return 2, 0, 24, nil
	}
	oldBytes = 8*old + 8
	n = (3*old + 1) >> 1
	newBytes = 8*n + 8
	if n <= old || newBytes <= oldBytes {
		e = arena.ErrBounds
	}
	return
}
func primeNextCapacity(old uint32, host uint64) (uint32, uint64, error) {
	n, _, _, e := primeSourceCapacity(old)
	if e != nil {
		return 0, 0, e
	}
	// Separate full-capacity typed storage guard, not a source overflow action.
	if uint64(n)*8+8 > math.MaxUint32 || uint64(n) > host/8 {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * 8, nil
}
func (v *primeUnsigned64Vector) push(b *arena.Budget, value uint64) error {
	if v.size == math.MaxUint32 {
		return arena.ErrBounds
	}
	if uint64(v.size) == uint64(cap(v.values)) {
		n, bytes, e := primeNextCapacity(uint32(cap(v.values)), uint64(int(^uint(0)>>1)))
		if e != nil {
			return e
		}
		if e = b.Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint64, n)
		copy(next, v.values)
		b.Release(uint64(cap(v.values)) * 8)
		v.values = next
	}
	v.values[v.size] = value
	v.size++
	return nil
}
func (v *primeUnsigned64Vector) finalize(b *arena.Budget) {
	b.Release(uint64(cap(v.values)) * 8)
	v.values = nil
	v.size = 0
}
func NewPrimeGenerator(b *arena.Budget) (*PrimeGenerator, error) {
	bytes := uint64(reflect.TypeOf(PrimeGenerator{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	g := &PrimeGenerator{budget: b, header: bytes}
	g.self = g
	e := g.primes.push(b, 2)
	if e == nil {
		e = g.primes.push(b, 3)
	}
	if e == nil {
		e = g.process(128)
	}
	if e != nil {
		g.Close()
		return nil, e
	}
	return g, nil
}
func (g *PrimeGenerator) check() error {
	if g == nil || g.self != g {
		return arena.ErrConfig
	}
	if g.Closed {
		return arena.ErrClosed
	}
	return nil
}

// process translates the source ascending odd candidate/filter/append loops.
// todo is a lexical source svector; its actual heap backing is always refunded
// before a Go error return. The already published cache prefix is retained.
func (g *PrimeGenerator) process(k uint64) error {
	if g.primes.size == 0 {
		return arena.ErrBounds
	} // actual source back() read
	var todo primeUnsigned64Vector
	defer todo.finalize(g.budget)
	begin := g.primes.values[g.primes.size-1] + 2
	end := begin + k
	for i := begin; i < end; i += 2 {
		if e := todo.push(g.budget, i); e != nil {
			return e
		}
	}
	j := uint32(1)
	if modularSourceDebug && (g.primes.size <= j || g.primes.values[j] != 3) {
		return ErrNumericCallerAssertion
	}
	for todo.size != 0 {
		sz := g.primes.size
		for ; j < sz; j++ {
			p := g.primes.values[j]
			todoSize := todo.size
			k1, k2 := uint32(0), uint32(0)
			for ; k1 < todoSize; k1++ {
				if todo.values[k1]%p == 0 {
					continue
				}
				todo.values[k2] = todo.values[k1]
				k2++
			}
			todo.size = k2
			if k2 == 0 {
				return nil
			}
			if p > todo.values[k2-1]/p+1 {
				for k1 := uint32(0); k1 < k2; k1++ {
					if e := g.primes.push(g.budget, todo.values[k1]); e != nil {
						return e
					}
				}
				return nil
			}
		}
		p := g.primes.values[g.primes.size-1]
		p = p * p
		todoSize := todo.size
		k1 := uint32(0)
		for ; k1 < todoSize; k1++ {
			if todo.values[k1] < p {
				if e := g.primes.push(g.budget, todo.values[k1]); e != nil {
					return e
				}
			} else {
				break
			}
		}
		k2 := uint32(0)
		for ; k1 < todoSize; k1, k2 = k1+1, k2+1 {
			todo.values[k2] = todo.values[k1]
		}
		todo.size = k2
	}
	return nil
}
func (g *PrimeGenerator) At(index uint32) (uint64, error) {
	if e := g.check(); e != nil {
		return 0, e
	}
	if index < g.primes.size {
		return g.primes.values[index], nil
	}
	if index > 1<<20 {
		return 0, ErrPrimeCapacity
	}
	if e := g.process(1024); e != nil {
		return 0, e
	}
	if index < g.primes.size {
		return g.primes.values[index], nil
	}
	// Literal source inequality is reversed. Once true, cache size never
	// decreases, so the loop cannot terminate. Execute its first whole body
	// (allowing actual allocation refusal), then guard its repeated condition.
	for index <= g.primes.size {
		if e := g.process(1024 * 16); e != nil {
			return 0, e
		}
		if index <= g.primes.size {
			return 0, ErrPrimeNontermination
		}
	}
	if index >= g.primes.size {
		return 0, arena.ErrBounds
	} // actual source indexed read
	return g.primes.values[index], nil
}
func (g *PrimeGenerator) Finalize() error {
	if e := g.check(); e != nil {
		return e
	}
	g.primes.finalize(g.budget)
	return nil
}
func (g *PrimeGenerator) Close() error {
	if e := g.check(); e != nil {
		return e
	}
	g.primes.finalize(g.budget)
	g.budget.Release(g.header)
	g.Closed = true
	return nil
}
func (g *PrimeGenerator) Stats() PrimeGeneratorStats {
	if g == nil || g.self != g || g.Closed {
		return PrimeGeneratorStats{}
	}
	s := PrimeGeneratorStats{Count: g.primes.size, Capacity: uint32(cap(g.primes.values)), OwnerBytes: g.header, BackingBytes: uint64(cap(g.primes.values)) * 8}
	s.TotalBytes = s.OwnerBytes + s.BackingBytes
	return s
}

// PrimeIterator is a nonowning concrete value. Its source uint32 index wraps
// and increments before the actual generator action, including all failures.
type PrimeIterator struct {
	generator *PrimeGenerator
	Index     uint32
}

func NewPrimeIterator(g *PrimeGenerator) (PrimeIterator, error) {
	if e := g.check(); e != nil {
		return PrimeIterator{}, e
	}
	return PrimeIterator{generator: g}, nil
}
func (it *PrimeIterator) Next() (uint64, error) {
	if it == nil {
		return 0, arena.ErrConfig
	}
	index := it.Index
	it.Index++
	return it.generator.At(index)
}
