package arena_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
)

// Actual rational numerals are pointer-free scalar records. Pool32 stores the
// fixture cell index; one cold caller-owned array contains coefficient values.
// This adapter is not a public port of util/rational.h's wrapper or source cell.
type rationalPoolCell struct {
	coeff                  mpnum.Rational
	i, j, nextRow, nextCol uint32
	initialized            bool
}
type rationalPoolPair struct {
	ref   arena.Pool32Ref
	value int64
}
type rationalPoolFixture struct {
	budget                           *arena.Budget
	q                                *mpnum.RationalManager
	pool                             *arena.Pool32
	cells                            []rationalPoolCell
	pairs                            []rationalPoolPair
	constructed, destroyed, recycled uint32
	fixtureBytes                     uint64
	closed                           bool
}

func newRationalPoolFixture(t *testing.T, capacity uint32, pageSize uint32) *rationalPoolFixture {
	t.Helper()
	if e := arena.ValidateRecordType(reflect.TypeOf(rationalPoolCell{})); e != nil {
		t.Fatal("pointer-free actual rational cell", e)
	}
	b := arena.NewBudget(64 << 20)
	bytes := uint64(reflect.TypeOf(rationalPoolFixture{}).Size()) + uint64(capacity)*(uint64(reflect.TypeOf(rationalPoolCell{}).Size())+uint64(reflect.TypeOf(rationalPoolPair{}).Size()))
	if e := b.Reserve(bytes); e != nil {
		t.Fatal(e)
	}
	q, e := mpnum.NewRationalManager(mpnum.RationalConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if e != nil {
		b.Release(bytes)
		t.Fatal(e)
	}
	f := &rationalPoolFixture{budget: b, q: q, cells: make([]rationalPoolCell, capacity), pairs: make([]rationalPoolPair, 0, capacity), fixtureBytes: bytes}
	p, e := arena.NewPool32(arena.Pool32Config{PageSize: pageSize, Budget: b}, arena.Pool32Policy{Construct: f.construct, Reset: f.reset, Destroy: f.destroy})
	if e != nil {
		q.Close()
		b.Release(bytes)
		t.Fatal(e)
	}
	f.pool = p
	t.Cleanup(func() { f.close(t) })
	return f
}

// Callbacks allocate no fixture storage and make no rational Reserve calls.
// Invalid coefficient ownership/closed manager is an invariant failure; callbacks
// must never receive it. Manager teardown occurs strictly after pool teardown.
func (f *rationalPoolFixture) construct() uint32 {
	if f.constructed >= uint32(len(f.cells)) {
		panic("bounded rational fixture exhausted")
	}
	id := f.constructed + 1
	f.cells[id-1] = rationalPoolCell{coeff: mpnum.NewRational(), initialized: true}
	f.constructed++
	return id
}
func (f *rationalPoolFixture) reset(id uint32) uint32 {
	c := &f.cells[id-1]
	if !c.initialized {
		panic("reset of destroyed rational cell")
	}
	if e := f.q.Reset(&c.coeff); e != nil {
		panic(e)
	}
	f.recycled++
	return id
}
func (f *rationalPoolFixture) destroy(id uint32) {
	c := &f.cells[id-1]
	if !c.initialized {
		panic("duplicate rational destruction")
	}
	// C++ destructor body increments its counter before member destruction.
	f.destroyed++
	if e := f.q.Del(&c.coeff); e != nil {
		panic(e)
	}
	c.initialized = false
}
func (f *rationalPoolFixture) cell(t *testing.T, r arena.Pool32Ref) *rationalPoolCell {
	t.Helper()
	id, e := f.pool.Read(r)
	if e != nil {
		t.Fatal(e)
	}
	if id == 0 || id > uint32(len(f.cells)) || !f.cells[id-1].initialized {
		t.Fatal("invalid fixture cell identity")
	}
	return &f.cells[id-1]
}
func (f *rationalPoolFixture) ensureZero(t *testing.T, r arena.Pool32Ref) {
	t.Helper()
	zero := mpnum.NewRational()
	ok, e := f.q.Equal(&f.cell(t, r).coeff, &zero)
	if e != nil || !ok {
		t.Fatal("source coefficient zero assertion", e)
	}
}
func (f *rationalPoolFixture) close(t *testing.T) {
	t.Helper()
	if f.closed {
		return
	}
	f.budget.FailAfter(-1)
	if e := f.pool.Close(); e != nil {
		t.Fatal(e)
	}
	if f.constructed != f.destroyed {
		t.Fatalf("source constructor/destructor balance %d/%d", f.constructed, f.destroyed)
	}
	for i := uint32(0); i < f.constructed; i++ {
		if f.cells[i].initialized {
			t.Fatal("initialized rational survives destruction")
		}
	}
	f.q.Close()
	f.budget.Release(f.fixtureBytes)
	f.closed = true
	if f.budget.Used() != 0 {
		t.Fatal("real rational fixture owner leak", f.budget.Used())
	}
}

// Source src/test/object_allocator.cpp:tst1's rational(10), recycle, zero ENSURE.
func TestPool32SourceRationalTst1(t *testing.T) {
	f := newRationalPoolFixture(t, 4, 2)
	c1, e := f.pool.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.pool.Allocate(); e != nil {
		t.Fatal(e)
	}
	if e = f.q.SetInt64(&f.cell(t, c1).coeff, 10); e != nil {
		t.Fatal(e)
	}
	if e = f.pool.Recycle(c1); e != nil {
		t.Fatal(e)
	}
	c3, e := f.pool.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	f.ensureZero(t, c3)
	if f.constructed != 2 || f.recycled != 1 || f.destroyed != 0 {
		t.Fatal("source retained constructed state lifecycle")
	}
	if f.pool.IsLive(c1) {
		t.Fatal("recycled rational identity remained live")
	}
	f.close(t)
}

// The source supplies no fixed C rand sequence. This finite deterministic
// nonnegative generator preserves its operation shapes and original100000 count.
type rationalPoolRand struct{ state uint32 }

func (r *rationalPoolRand) next() uint32 { r.state = r.state*1664525 + 1013904223; return r.state >> 1 }

func TestPool32SourceRationalTst2(t *testing.T) {
	const operations = 100000
	f := newRationalPoolFixture(t, operations+2, 1024)
	if f.pool.Stats().Workers < 2 {
		t.Fatal("source capacity>=2")
	}
	m1, e := f.pool.Worker(0)
	if e != nil {
		t.Fatal(e)
	}
	m2, e := f.pool.Worker(1)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.pool.EnableConcurrent(true); e != nil {
		t.Fatal(e)
	}
	rng := rationalPoolRand{state: 0x13579bdf}
	expected := mpnum.NewRational()
	resets, recycles := uint32(0), uint32(0)
	var destinations, allocations [2]uint32
	for i := 0; i < operations; i++ {
		choice := rng.next() % 6
		if choice < 4 {
			worker := m1
			workerID := uint32(0)
			if choice >= 2 {
				worker = m2
				workerID = 1
			}
			r, e := worker.Allocate()
			if e != nil {
				t.Fatal(e)
			}
			f.ensureZero(t, r)
			allocations[workerID]++
			val := int64(rng.next())
			if e = f.q.SetInt64(&f.cell(t, r).coeff, val); e != nil {
				t.Fatal(e)
			}
			f.pairs = append(f.pairs, rationalPoolPair{r, val})
		} else if len(f.pairs) > 0 {
			// The source inner idx SHADOWS outer operation idx. The selected pair
			// index, not choice4/5, determines recycle worker(idx<5→worker0).
			idx := int(rng.next() % uint32(len(f.pairs)))
			pair := f.pairs[idx]
			if e = f.q.SetInt64(&expected, pair.value); e != nil {
				t.Fatal(e)
			}
			ok, e := f.q.Equal(&f.cell(t, pair.ref).coeff, &expected)
			if e != nil || !ok {
				t.Fatalf("source coefficient equality at operation%d", i)
			}
			if idx < 5 {
				e = m1.Recycle(pair.ref)
				destinations[0]++
			} else {
				e = m2.Recycle(pair.ref)
				destinations[1]++
			}
			if e != nil {
				t.Fatal(e)
			}
			recycles++
			copy(f.pairs[idx:], f.pairs[idx+1:])
			f.pairs[len(f.pairs)-1] = rationalPoolPair{}
			f.pairs = f.pairs[:len(f.pairs)-1]
		}
		if rng.next()%5000 == 0 {
			if e = f.pool.EnableConcurrent(false); e != nil {
				t.Fatal(e)
			}
			var stale arena.Pool32Ref
			if len(f.pairs) > 0 {
				stale = f.pairs[0].ref
			}
			if e = f.pool.Reset(); e != nil {
				t.Fatal(e)
			}
			if stale.Slot != 0 && f.pool.IsLive(stale) {
				t.Fatal("pool reset retained old rational identity")
			}
			clear(f.pairs)
			f.pairs = f.pairs[:0]
			if e = f.pool.EnableConcurrent(true); e != nil {
				t.Fatal(e)
			}
			resets++
		}
	}
	if resets == 0 || destinations[0] == 0 || destinations[1] == 0 || allocations[0] == 0 || allocations[1] == 0 {
		t.Fatal("deterministic source script missed worker/reset branches", resets, destinations, allocations)
	}
	if f.recycled != recycles {
		t.Fatal("reset procedure count differs from recycle operations")
	}
	// Added final surviving-value oracle; source checks values only at recycle.
	for _, pair := range f.pairs {
		if e = f.q.SetInt64(&expected, pair.value); e != nil {
			t.Fatal(e)
		}
		ok, e := f.q.Equal(&f.cell(t, pair.ref).coeff, &expected)
		if e != nil || !ok {
			t.Fatal("final surviving rational changed")
		}
	}
	if e = f.q.Del(&expected); e != nil {
		t.Fatal(e)
	}
	t.Logf("source100000 operations: resets=%d recycle destinations=%v allocations=%v constructed=%d recycled=%d", resets, destinations, allocations, f.constructed, f.recycled)
	f.close(t)
}

// Independent extension: true nonintegral, big-cell rationals exercise actual
// numerical release in reset/destruction; source integer-valued fixture alone
// cannot establish those paths. Callbacks remain allocation-free even with an
// injected zero reservation allowance.
func TestPool32RationalBigFractionLifetime(t *testing.T) {
	f := newRationalPoolFixture(t, 4, 2)
	r, e := f.pool.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	c := f.cell(t, r)
	if e = f.q.SetFraction(&c.coeff, math.MinInt64, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	if c.coeff.IsSmall() {
		t.Fatal("fixture failed to use actual big rational cells")
	}
	if got, e := f.q.String(&c.coeff); e != nil || got != "-9223372036854775808/18446744073709551615" {
		t.Fatal("actual coefficient mathematics", got, e)
	}
	c.i, c.j, c.nextRow, c.nextCol = 3, 7, 19, 23
	// Warm destination free metadata once; the next recycle has no allocator grow.
	if e = f.pool.Recycle(r); e != nil {
		t.Fatal(e)
	}
	r, e = f.pool.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	c = f.cell(t, r)
	if e = f.q.SetFraction(&c.coeff, math.MinInt64, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	f.budget.FailAfter(0)
	if e = f.pool.Recycle(r); e != nil {
		t.Fatal("valid rational reset requested allocation", e)
	}
	r, e = f.pool.Allocate()
	if e != nil {
		t.Fatal(e)
	}
	f.ensureZero(t, r)
	c = f.cell(t, r)
	if c.i != 3 || c.j != 7 || c.nextRow != 19 || c.nextCol != 23 {
		t.Fatal("reset changed noncoefficient cell state")
	}
	f.budget.FailAfter(-1)
	if e = f.q.SetFraction(&c.coeff, math.MinInt64, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	f.budget.FailAfter(0)
	if e = f.pool.Close(); e != nil {
		t.Fatal("valid rational destroy requested allocation", e)
	}
	if f.constructed != f.destroyed || f.destroyed != 1 {
		t.Fatal("big rational destruction balance")
	}
	f.budget.FailAfter(-1)
	f.close(t)
	if _, e = f.q.String(&c.coeff); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("manager lifetime after pool destruction", e)
	}
}
