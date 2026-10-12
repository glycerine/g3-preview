// Source: src/util/prime_generator.h:26-52; src/util/prime_generator.cpp:23-130.
package prime

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrCapacity = errors.New("mpnum: prime generator capacity exceeded")
var ErrNontermination = errors.New("mpnum: source prime growth loop cannot terminate")
var ErrAssertion = errors.New("mpnum: prime generator source assertion")

// Generator owns its scalar cache. Implicit owning Go copies are rejected.
// Explicit owner lifetime replaces the source global generator/finalizer;
// callers may share this owner across borrowed iterators under confinement.
type Generator struct {
	self   *Generator
	budget *arena.Budget
	primes []uint64
	header uint64
	closed bool
}
type Stats struct {
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
func appendPrimeValue(b *arena.Budget, values *[]uint64, value uint64) error {
	if uint64(len(*values)) == math.MaxUint32 {
		return arena.ErrBounds
	}
	if len(*values) == cap(*values) {
		n, bytes, e := primeNextCapacity(uint32(cap(*values)), uint64(int(^uint(0)>>1)))
		if e != nil {
			return e
		}
		if e = b.Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint64, len(*values), n)
		copy(next, *values)
		b.Release(uint64(cap(*values)) * 8)
		*values = next
	}
	*values = append(*values, value)
	return nil
}
func releasePrimeValues(b *arena.Budget, values *[]uint64) {
	b.Release(uint64(cap(*values)) * 8)
	*values = nil
}
func NewGenerator(b *arena.Budget) (*Generator, error) {
	bytes := uint64(reflect.TypeOf(Generator{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	g := &Generator{budget: b, header: bytes}
	g.self = g
	e := appendPrimeValue(b, &g.primes, 2)
	if e == nil {
		e = appendPrimeValue(b, &g.primes, 3)
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
func (g *Generator) check() error {
	if g == nil || g.self != g {
		return arena.ErrConfig
	}
	if g.closed {
		return arena.ErrClosed
	}
	return nil
}

func (g *Generator) Check() error { return g.check() }

// process translates the source ascending odd candidate/filter/append loops.
// The temporary candidate list uses native Go slice storage; the already
// published prime-cache prefix is retained if its allocation is refused.
func (g *Generator) process(k uint64) error {
	if len(g.primes) == 0 {
		return arena.ErrBounds
	} // actual source back() read
	var todo []uint64
	begin := g.primes[len(g.primes)-1] + 2
	end := begin + k
	for i := begin; i < end; i += 2 {
		todo = append(todo, i)
	}
	j := uint32(1)
	if sourceDebug && (uint32(len(g.primes)) <= j || g.primes[j] != 3) {
		return ErrAssertion
	}
	for len(todo) != 0 {
		sz := uint32(len(g.primes))
		for ; j < sz; j++ {
			p := g.primes[j]
			todoSize := uint32(len(todo))
			k1, k2 := uint32(0), uint32(0)
			for ; k1 < todoSize; k1++ {
				if todo[k1]%p == 0 {
					continue
				}
				todo[k2] = todo[k1]
				k2++
			}
			todo = todo[:k2]
			if k2 == 0 {
				return nil
			}
			if p > todo[k2-1]/p+1 {
				for k1 := uint32(0); k1 < k2; k1++ {
					if e := appendPrimeValue(g.budget, &g.primes, todo[k1]); e != nil {
						return e
					}
				}
				return nil
			}
		}
		p := g.primes[len(g.primes)-1]
		p = p * p
		todoSize := uint32(len(todo))
		k1 := uint32(0)
		for ; k1 < todoSize; k1++ {
			if todo[k1] < p {
				if e := appendPrimeValue(g.budget, &g.primes, todo[k1]); e != nil {
					return e
				}
			} else {
				break
			}
		}
		k2 := uint32(0)
		for ; k1 < todoSize; k1, k2 = k1+1, k2+1 {
			todo[k2] = todo[k1]
		}
		todo = todo[:k2]
	}
	return nil
}
func (g *Generator) At(index uint32) (uint64, error) {
	if e := g.check(); e != nil {
		return 0, e
	}
	if index < uint32(len(g.primes)) {
		return g.primes[index], nil
	}
	if index > 1<<20 {
		return 0, ErrCapacity
	}
	if e := g.process(1024); e != nil {
		return 0, e
	}
	if index < uint32(len(g.primes)) {
		return g.primes[index], nil
	}
	// Literal source inequality is reversed. Once true, cache size never
	// decreases, so the loop cannot terminate. Execute its first whole body
	// (allowing actual allocation refusal), then guard its repeated condition.
	for index <= uint32(len(g.primes)) {
		if e := g.process(1024 * 16); e != nil {
			return 0, e
		}
		if index <= uint32(len(g.primes)) {
			return 0, ErrNontermination
		}
	}
	if index >= uint32(len(g.primes)) {
		return 0, arena.ErrBounds
	} // actual source indexed read
	return g.primes[index], nil
}
func (g *Generator) Finalize() error {
	if e := g.check(); e != nil {
		return e
	}
	releasePrimeValues(g.budget, &g.primes)
	return nil
}
func (g *Generator) Close() error {
	if e := g.check(); e != nil {
		return e
	}
	releasePrimeValues(g.budget, &g.primes)
	g.budget.Release(g.header)
	g.closed = true
	return nil
}
func (g *Generator) Closed() bool { return g == nil || g.self != g || g.closed }
func (g *Generator) Stats() Stats {
	if g == nil || g.self != g || g.closed {
		return Stats{}
	}
	s := Stats{Count: uint32(len(g.primes)), Capacity: uint32(cap(g.primes)), OwnerBytes: g.header, BackingBytes: uint64(cap(g.primes)) * 8}
	s.TotalBytes = s.OwnerBytes + s.BackingBytes
	return s
}

// Iterator is a nonowning concrete value. Its source uint32 index wraps
// and increments before the actual generator action, including all failures.
type Iterator struct {
	generator *Generator
	Index     uint32
}

func NewIterator(g *Generator) (Iterator, error) {
	if e := g.check(); e != nil {
		return Iterator{}, e
	}
	return Iterator{generator: g}, nil
}
func (it *Iterator) Next() (uint64, error) {
	if it == nil {
		return 0, arena.ErrConfig
	}
	index := it.Index
	it.Index++
	return it.generator.At(index)
}
