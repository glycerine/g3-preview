package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialBerlekampConstructorActualNumericReservationPrefixes(t *testing.T) {
	completed, retained, addMul := false, false, false
	for ordinal := int64(0); ordinal < 512 && !completed; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m, b := s.manager, q.Budget()
			three := unumeral.NewInt(3)
			serviceMust(t, m.SetZP(&three))
			// Concrete integer-mode constructor profile with remembered prime 3.
			// Actual BIG coefficients exercise allocation leaves; no finite-field
			// factorization or C++ malloc correspondence is claimed here.
			serviceMust(t, m.SetZ())
			var big unumeral.Int
			serviceMust(t, q.Integers().SetLimbs(&big, []uint32{1, 0, 1}))
			t.Cleanup(func() { serviceMust(t, q.Integers().Del(&big)) })
			f := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(f, []unumeral.Int{big, big, unumeral.NewInt(1)}))
			matrix := upolyBerlekampMatrix{m: m}
			t.Cleanup(func() { b.FailAfter(-1); serviceMust(t, m.CleanupBerlekampMatrix()); matrix.close() })
			view := mustUPolyView(t, m, f)
			b.FailAfter(ordinal)
			body := matrix.init(m, view)
			b.FailAfter(-1)
			if matrix.numeric.pending() {
				retained = true
				if m.pendingBerlekamp != &matrix {
					t.Fatal("actual numeric acquisition lost matrix owner", ordinal, body)
				}
			}
			if matrix.numeric.addMul.Matches(q.Integers()) {
				addMul = true
			}
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected constructor numerical prefix", ordinal, body)
			}
			if body == nil {
				completed = true
				// Literal integer-mode recurrence for B=2^64+1 gives row
				// [B^2, B^2-B-1] after subtracting the diagonal identity.
				values := m.values(&matrix.values)
				first, e := q.Integers().String(&values[2])
				serviceMust(t, e)
				second, e := q.Integers().String(&values[3])
				serviceMust(t, e)
				if first != "340282366920938463500268095579187314689" || second != "340282366920938463481821351505477763071" {
					t.Fatal("independent literal BIG recurrence", first, second)
				}
			}
			serviceMust(t, m.CleanupBerlekampMatrix())
			matrix.close()
			if matrix.numeric.pending() || m.pendingBerlekamp != nil || matrix.values.backing != 0 {
				t.Fatal("exact numeric cleanup left matrix acquisition", ordinal, body)
			}
		})
	}
	if !completed || !retained || !addMul {
		t.Fatal("missing actual constructor completion/allocation/AddMul coverage", completed, retained, addMul)
	}
}

func TestUPolynomialBerlekampNumericCleanupPreservesIndependentGCD(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 32 && !reached; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m, z := s.manager, q.Integers()
			three := unumeral.NewInt(3)
			serviceMust(t, m.SetZP(&three))
			f := upolySourceNew(t, m, []int32{1, 0, 1})
			var large, out unumeral.Int
			serviceMust(t, z.SetLimbs(&large, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			var independent unumeral.IntegerRawInvocationReceipt
			matrix := upolyBerlekampMatrix{m: m}
			t.Cleanup(func() {
				q.Budget().FailAfter(-1)
				serviceMust(t, m.CleanupBerlekampMatrix())
				matrix.close()
				serviceMust(t, z.CleanupRawInvocation(&independent, true))
				serviceMust(t, z.CleanupRawNumeral(&out))
				serviceMust(t, z.Del(&large))
			})
			prior := z.GCDInvocation()
			q.Budget().FailAfter(ordinal)
			body := z.GCD(&large, &large, &out)
			q.Budget().FailAfter(-1)
			independent = z.GCDInvocation()
			if body == nil || independent == prior || !independent.Matches(z) {
				return
			}
			reached = true
			serviceMust(t, matrix.init(m, mustUPolyView(t, m, f)))
			serviceMust(t, matrix.numeric.cleanup(false))
			serviceMust(t, matrix.numeric.cleanup(true))
			matrix.close()
			if !independent.Matches(z) || z.GCDInvocation() != independent || matrix.values.backing != 0 {
				t.Fatal("matrix numerical cleanup adopted independent GCD", body)
			}
			serviceMust(t, z.CleanupRawInvocation(&independent, true))
		})
	}
	if !reached {
		t.Fatal("actual independently retained GCD not reached")
	}
}

func TestUPolynomialBerlekampActualNumericTransportProtectsCaller(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var matrix upolyBerlekampMatrix
	armed, calls := false, 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			if !matrix.numeric.executing || matrix.numeric.generation == 0 {
				t.Fatal("actual numeric transport lost executing caller")
			}
			values, index, size := matrix.values, matrix.deleteIndex, matrix.size
			var untouched upolyVectorState
			_, de := matrix.diagonalize()
			_, ne := matrix.next(&untouched)
			for _, e := range []error{matrix.init(matrix.m, UPolynomialView{}), de, ne, matrix.m.Close(), matrix.m.SourceFinalize(), matrix.m.CleanupBerlekampMatrix()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("numeric callback acquired executing matrix/core", e)
				}
			}
			matrix.close()
			if matrix.values != values || matrix.deleteIndex != index || matrix.size != size || matrix.m.pendingBerlekamp != nil || untouched.backing != 0 {
				t.Fatal("numeric callback changed actual caller frontier")
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	matrix.m = m
	var held umemory.Block
	var big unumeral.Int
	f, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() {
		armed = false
		b.FailAfter(-1)
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupBerlekampMatrix())
		matrix.close()
		serviceMust(t, m.DelPolynomial(&f))
		serviceMust(t, q.Integers().Del(&big))
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, m.Close())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual transport owner refund", b.Used())
		}
	})
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	serviceMust(t, m.SetZ())
	words := make([]uint32, 80)
	words[79] = 1
	serviceMust(t, q.Integers().SetLimbs(&big, words))
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, big, unumeral.NewInt(1)}))
	view := mustUPolyView(t, m, f)
	pad := int64(99996) - rt.State().LocalSize - 12
	if pad <= 0 {
		t.Fatal("actual runtime threshold", rt.State())
	}
	held, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := matrix.init(m, view)
	armed = false
	serviceMust(t, rt.SetMaxSize(0))
	if calls == 0 || !numericPhysicalReservationRefusal(body) {
		t.Fatal("actual runtime transport prefix not reached", calls, body)
	}
	serviceMust(t, m.CleanupBerlekampMatrix())
	matrix.close()
}
