package euclid

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanEquationConstructionSortAndOwnedRelease(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(5))
	euclidMust(t, rb.Resize(4))
	as := []numeral.Int{{}, {}}
	euclidMust(t, z.SetString(&as[0], "123456789012345678901"))
	euclidMust(t, z.SetString(&as[1], "-7"))
	c := numeral.Int{}
	euclidMust(t, z.SetString(&c, "987654321098765432109"))
	bs := []numeral.Rational{numeral.NewRational(), numeral.NewRational()}
	euclidMust(t, m.SetString(&bs[0], "23/11"))
	euclidMust(t, m.SetString(&bs[1], "-5/13"))
	eq, e := makeEquation(m, as, []uint32{4, 1}, &c, bs, []uint32{3, 0}, ib, rb, true, nil)
	euclidMust(t, e)
	if len(eq.as) != 2 || len(eq.bs) != 2 || eq.xs[0] != 1 || eq.xs[1] != 4 || eq.js[0] != 0 || eq.js[1] != 3 {
		t.Fatal("equation source scalar order", eq.xs, eq.js)
	}
	for i, want := range []string{"-7", "123456789012345678901"} {
		got, e := z.String(&eq.as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("equation integer coefficient", i, got, want)
		}
	}
	for i, want := range []string{"-5/13", "23/11"} {
		got, e := m.String(&eq.bs[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("equation rational justification", i, got, want)
		}
	}
	if got, e := z.String(&eq.c); e != nil || got != "987654321098765432109" {
		t.Fatal("equation constant copy", got, e)
	}
	// Closing the equation releases its numerals; the reusable buffers and
	// original caller numerals keep their separate ownership.
	euclidMust(t, eq.Close())
	euclidMust(t, eq.Close())
	for i, want := range []string{"123456789012345678901", "-7"} {
		got, e := z.String(&as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("caller integer changed", i, got)
		}
	}
	for i, want := range []string{"23/11", "-5/13"} {
		got, e := m.String(&bs[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("caller rational changed", i, got)
		}
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	for i := range bs {
		euclidMust(t, m.Del(&bs[i]))
	}
	euclidMust(t, z.Del(&c))
}

func TestEuclideanEquationUnsortedDecompositionBranch(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(3))
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(-1)}
	c := numeral.NewInt(0)
	eq, e := makeEquation(m, as, []uint32{2, 0}, &c, nil, nil, ib, rb, false, nil)
	euclidMust(t, e)
	if len(eq.bs) != 0 || eq.xs[0] != 0 || eq.xs[1] != 2 || eq.as[0].Val != -1 || eq.as[1].Val != 2 {
		t.Fatal("decomposition equation sort/justification", eq.xs, eq.as, eq.bs)
	}
	euclidMust(t, eq.Close())
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
}

func TestEuclideanEquationHeaderRefusalPrecedesBufferMutation(t *testing.T) {
	m := euclidNumeralManager(t)
	ib, e := numeral.NewScopedIntVector(m.Integers())
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	a := []numeral.Int{numeral.NewInt(9)}
	c := numeral.NewInt(2)
	eq, e := makeEquation(m, a, []uint32{0}, &c, nil, nil, ib, rb, true, nil)
	if e == nil || eq != nil {
		t.Fatal("equation header refusal", eq, e)
	}
	m.BudgetField.FailAfter(-1)
	slot, e := ib.BorrowSlot(0)
	euclidMust(t, e)
	if slot.Val != 0 || a[0].Val != 9 || c.Val != 2 || m.BudgetField.Used() != before {
		t.Fatal("header refusal changed input or buffer", slot, a, c)
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
}

func TestEuclideanEquationIntegerCopyRefusalKeepsBufferOwner(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	a := numeral.NewInt(0)
	value := strings.Repeat("1234567890", 120)
	euclidMust(t, z.SetString(&a, value))
	c := numeral.NewInt(11)
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(1) // equation header succeeds; first big coefficient copy fails.
	eq, e := makeEquation(m, []numeral.Int{a}, []uint32{0}, &c, nil, nil, ib, rb, true, nil)
	m.BudgetField.FailAfter(-1)
	var failure umemory.Failure
	if eq != nil || !errors.As(e, &failure) || failure.Kind != umemory.FailureMemout {
		t.Fatal("source integer copy refusal", eq, e)
	}
	slot, e := ib.BorrowSlot(0)
	euclidMust(t, e)
	if !z.IsFailureRef(slot.Cell) {
		t.Fatal("failed buffer numeral ownership", slot)
	}
	euclidMust(t, ib.Cleanup())
	euclidMust(t, rb.Close())
	if m.BudgetField.Used() != before {
		t.Fatal("failed equation retained header or numeral", before, m.BudgetField.Used())
	}
	got, e := z.String(&a)
	euclidMust(t, e)
	if got != value || c.Val != 11 {
		t.Fatal("borrowed input changed", got, c)
	}
	euclidMust(t, z.Del(&a))
}

func TestEuclideanEquationFailedConstantCopyReleasesConstructedPrefix(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	var a numeral.Int
	euclidMust(t, z.SetString(&a, "123456789012345678901"))
	bad := numeral.Int{Val: 1, Cell: z.FailedRef}
	before := m.BudgetField.Used()
	eq, e := makeEquation(m, []numeral.Int{a}, []uint32{0}, &bad, nil, nil, ib, rb, true, nil)
	if e == nil || eq != nil {
		t.Fatal("failed constant copy must clean the partial equation", eq, e)
	}
	if m.BudgetField.Used() != before {
		t.Fatal("partial equation header remained charged")
	}
	if got, e := z.String(&a); e != nil || got != "123456789012345678901" {
		t.Fatal("input lost its independent value", got, e)
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
	euclidMust(t, z.Del(&a))
}

func TestEuclideanEquationRationalCopyRefusalAndBufferCleanup(t *testing.T) {
	for ordinal := int64(0); ordinal <= 3; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := euclidNumeralManager(t)
			z := m.Integers()
			ib, e := numeral.NewScopedIntVector(z)
			euclidMust(t, e)
			rb, e := numeral.NewScopedRationalVector(m)
			euclidMust(t, e)
			euclidMust(t, ib.Resize(1))
			euclidMust(t, rb.Resize(1))
			input := numeral.NewRational()
			value := strings.Repeat("1234567890", 120) + "/" + strings.Repeat("9876543210", 120) + "7"
			euclidMust(t, m.SetString(&input, value))
			value, e = m.String(&input)
			euclidMust(t, e)
			as := []numeral.Int{numeral.NewInt(7)}
			c := numeral.NewInt(11)
			before := m.BudgetField.Used()
			m.BudgetField.FailAfter(ordinal)
			eq, e := makeEquation(m, as, []uint32{0}, &c, []numeral.Rational{input},
				[]uint32{0}, ib, rb, true, nil)
			m.BudgetField.FailAfter(-1)
			if (ordinal < 3) != (e != nil) {
				t.Fatal("header, numerator, denominator allocation sequence", ordinal, eq, e)
			}
			if e != nil {
				var failure umemory.Failure
				if !errors.Is(e, arena.ErrInjectedFailure) &&
					(!errors.As(e, &failure) || failure.Kind != umemory.FailureMemout) {
					t.Fatal("rational copy refusal", ordinal, e)
				}
				if eq != nil {
					t.Fatal("completed equation retained after source copy refusal", ordinal)
				}
				slot, err := rb.BorrowSlot(0)
				euclidMust(t, err)
				if ordinal == 1 && !z.IsFailureRef(slot.Num.Cell) {
					t.Fatal("numerator copy refusal prefix", slot)
				}
				if ordinal == 2 && (z.IsFailureRef(slot.Num.Cell) || !z.IsFailureRef(slot.Den.Cell)) {
					t.Fatal("denominator refusal retains copied numerator", slot)
				}
				euclidMust(t, rb.Cleanup())
			} else {
				got, err := m.String(&eq.bs[0])
				euclidMust(t, err)
				if got != value {
					t.Fatal("copied justification", got)
				}
				euclidMust(t, eq.Close())
				euclidMust(t, rb.Close())
			}
			euclidMust(t, ib.Close())
			if m.BudgetField.Used() != before {
				t.Fatal("partial equation or buffer retained budget", ordinal, before, m.BudgetField.Used())
			}
			got, err := m.String(&input)
			euclidMust(t, err)
			if got != value || as[0].Val != 7 || c.Val != 11 {
				t.Fatal("borrowed equation inputs changed", ordinal, got, as, c)
			}
			euclidMust(t, m.Del(&input))
		})
	}
}

func TestEuclideanStoppedRationalEquationCleanup(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	_, e = s.mkVar(false)
	euclidMust(t, e)
	j, e := s.mkJustification()
	euclidMust(t, e)
	input := numeral.NewRational()
	value := strings.Repeat("1234567890", 120) + "/" + strings.Repeat("9876543210", 120) + "7"
	euclidMust(t, m.SetString(&input, value))
	a, c := numeral.NewInt(7), numeral.NewInt(11)
	m.BudgetField.FailAfter(2)
	_, e = s.addEquation([]numeral.Int{a}, []uint32{0}, &c,
		[]numeral.Rational{input}, []uint32{j}, true)
	m.BudgetField.FailAfter(-1)
	var failure umemory.Failure
	if !errors.As(e, &failure) || failure.Kind != umemory.FailureMemout || s.stop == nil {
		t.Fatal("denominator copy stopped equation owner", e)
	}
	euclidMust(t, s.Close())
	euclidMust(t, m.Del(&input))
}

func TestEuclideanEquationFailedConstructionRetainsStoppedDestructor(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	euclidMust(t, e)
	m, e := numeral.NewRationalManager(numeral.RationalConfig{Synchronized: true, Integers: numeral.IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget, PageSize: 1},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	euclidMust(t, e)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	var a numeral.Int
	digits := make([]uint32, 70)
	digits[0], digits[69] = 1, 1
	euclidMust(t, z.SetLimbs(&a, digits))
	bad := numeral.Int{Val: 1, Cell: z.FailedRef}
	first, e := runtime.Allocate(8)
	euclidMust(t, e)
	second, e := runtime.Allocate(8)
	euclidMust(t, e)
	initialDelta := int64(-1000000)
	if e = runtime.Reallocate(&first, uint64(16)+uint64(initialDelta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("establish first TLS synchronization", e)
	}
	euclidMust(t, runtime.Deallocate(&first))
	state := runtime.State()
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	if e = runtime.Reallocate(&second, uint64(target)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("establish near-minimum global accounting", e)
	}
	euclidMust(t, runtime.Deallocate(&second))
	lane, e := runtime.Allocate(8)
	euclidMust(t, e)
	delta := int64(-100001) - runtime.State().LocalSize
	if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("establish deallocation boundary", e)
	}
	eq, e := makeEquation(m, []numeral.Int{a}, []uint32{0}, &bad, nil, nil, ib, rb, true, nil)
	if !errors.Is(e, arena.ErrBounds) || !errors.Is(e, arena.ErrInvalidSlot) || eq == nil || len(eq.as) != 1 || eq.closed {
		t.Fatal("stopped cleanup must return the partial owner", eq, e)
	}
	if !numeral.ValidNumericRef(z.Cells, eq.as[0].Cell) {
		t.Fatal("stopped destruction lost the coefficient owner")
	}
	recovery, e := runtime.Allocate(200000)
	euclidMust(t, e)
	euclidMust(t, eq.Close())
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
	euclidMust(t, z.Del(&a))
	euclidMust(t, runtime.Deallocate(&lane))
	state = runtime.State()
	delta = int64(-500000) - state.GlobalSize - state.LocalSize
	if e = runtime.Reallocate(&recovery, uint64(200000)+uint64(delta)); e == nil {
		t.Fatal("restore accounting range before teardown")
	}
	euclidMust(t, runtime.Deallocate(&recovery))
	euclidMust(t, m.CloseChecked())
	euclidMust(t, runtime.Close())
	if budget.Used() != 0 {
		t.Fatal("retained equation leaked allocation", budget.Used())
	}
}
