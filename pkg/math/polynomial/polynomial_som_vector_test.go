// Source: src/math/polynomial/polynomial.cpp:1885-2214,3875,6865.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSOMVectorDefaultOwnerAndSourceMembers(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	b, e := v.At(0)
	serviceMust(t, e)
	empty, e := b.Empty()
	serviceMust(t, e)
	if !empty {
		t.Fatal("actual default buffer")
	}
	serviceMust(t, b.Reset()) // source early empty return does not read owner.
	serviceMust(t, v.SetOwner(m))
	x, e := m.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	a := util.NewInt(3)
	serviceMust(t, b.Add(&a, x))
	a = util.NewInt(4)
	serviceMust(t, b.Add(&a, x))
	n, e := b.Size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("real SOM duplicate accumulation", n)
	}
	value, e := b.Coefficient(0)
	serviceMust(t, e)
	value32, e := m.integer.Int64(&value)
	serviceMust(t, e)
	if value32 != 7 {
		t.Fatal("actual modular coefficient", value32)
	}
	// Implicit som_buffer member destructors free pointer/numeral/position
	// raw backing only. They must not perform Reset or release leaked INCs.
	info, e := m.monomials.headerCore(x)
	serviceMust(t, e)
	before := info.RefCount
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	info, e = m.monomials.headerCore(x)
	serviceMust(t, e)
	after := info.RefCount
	if before != after || after != 1 {
		t.Fatal("implicit source child dtor invented DEC", before, after)
	}
	if !b.sourceFinalized || b.closed || b.retiredMonomialCount != 1 || b.retiredNumeralCount != 1 {
		t.Fatal("actual owned fields retained before source raw free")
	}
	if _, e = v.At(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source finalized vector entry", e)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
	polynomialOperationMonomialAudit(t, m, nil, nil)
}
func TestPolynomialSOMVectorActualColdChildRefusal(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	m.integer.Budget().FailAfter(0)
	_, e = v.At(0)
	if !errors.Is(e, arena.ErrInjectedFailure) || v.pending != nil {
		t.Fatal("real Go class header refusal before source payload", e)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	m.integer.Budget().FailAfter(-1)
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMVectorAtEveryReservation(t *testing.T) {
	success := false
	failures := 0
	incoming := false
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			serviceMust(t, v.SetOwner(m))
			m.integer.Budget().FailAfter(ordinal)
			b, e := v.At(2)
			if e == nil {
				success = true
				if b == nil || len(v.children) != 3 {
					t.Fatal("actual lazy capacity")
				}
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual header/raw refusal", e)
				}
				if v.pending != nil {
					incoming = true
					if v.pending.sourceStarted || v.pending.closed {
						t.Fatal("incoming successfully constructed child prematurely deleted")
					}
				}
			}
			polynomialSOMPhysicalAudit(t, m, v, nil)
			// Destructor body visits published pointers only, preserving unpushed
			// allocated local child and unpublished initial raw allocation.
			pending := v.pending
			serviceMust(t, v.SourceFinalize())
			if pending != nil && (v.pending != pending || pending.sourceStarted || pending.closed) {
				t.Fatal("source dtor swept incoming push local")
			}
			polynomialSOMPhysicalAudit(t, m, v, nil)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, v.CloseChecked())
			polynomialPhysicalAudit(t, m)
		})
	}
	if !success || failures == 0 || !incoming {
		t.Fatal("bounded real class/push campaign", success, failures, incoming)
	}
}
func TestPolynomialSOMBufferAddEveryReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			serviceMust(t, v.SetOwner(m))
			b, e := v.At(0)
			serviceMust(t, e)
			x, e := m.monomials.MkVariable(0, 2)
			serviceMust(t, e)
			var a util.Int
			serviceMust(t, m.integer.SetString(&a, "123456789012345678901234567890123456789012345678901234567890"))
			m.integer.Budget().FailAfter(ordinal)
			e = b.Add(&a, x)
			if e == nil {
				success = true
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual cold/raw/numeric failure", e)
				}
			}
			n, ne := b.buffer.ms.size(m)
			serviceMust(t, ne)
			// Authority is the actual completed INC publication, not RC inference.
			joined := n != 0 || b.buffer.hasPendingReference
			polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
			serviceMust(t, v.SourceFinalize())
			polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, v.CloseChecked())
			if !joined {
				serviceMust(t, m.monomials.CleanupValue(&x))
			}
			serviceMust(t, m.integer.CleanupPendingAllocation())
			serviceMust(t, m.integer.Del(&a))
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, nil, nil)
		})
	}
	if !success || failures == 0 {
		t.Fatal("bounded true SOM add campaign", success, failures)
	}
}

func TestPolynomialSOMVectorActualModularAndResetPrefix(t *testing.T) {
	m := polynomialCellFixture(t)
	serviceMust(t, m.modular.SetZPUint64(17))
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b0, e := v.At(0)
	serviceMust(t, e)
	b1, e := v.At(1)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	a := util.NewInt(6)
	serviceMust(t, b0.Add(&a, x))
	a = util.NewInt(7)
	serviceMust(t, b0.Add(&a, x))
	value, e := b0.Coefficient(0)
	serviceMust(t, e)
	got, e := m.integer.Int64(&value)
	serviceMust(t, e)
	if got != -4 {
		t.Fatal("actual balanced Z17 manager accumulation", got)
	}
	a = util.NewInt(3)
	serviceMust(t, b1.Add(&a, x))
	serviceMust(t, v.ResetN(1))
	n, e := b0.Size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("source clamped reset first buffer", n)
	}
	n, e = b1.Size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("later actual buffer changed", n)
	}
	h, e := m.monomials.headerCore(x)
	serviceMust(t, e)
	if h.RefCount != 1 {
		t.Fatal("source exactly one completed DEC", h.RefCount)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.ResetN(100))
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}
func TestPolynomialSOMBufferZeroAndDuplicateNoColdReservation(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	a := util.NewInt(2)
	serviceMust(t, b.Add(&a, x))
	before := m.integer.Budget().Used()
	m.integer.Budget().FailAfter(0)
	zero := util.NewInt(0)
	serviceMust(t, b.Add(&zero, x))
	a = util.NewInt(3)
	serviceMust(t, b.Add(&a, x))
	if m.integer.Budget().Used() != before {
		t.Fatal("early/duplicate branches reserved new cold ownership")
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferAddMulIndependentFiniteModel(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	p, e := m.MkTerms([]util.Int{util.NewInt(2), util.NewInt(3)}, []Monomial{x, u})
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	factor := util.NewInt(4)
	serviceMust(t, b.AddMul(&factor, x, p)) // 8*x^2 + 12*x
	serviceMust(t, b.AddMul(&factor, x, p)) // duplicate branch => 16*x^2 + 24*x
	for value := int64(-3); value <= 3; value++ {
		var got int64
		n, e := b.Size()
		serviceMust(t, e)
		if n != 2 {
			t.Fatal("actual unique monomial accumulation", n)
		}
		for i := uint32(0); i < n; i++ {
			c, e := b.Coefficient(i)
			serviceMust(t, e)
			coefficient, e := m.integer.Int64(&c)
			serviceMust(t, e)
			term, e := b.Monomial(i)
			serviceMust(t, e)
			info, e := m.monomials.Info(term)
			serviceMust(t, e)
			var power int64 = 1
			for j := uint32(0); j < info.Size; j++ {
				q, e := m.monomials.ReadPower(term, j)
				serviceMust(t, e)
				for k := uint32(0); k < q.Degree; k++ {
					power *= value
				}
			}
			got += coefficient * power
		}
		if got != 16*value*value+24*value {
			t.Fatal("independent source addmul model", value, got)
		}
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferAddMulEveryReservation(t *testing.T) {
	success, failures := false, 0
	sawGenerated, sawCallee := false, false
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			serviceMust(t, v.SetOwner(m))
			b, e := v.At(0)
			serviceMust(t, e)
			x, e := m.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := m.MkTerms([]util.Int{util.NewInt(3)}, []Monomial{x})
			serviceMust(t, e)
			polynomialCaller(t, m, p)
			var a util.Int
			serviceMust(t, m.integer.SetString(&a, "123456789012345678901234567890123456789012345678901234567890"))
			m.integer.Budget().FailAfter(ordinal)
			e = b.AddMul(&a, x, p)
			if e == nil {
				success = true
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual addmul prefix category", e)
				}
			}
			if b.generatedCall {
				sawGenerated = true
			}
			if b.calleeCall {
				sawCallee = true
			}
			polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
			generated, callee := m.generatedActive, m.generatedCalleePending
			serviceMust(t, v.SourceFinalize())
			if m.generatedActive != generated || m.generatedCalleePending != callee {
				t.Fatal("implicit member dtor aborted source local/callee")
			}
			polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, v.CloseChecked())
			serviceMust(t, m.integer.CleanupPendingAllocation())
			serviceMust(t, m.integer.Del(&a))
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, []Polynomial{p}, nil)
		})
	}
	if !success || failures == 0 || !sawGenerated || !sawCallee {
		t.Fatal("real product/local/INC prefix campaign", success, failures, sawGenerated, sawCallee)
	}
}

func TestPolynomialSOMBufferActualOverloadsAndUnreadZero(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	source, e := v.At(0)
	serviceMust(t, e)
	out, e := v.At(1)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	a := util.NewInt(2)
	serviceMust(t, source.Add(&a, x))
	a = util.NewInt(3)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	serviceMust(t, source.Add(&a, u))
	zero := util.NewInt(0)
	serviceMust(t, out.AddMulBuffer(&zero, x, nil)) // zero precedes source pointer read.
	factor := util.NewInt(4)
	serviceMust(t, out.AddMulBuffer(&factor, x, source))
	one := util.NewInt(1)
	serviceMust(t, out.AddScaledBuffer(&one, source)) // 8*x^2+14*x+3
	result, e := out.Mk(false)
	serviceMust(t, e)
	polynomialCaller(t, m, result)
	n, e := m.Info(result)
	serviceMust(t, e)
	if n.Size != 3 {
		t.Fatal("actual overload result", n)
	}
	serviceMust(t, out.AddPolynomial(result))
	for val := int64(-2); val <= 2; val++ {
		var got int64
		n, e := out.Size()
		serviceMust(t, e)
		for i := uint32(0); i < n; i++ {
			a, e := out.Coefficient(i)
			serviceMust(t, e)
			c, e := m.integer.Int64(&a)
			serviceMust(t, e)
			t0, e := out.Monomial(i)
			serviceMust(t, e)
			info, e := m.monomials.Info(t0)
			serviceMust(t, e)
			pow := int64(1)
			for j := uint32(0); j < info.TotalDegree; j++ {
				pow *= val
			}
			got += c * pow
		}
		if got != 8*val*val+14*val+3 {
			t.Fatal("independent overload oracle", val, got)
		}
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferSourceMaxAndDisplay(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	a := util.NewInt(3)
	serviceMust(t, b.Add(&a, x))
	a = util.NewInt(-3)
	serviceMust(t, b.Add(&a, x))
	// Zero entries remain real fields until remove_zeros; max ignores them.
	max, e := b.GradedLexMaxPosition()
	serviceMust(t, e)
	if max != ^uint32(0) {
		t.Fatal("source all-zero max", max)
	}
	a = util.NewInt(7)
	serviceMust(t, b.Add(&a, x))
	max, e = b.GradedLexMaxPosition()
	serviceMust(t, e)
	if max != 0 {
		t.Fatal("source max position", max)
	}
	sink := &monomialTestSink{}
	serviceMust(t, b.DisplayInto(sink))
	if sink.text.String() != "7*x0\n" {
		t.Fatal("source SOM display", sink.text)
	}
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

// Public At creates actual empty SOM objects and pointer-vector storage.
// The public TLS recipe reaches signed-host/source undefined arithmetic;
// this is stopped postphysical sync evidence, not ordinary OOM/free refusal.
func TestPolynomialSOMVectorSourceFinalizeStickyPostPhysicalSuffix(t *testing.T) {
	const lastChild = 13000
	// This fixture targets the later pointer-vector TLS release, rather than
	// setup OOM. Fund the actual reflected child headers independently of the
	// setup headroom so added cold invocation fields do not prevent that free.
	childBytes := uint64(lastChild+1) * uint64(reflect.TypeOf(PolynomialSOMBuffer{}).Size())
	b := arena.NewBudget(childBytes + 1<<27)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: r, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	_, e = v.At(lastChild)
	serviceMust(t, e)
	// Completed empty SOM children remain source members until the vector dtor.
	cap, e := v.data.capacity(m)
	serviceMust(t, e)
	if uint64(cap)*8+8 < 100000 {
		t.Fatal("real pointer vector insufficient for TLS sync", cap)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	// Establish real ordinary ID freelist capacity before entering the guarded
	// signed-overflow domain. This isolates the reached vector-member free;
	// unprimed ID allocation-before-free refusal is separate evidence.
	var constants [6]Polynomial
	for i := range constants {
		a := util.NewInt(int32(i + 2))
		constants[i], e = m.MkConstantTake(&a)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(constants[i]))
	}
	for _, p := range constants {
		serviceMust(t, m.DecRef(p))
	}
	idSize, e := m.ids.Size()
	serviceMust(t, e)
	idCap, e := m.ids.Capacity()
	serviceMust(t, e)
	if idSize != 6 || idCap < idSize+2 {
		t.Fatal("actual polynomial freelist room", idSize, idCap)
	}
	_, e = mm.MkVar()
	serviceMust(t, e)
	mon, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, mm.IncRef(mon))
	serviceMust(t, mm.DecRef(mon))
	monoSize, e := mm.ids.Size()
	serviceMust(t, e)
	monoCap, e := mm.ids.Capacity()
	serviceMust(t, e)
	if monoSize != 1 || monoCap < 2 {
		t.Fatal("actual monomial freelist room", monoSize, monoCap)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	contextTLSNearMinimum(t, r)
	before := r.Stats().LiveBlocks
	e = v.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || v.data.raw != (util.Block{}) || v.sourceStage != 1 || v.sourceFinalized || r.Stats().LiveBlocks+1 != before {
		t.Fatal("actual suffix physical release before stopped sync", e, v.sourceStage, r.State())
	}
	cause := v.sourceCause
	polynomialSOMPhysicalAudit(t, m, v, nil)
	used := b.Used()
	count, e := r.AllocationCount()
	serviceMust(t, e)
	if e = v.SourceFinalize(); e != cause || b.Used() != used {
		t.Fatal("literal source retry acknowledged owning release", e, cause)
	}
	afterCount, e := r.AllocationCount()
	serviceMust(t, e)
	if afterCount != count {
		t.Fatal("retry fabricated source event")
	}
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("source-stopped ordinary size", e)
	}
	serviceMust(t, v.CloseChecked())
	if v.sourceFinalized || v.sourceCause != cause || v.sourceStage != 1 {
		t.Fatal("owning abort repaired source history")
	}
	contextTLSFinish(t, r, m.CloseChecked)
	// The source reference decrement executes once. Its stopped destructor
	// continuation belongs to explicit owning Close, never another DEC.
	beforeMM := r.Stats().LiveBlocks
	e = mm.DecManagerRef()
	if e != nil {
		if !errors.Is(e, arena.ErrBounds) || mm.managerRefs != 0 || r.Stats().LiveBlocks >= beforeMM {
			t.Fatal("source MM decrement/destructor prefix", e, mm.managerRefs, r.State())
		}
		contextTLSFinish(t, r, mm.CloseChecked)
	}
	contextTLSFinish(t, r, q.CloseChecked)
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual owning graph retained", b.Used())
	}
}

func TestPolynomialSOMBufferSourceSortMovesCoefficients(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	cube, e := m.monomials.MkVariable(0, 3)
	serviceMust(t, e)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	for i, t0 := range []Monomial{x, u, cube} {
		a := util.NewInt([]int32{5, 7, 11}[i])
		serviceMust(t, b.Add(&a, t0))
	}
	serviceMust(t, b.SortGradedLex())
	for i, want := range []int64{11, 5, 7} {
		a, e := b.Coefficient(uint32(i))
		serviceMust(t, e)
		got, e := m.integer.Int64(&a)
		serviceMust(t, e)
		if got != want {
			t.Fatal("source sorted coefficient move", i, got, want)
		}
	}
	serviceMust(t, v.Reset())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferSourceSortEveryReservation(t *testing.T) {
	success, failures := false, 0
	sawMoved := false
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			serviceMust(t, v.SetOwner(m))
			b, e := v.At(0)
			serviceMust(t, e)
			var input util.Int
			serviceMust(t, m.integer.SetString(&input, "12345678901234567890123456789012345678901234567890"))
			for _, degree := range []uint32{1, 0, 4, 2} {
				var term Monomial
				if degree == 0 {
					term, e = m.monomials.Unit()
				} else {
					term, e = m.monomials.MkVariable(0, degree)
				}
				serviceMust(t, e)
				serviceMust(t, b.Add(&input, term))
			}
			m.integer.Budget().FailAfter(ordinal)
			e = b.SortGradedLex()
			if e == nil {
				success = true
			} else {
				failures++
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual lexical source vector refusal", e)
				}
			}
			if b.retiredNumeralCount > 0 {
				sawMoved = true
			}
			polynomialSOMPhysicalAudit(t, m, v, nil, &input)
			serviceMust(t, v.SourceFinalize())
			polynomialSOMPhysicalAudit(t, m, v, nil, &input)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, v.CloseChecked())
			serviceMust(t, m.integer.CleanupPendingAllocation())
			serviceMust(t, m.integer.Del(&input))
			polynomialPhysicalAudit(t, m)
			polynomialOperationMonomialAudit(t, m, nil, nil)
		})
	}
	if !success || failures == 0 || !sawMoved {
		t.Fatal("bounded actual sort move/source suffix sweep", success, failures, sawMoved)
	}
}

func TestPolynomialSOMBufferActualModDAndScalarOverloads(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	square, e := m.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	a := util.NewInt(5)
	serviceMust(t, b.Add(&a, x))
	a = util.NewInt(7)
	serviceMust(t, b.Add(&a, square))
	a = util.NewInt(11)
	serviceMust(t, b.AddScalar(&a))
	d, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	serviceMust(t, d.SetDegree(0, 2))
	serviceMust(t, b.ModD(d))
	for i, want := range []int64{5, 0, 11} {
		a, e := b.Coefficient(uint32(i))
		serviceMust(t, e)
		got, e := m.integer.Int64(&a)
		serviceMust(t, e)
		if got != want {
			t.Fatal("literal coefficient filter retains vector size", i, got, want)
		}
	}
	serviceMust(t, d.CloseChecked())
	out, e := v.At(1)
	serviceMust(t, e)
	serviceMust(t, out.AddMonomialBuffer(x, b))
	// CheckZeros=false really processes a retained zero coefficient, including
	// monomial creation/INC, rather than silently compacting the input.
	n, e := out.Size()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("source untrimmed buffer route", n)
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferNeverAbortsBorrowedCalleeCreation(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	m.integer.Budget().FailAfter(0)
	_, e = m.monomials.MkVariable(0, 23) // This invocation belongs directly to the borrowed MM caller.
	if !numericPhysicalReservationRefusal(e) || m.monomials.creation.stage == 0 {
		t.Fatal("real borrowed callee construction refusal", e, m.monomials.creation.stage)
	}
	before := m.monomials.creation
	zero := util.NewInt(0)
	serviceMust(t, b.AddMul(&zero, Monomial{}, Polynomial{}))
	if b.generatedCall || b.calleeCall || m.generatedCalleePending {
		t.Fatal("zero branch discovered foreign pending ownership")
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if m.monomials.creation != before {
		t.Fatal("class owning close swept borrowed callee")
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, m.monomials.CleanupCreation())
	polynomialPhysicalAudit(t, m)
	polynomialOperationMonomialAudit(t, m, nil, nil)
}

func TestPolynomialSOMBufferSourceSelfAliasReadsChangedFutureCoefficient(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	square, e := m.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	for i, term := range []Monomial{x, u, square} {
		a := util.NewInt([]int32{2, 1, 7}[i])
		serviceMust(t, b.Add(&a, term))
	}
	factor := util.NewInt(3)
	serviceMust(t, b.AddMulBuffer(&factor, x, b))
	// Source captures original size3, but each p->a(i) reads the live buffer.
	// First updates future x^2 coefficient7->13; last multiplies13, not7.
	n, e := b.Size()
	serviceMust(t, e)
	if n != 4 {
		t.Fatal("source alias final size", n)
	}
	for i, want := range []int64{5, 1, 13, 39} {
		a, e := b.Coefficient(uint32(i))
		serviceMust(t, e)
		got, e := m.integer.Int64(&a)
		serviceMust(t, e)
		if got != want {
			t.Fatal("source reached alias field", i, got, want)
		}
	}
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.SourceFinalize())
	polynomialSOMPhysicalAudit(t, m, v, nil)
	serviceMust(t, v.CloseChecked())
	polynomialPhysicalAudit(t, m)
}

func TestPolynomialSOMBufferStoppedSortRejectsOrdinaryMutation(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Profile: profile, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: rt, Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("SOM checkpoint"); return false }, Cooperation: func() error { t.Fatal("SOM cooperation"); return nil }})
	serviceMust(t, e)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	x, e := mm.MkVariable(0, 1)
	serviceMust(t, e)
	two := util.NewInt(2)
	serviceMust(t, b.Add(&two, x))
	var prep util.Block
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, v.CloseChecked())
		serviceMust(t, rt.Deallocate(&prep))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, mm.DecManagerRef())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("actual retained owner cleanup", budget.Used())
		}
	}()
	local := rt.State().LocalSize
	if local < 0 || local >= 99972 {
		t.Fatal("actual setup lane", rt.State())
	}
	prep, e = rt.Allocate(uint64(99980 - local - 8))
	serviceMust(t, e)
	before := rt.Stats()
	serviceMust(t, rt.SetMaxSize(1))
	e = b.SortGradedLex()
	var refusal util.Failure
	if !errors.As(e, &refusal) || refusal.Kind != util.FailureMemout || !b.sortActive || b.sortNumerals.pending == (util.Block{}) || b.sortNumerals.raw != (util.Block{}) || rt.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("actual returned default-local allocation", e, rt.State())
	}
	polynomialSOMPhysicalWithExternal(t, m, v, nil, []util.Block{prep}) // BEFORE lifting actual limit.
	oldMS, oldAS := b.buffer.ms.raw, b.buffer.as.raw
	one := util.NewInt(1)
	if e = b.Add(&one, x); e != ErrPolynomialPending {
		t.Fatal("stopped lexical context allowed ordinary mutation", e)
	}
	if b.buffer.ms.raw != oldMS || b.buffer.as.raw != oldAS {
		t.Fatal("owning guard modified original raw fields")
	}
	// Identity-only physical reads verify the actual old coefficient, without
	// going through the ordinary stopped-owner API.
	value, e := b.buffer.as.coefficient(m, 0)
	serviceMust(t, e)
	got, e := m.integer.Int64(&value)
	serviceMust(t, e)
	if got != 2 {
		t.Fatal("guard came after mutation", got)
	}
	if e = b.Reset(); e != ErrPolynomialPending {
		t.Fatal("stopped reset", e)
	}
	if _, e = b.Mk(false); e != ErrPolynomialPending {
		t.Fatal("stopped construction", e)
	}
	if e = b.SortGradedLex(); e != ErrPolynomialPending {
		t.Fatal("stopped scope reuse", e)
	}
	peer, e := v.At(1)
	serviceMust(t, e)
	zero := util.NewInt(0)
	serviceMust(t, peer.Add(&zero, x)) // another actual owner is not poisoned.
	polynomialSOMPhysicalWithExternal(t, m, v, nil, []util.Block{prep})
}
