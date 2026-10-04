// Source: src/math/polynomial/linear_eq_solver.h; src/util/vector.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012.
package mpnum

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func linearDefaultPhysicalCensus(t *testing.T, q *RationalManager, s *RationalLinearSolver, baseline uint32) {
	t.Helper()
	blocks := map[memory.Block]bool{}
	vs := []*linearNumeralVector{&s.A, &s.b}
	for i := range s.rows {
		vs = append(vs, &s.rows[i])
	}
	for _, v := range vs {
		for _, b := range []memory.Block{v.raw, v.old, v.pending} {
			if b == (memory.Block{}) {
				continue
			}
			if blocks[b] {
				t.Fatal("duplicate named vector block", b)
			}
			blocks[b] = true
			if _, e := s.runtime.Size(b); e != nil {
				t.Fatal("named physical block", e)
			}
		}
	}
	if s.runtime.Stats().LiveBlocks != baseline+uint32(len(blocks)) {
		t.Fatal("unnamed physical block", s.runtime.Stats(), len(blocks), baseline)
	}
	want := q.headerBytes + integerActualOwnedBytes(q.z) + s.header + uint64(cap(s.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size())
	if q.budget.Used() != want {
		t.Fatal("complete actual storage census", q.budget.Used(), want)
	}
}

func TestLinearRationalResizeEveryPhysicalReservationPrefix(t *testing.T) {
	for fail := int64(0); fail < 100; fail++ {
		q, b := rationalManager(t, false)
		s, e := NewRationalLinearSolver(q)
		serviceMust(t, e)
		base := q.z.memory.Stats().LiveBlocks
		b.FailAfter(fail)
		e = s.Resize(5)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("real reservation category", fail, e)
		}
		// Snapshot every named block/header and actual cold byte before lifting
		// refusal. This campaign allocates defaults, not failed numeric setters.
		linearDefaultPhysicalCensus(t, q, s, base)
		closeErr := s.Close()
		if closeErr != nil {
			if !errors.Is(closeErr, ErrPolynomialAssertion) && !errors.Is(closeErr, arena.ErrBounds) && !errors.Is(closeErr, memory.ErrRuntimeUninitialized) {
				t.Fatal("source partial destruction", fail, closeErr)
			}
			serviceMust(t, s.Close())
		}
		if q.z.memory.Stats().LiveBlocks != base {
			t.Fatal("owning abort retained raw", fail)
		}
		if b.Used() != q.headerBytes+integerActualOwnedBytes(q.z) {
			t.Fatal("owning abort retained cold bytes", fail)
		}
		b.FailAfter(-1)
		if e == nil {
			t.Log("all prefix failures before success", fail)
			return
		}
	}
	t.Fatal("no successful actual prefix endpoint")
}

func TestLinearRationalIndependentFiniteTwoByTwo(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	rows := [2][2]Rational{{NewRational(), NewRational()}, {NewRational(), NewRational()}}
	rhs := [2]Rational{NewRational(), NewRational()}
	xs := [2]Rational{NewRational(), NewRational()}
	for a := int32(-2); a <= 2; a++ {
		if a == 0 {
			continue
		}
		for b := int32(-2); b <= 2; b++ {
			for c := int32(-2); c <= 2; c++ {
				for d := int32(-2); d <= 2; d++ {
					det := a*d - b*c
					if det == 0 {
						continue
					}
					for i, row := range [2][2]int32{{a, b}, {c, d}} {
						for j, v := range row {
							serviceMust(t, q.SetInt32(&rows[i][j], v))
						}
					}
					serviceMust(t, q.SetInt32(&rhs[0], -3))
					serviceMust(t, q.SetInt32(&rhs[1], 5))
					serviceMust(t, s.Add(0, rows[0][:], &rhs[0]))
					serviceMust(t, s.Add(1, rows[1][:], &rhs[1]))
					ok, e := s.Solve(xs[:])
					serviceMust(t, e)
					if !ok {
						t.Fatal("nonsingular independent determinant")
					}
					for i, num := range [2]int32{-3*d - 5*b, 5*a + 3*c} {
						want := NewRational()
						serviceMust(t, q.SetFractionSigned(&want, num, det))
						eq, e := q.Equal(&xs[i], &want)
						serviceMust(t, e)
						if !eq {
							t.Fatal("independent Cramer oracle", a, b, c, d, i)
						}
						serviceMust(t, q.Del(&want))
					}
				}
			}
		}
	}
	for i := range xs {
		serviceMust(t, q.Del(&xs[i]))
		serviceMust(t, q.Del(&rhs[i]))
		for j := range rows[i] {
			serviceMust(t, q.Del(&rows[i][j]))
		}
	}
	serviceMust(t, s.Close())
}

func TestLinearRationalSourceBackSubstitutionOutputGuard(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	one := NewRational()
	serviceMust(t, q.SetInt64(&one, 1))
	zero := NewRational()
	b0, b1 := NewRational(), NewRational()
	serviceMust(t, q.SetInt64(&b0, 5))
	serviceMust(t, q.SetInt64(&b1, 2))
	serviceMust(t, s.Add(0, []Rational{one, one}, &b0))
	serviceMust(t, s.Add(1, []Rational{zero, one}, &b1))
	xs := []Rational{NewRational()}
	serviceMust(t, q.SetInt64(&xs[0], 77))
	ok, e := s.Solve(xs)
	if ok || !errors.Is(e, arena.ErrBounds) {
		t.Fatal("descending reached result guard", ok, e)
	}
	want := NewRational()
	serviceMust(t, q.SetInt64(&want, 77))
	eq, e := q.Equal(&xs[0], &want)
	serviceMust(t, e)
	if !eq {
		t.Fatal("unreached lower output changed")
	}
	actual, e := s.RightHandSide(0)
	serviceMust(t, e)
	eq, e = q.Equal(&actual, &b0)
	serviceMust(t, e)
	if !eq {
		t.Fatal("back substitution ran before output publication")
	}
	for _, a := range []*Rational{&one, &zero, &b0, &b1, &xs[0], &want} {
		serviceMust(t, q.Del(a))
	}
	serviceMust(t, s.Close())
}

func TestLinearModularIndependentFiniteTwoByTwo(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := NewIntegerModularManager(q.z)
	serviceMust(t, e)
	serviceMust(t, m.SetZPUint64(7))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	serviceMust(t, s.Resize(2))
	// An independent bounded modular inverse oracle, without calling field
	// division or reproducing Gaussian elimination.
	for a := int64(1); a < 7; a++ {
		for b := int64(0); b < 7; b++ {
			for c := int64(0); c < 7; c++ {
				for d := int64(0); d < 7; d++ {
					det := ((a*d-b*c)%7 + 7) % 7
					if det == 0 {
						continue
					}
					inverse := int64(0)
					for k := int64(1); k < 7; k++ {
						if det*k%7 == 1 {
							inverse = k
							break
						}
					}
					rows := [2][2]Int{}
					rhs := [2]Int{}
					xs := [2]Int{}
					for i, row := range [2][2]int64{{a, b}, {c, d}} {
						for j, v := range row {
							serviceMust(t, m.SetInt64(&rows[i][j], v))
						}
					}
					serviceMust(t, m.SetInt64(&rhs[0], 3))
					serviceMust(t, m.SetInt64(&rhs[1], 5))
					serviceMust(t, s.Add(0, rows[0][:], &rhs[0]))
					serviceMust(t, s.Add(1, rows[1][:], &rhs[1]))
					ok, e := s.Solve(xs[:])
					serviceMust(t, e)
					if !ok {
						t.Fatal("independent invertible matrix", a, b, c, d)
					}
					for i, n := range [2]int64{3*d - 5*b, 5*a - 3*c} {
						want := Int{}
						serviceMust(t, m.SetInt64(&want, ((n*inverse)%7+7)%7))
						eq, e := m.Equal(&xs[i], &want)
						serviceMust(t, e)
						if !eq {
							t.Fatal("independent modular inverse oracle", a, b, c, d, i)
						}
						serviceMust(t, m.Del(&want))
						serviceMust(t, m.Del(&xs[i]))
						serviceMust(t, m.Del(&rhs[i]))
						for j := range rows[i] {
							serviceMust(t, m.Del(&rows[i][j]))
						}
					}
				}
			}
		}
	}
	serviceMust(t, s.Close())
	serviceMust(t, m.Close())
}

func TestLinearSourceNonNullZeroCapacityReachedAssertion(t *testing.T) {
	q, b := rationalManager(t, false)
	v := linearNumeralVector{width: 24}
	serviceMust(t, v.expand(q.z.memory))
	raw := v.raw
	serviceMust(t, q.z.memory.WriteUint32(raw, 0, 0))
	before := b.Used()
	b.FailAfter(0)
	e := v.expand(q.z.memory)
	b.FailAfter(-1)
	if modularSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) {
			t.Error("source capacity assertion before growth", e)
		}
	} else {
		var failure memory.Failure
		if !errors.As(e, &failure) || failure.Kind != memory.FailureVectorOverflow {
			t.Error("release source overflow", e)
		}
	}
	if v.raw != raw || v.pending != (memory.Block{}) || v.old != (memory.Block{}) || b.Used() != before {
		t.Error("guard preceded allocation/publication")
	}
	serviceMust(t, q.z.memory.WriteUint32(raw, 0, 2))
	storage := linearStorage{runtime: q.z.memory}
	serviceMust(t, storage.freeTokens(&v))
}

func TestLinearResizePostphysicalTLSRefusalRetainsUnpublishedVector(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	var prep memory.Block
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		if e := s.Close(); e != nil {
			serviceMust(t, s.Close())
		}
		serviceMust(t, rt.Deallocate(&prep))
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Error("actual complete owner refund", b.Used())
		}
	}()
	local := rt.State().LocalSize
	if local < 0 || local >= 99990 {
		t.Fatal("initial source lane", rt.State())
	}
	// A real caller-owned extent primes the actual TLS lane. No counter edits.
	prep, e = rt.Allocate(uint64(99990 - local - 8))
	serviceMust(t, e)
	if rt.State().LocalSize != 99990 {
		t.Fatal("real physical lane priming", rt.State())
	}
	base := rt.Stats().LiveBlocks
	count, e := rt.AllocationCount()
	serviceMust(t, e)
	localCount := rt.State().LocalCount
	serviceMust(t, rt.SetMaxSize(1))
	e = s.Resize(1)
	var refusal memory.Failure
	if !errors.As(e, &refusal) || refusal.Kind != memory.FailureMemout || !rt.State().OutOfMemory {
		t.Fatal("reached TLS malloc refusal", e, rt.State())
	}
	if s.A.raw != (memory.Block{}) || s.A.pending == (memory.Block{}) || s.n != 1 || len(s.rows) != 1 {
		t.Fatal("actual unpublished malloc root", s.A, s.n, len(s.rows))
	}
	size, e := rt.Size(s.A.pending)
	serviceMust(t, e)
	afterCount, e := rt.AllocationCount()
	serviceMust(t, e)
	if size != 16 || afterCount != count+uint64(localCount)+1 || rt.Stats().LiveBlocks != base+1 {
		t.Fatal("exact actual allocated extent", size, afterCount, count, rt.Stats())
	}
	linearDefaultPhysicalCensus(t, q, s, base)
	pending := s.A.pending
	// Literal completed solver/member destruction cannot free the vector
	// helper's unreturned allocation. Only the explicit owning adapter owns it.
	serviceMust(t, s.SourceFinalize())
	if s.A.pending != pending || s.A.raw != (memory.Block{}) || rt.Stats().LiveBlocks != base+1 {
		t.Fatal("source destructor consumed unpublished helper root")
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, s.Close())
	if rt.Stats().LiveBlocks != base {
		t.Fatal("owning abort did not consume actual pending extent")
	}
}

// All arithmetic roots come from actual manager fields, physical row/RHS
// fields, and explicit caller numerals. Locator tables provide no owning roots.
func linearArithmeticPhysicalCensus(t *testing.T, q *RationalManager, s *RationalLinearSolver, callers ...*Rational) {
	t.Helper()
	values := []Int{q.nTmp, q.addTmp1, q.addTmp2, q.addMulTmp.num, q.addMulTmp.den, q.ltTmp1.num, q.ltTmp1.den, q.ltTmp2.num, q.ltTmp2.den}
	for _, a := range callers {
		values = append(values, a.num, a.den)
	}
	var blocks []memory.Block
	vs := []*linearNumeralVector{&s.A, &s.b}
	for i := range s.rows {
		vs = append(vs, &s.rows[i])
	}
	for _, v := range vs {
		if v.old != (memory.Block{}) || v.pending != (memory.Block{}) {
			t.Fatal("prepared arithmetic fixture acquired unreturned vector storage")
		}
		if v.raw != (memory.Block{}) {
			blocks = append(blocks, v.raw)
		}
		if v.width == 24 && v.raw != (memory.Block{}) {
			for i := uint32(0); i < v.initialized; i++ {
				a, e := v.rationalAt(s.runtime, i)
				serviceMust(t, e)
				values = append(values, a.num, a.den)
			}
		}
	}
	refs := make([]*Int, len(values))
	for i := range values {
		refs[i] = &values[i]
	}
	integerPhysicalAuditWithRaw(t, q.z, blocks, refs...)
	want := q.headerBytes + integerActualOwnedBytes(q.z) + s.header + uint64(cap(s.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size())
	if q.budget.Used() != want {
		t.Fatal("complete arithmetic physical/cold census", q.budget.Used(), want)
	}
}

func TestLinearRationalArithmeticEveryReservationPrefix(t *testing.T) {
	for _, solve := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 512; ordinal++ {
			q, b := rationalPublicationDirectOwner(t, false)
			s, e := NewRationalLinearSolver(q)
			serviceMust(t, e)
			serviceMust(t, s.Resize(2))
			input := [6]Rational{NewRational(), NewRational(), NewRational(), NewRational(), NewRational(), NewRational()}
			for i, text := range []string{"79228162514264337593543950337", "1", "1", "79228162514264337593543950338", "158456325028528675187087900677", "237684487542793012780631851016"} {
				serviceMust(t, q.SetString(&input[i], text))
			}
			outputs := [2]Rational{NewRational(), NewRational()}
			if solve {
				serviceMust(t, s.Add(0, input[:2], &input[4]))
				serviceMust(t, s.Add(1, input[2:4], &input[5]))
			}
			b.FailAfter(ordinal)
			if solve {
				var ok bool
				ok, e = s.Solve(outputs[:])
				if e == nil && !ok {
					t.Fatal("source nonzero pivots", ordinal)
				}
			} else {
				e = s.Add(0, input[:2], &input[4])
			}
			roots := []*Rational{&input[0], &input[1], &input[2], &input[3], &input[4], &input[5], &outputs[0], &outputs[1]}
			linearArithmeticPhysicalCensus(t, q, s, roots...)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual arithmetic reservation cause", solve, ordinal, e)
				}
				failures++
			} else {
				success = true
				if solve {
					for i, v := range []int64{2, 3} {
						want := NewRational()
						serviceMust(t, q.SetInt64(&want, v))
						equal, e := q.Equal(&outputs[i], &want)
						serviceMust(t, e)
						if !equal {
							t.Fatal("independent chosen solution", ordinal, i)
						}
						serviceMust(t, q.Del(&want))
					}
				}
			}
			b.FailAfter(-1)
			if closeErr := s.Close(); closeErr != nil {
				serviceMust(t, s.Close())
			}
			for _, a := range roots {
				serviceMust(t, q.Del(a))
			}
			serviceMust(t, q.CloseChecked())
			if b.Used() != 0 {
				t.Fatal("complete arithmetic owning refund", solve, ordinal, b.Used())
			}
			if success {
				t.Log("every actual refusal before success", solve, failures, ordinal)
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("bounded complete real arithmetic campaign", solve, success, failures)
		}
	}
}

// Explicit source-undefined fixture: caller physically truncates the old RHS
// through Runtime, leaving NUM readable and DEN out of bounds. This is a
// reached adapter diagnostic, not exhaustion or a valid C++ destructor case.
func TestLinearStoppedNumeratorMoveRetainsActualOwner(t *testing.T) {
	q, _ := rationalPublicationDirectOwner(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	serviceMust(t, s.Resize(1))
	rhs := NewRational()
	serviceMust(t, q.SetString(&rhs, "79228162514264337593543950337"))
	serviceMust(t, s.Add(0, []Rational{NewRational()}, &rhs))
	before, e := s.RightHandSide(0)
	serviceMust(t, e)
	// A recorded projection permits fixture cleanup on a failed test; no locator
	// or reference count supplies an owning root to production recovery.
	observed := before.num
	defer func() {
		if !s.closed {
			if e := s.Close(); e != nil {
				_ = s.Close()
			}
		}
		if validNumericRef(q.z.cells, observed.cell) {
			serviceMust(t, q.z.Del(&observed))
		}
		serviceMust(t, q.Del(&rhs))
	}()
	serviceMust(t, s.runtime.Reallocate(&s.b.raw, 20))
	e = s.b.expand(s.runtime)
	if !errors.Is(e, arena.ErrBounds) || s.b.raw == (memory.Block{}) || s.b.old == (memory.Block{}) || s.b.initialized != 0 || s.b.moved != 0 {
		t.Fatal("actual NUM move before failed DEN read", e, s.b)
	}
	moved, e := linearReadInt(s.runtime, s.b.raw, 8)
	serviceMust(t, e)
	old, e := linearReadInt(s.runtime, s.b.old, 8)
	serviceMust(t, e)
	if moved != before.num || old.cell != 0 {
		t.Fatal("physical owner split", moved, old)
	}
	values := []Int{q.nTmp, q.addTmp1, q.addTmp2, q.addMulTmp.num, q.addMulTmp.den, q.ltTmp1.num, q.ltTmp1.den, q.ltTmp2.num, q.ltTmp2.den, rhs.num, rhs.den, moved}
	refs := make([]*Int, len(values))
	for i := range values {
		refs[i] = &values[i]
	}
	row, e := s.row(0)
	serviceMust(t, e)
	integerPhysicalAuditWithRaw(t, q.z, []memory.Block{s.A.raw, row.raw, s.b.raw, s.b.old}, refs...)
	want := q.headerBytes + integerActualOwnedBytes(q.z) + s.header + uint64(cap(s.rows))*uint64(reflect.TypeOf(linearNumeralVector{}).Size())
	if q.budget.Used() != want {
		t.Fatal("immediate partial physical/cold graph", q.budget.Used(), want)
	}
	// Restore only the caller's malformed DEN span, using real Runtime resize and
	// writes. The source failure prefix above is already independently audited.
	serviceMust(t, s.runtime.Reallocate(&s.b.old, 32))
	serviceMust(t, linearWriteInt(s.runtime, s.b.old, 20, NewInt(1)))
	first := s.Close()
	if !errors.Is(first, arena.ErrBounds) {
		t.Fatal("literal partial member guard", first)
	}
	serviceMust(t, s.Close())
	if validNumericRef(q.z.cells, observed.cell) {
		t.Fatal("owning abort lost the actually moved numerator")
	}
}
