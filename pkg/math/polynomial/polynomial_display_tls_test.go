// Source: src/math/polynomial/polynomial.cpp:1558-1570; src/util/mpz.cpp;
// src/util/buffer.h; src/util/memory_manager.cpp.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func polynomialDisplayTLSOwner(t *testing.T) (*MultivariatePolynomialManager, *util.RationalManager, *util.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = r
	cfg.Profile = util.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { panic("display has no source checkpoint") }, Cooperation: func() error { panic("display has no source cooperation") }}})
	serviceMust(t, e)
	return m, q, r, b
}
func TestPolynomialDisplayExactNumericReceiptAndReplacement(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprint(replace), func(t *testing.T) {
			m, q, r, b := polynomialDisplayTLSOwner(t)
			z := q.Integers()
			digits := make([]uint32, 40000)
			digits[39999] = 1
			a, foreign := util.Int{}, util.Int{}
			serviceMust(t, z.SetLimbs(&a, digits))
			defer func() {
				serviceMust(t, r.SetMaxSize(0))
				serviceMust(t, m.CleanupDisplay())
				serviceMust(t, z.Allocator.CleanupPending())
				serviceMust(t, z.Del(&foreign))
				serviceMust(t, z.Del(&a))
				serviceMust(t, m.CloseChecked())
				serviceMust(t, q.CloseChecked())
				serviceMust(t, r.Close())
				if b.Used() != 0 {
					t.Fatal("whole exact receipt graph refund", b.Used())
				}
			}()
			f, e := m.core.displayFrame(&polynomialDisplayTestSink{})
			serviceMust(t, e)
			f.constructed = true
			serviceMust(t, r.SetMaxSize(1))
			cause := f.numericCall(0, &a)
			roots, e := z.Allocator.PendingRoots()
			serviceMust(t, e)
			if !numericPhysicalReservationRefusal(cause) || roots[0] == (util.Block{}) || f.numeric.Allocator != roots {
				t.Fatal("actual invoked Copy raw receipt", cause, roots, f.numeric.Allocator)
			}
			cause = f.finish(cause)
			polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
			old := roots
			if replace {
				serviceMust(t, r.SetMaxSize(0))
				serviceMust(t, z.Allocator.CleanupPending())
				serviceMust(t, r.SetMaxSize(1))
				e = z.Copy(&foreign, &a)
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual foreign replacement Copy", e)
				}
				roots, e = z.Allocator.PendingRoots()
				serviceMust(t, e)
				if roots[0] == (util.Block{}) || roots == old {
					t.Fatal("replacement needs a distinct actual token", roots, old)
				}
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a, &foreign}, 0)
			}
			serviceMust(t, r.SetMaxSize(0))
			serviceMust(t, m.CleanupDisplay())
			now, e := z.Allocator.PendingRoots()
			serviceMust(t, e)
			if replace {
				if now != roots {
					t.Fatal("owning display cleanup swept later foreign raw", now, roots)
				}
			} else if now != ([3]util.Block{}) {
				t.Fatal("owning display cleanup omitted its own exact Copy raw", now)
			}
			if f.bodyCause == nil || !errors.Is(f.bodyCause, cause) {
				t.Fatal("original body cause lost during owning continuation")
			}
			polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a, &foreign}, 0)
		})
	}
}
func TestPolynomialDisplaySinkCreatesForeignPending(t *testing.T) {
	m, q, r, b := polynomialDisplayTLSOwner(t)
	z := q.Integers()
	a, foreign := util.Int{}, util.Int{}
	serviceMust(t, z.Copy(&a, &z.Two64))
	p, e := m.core.MkConstantTake(&a)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, r.SetMaxSize(0))
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, z.Allocator.CleanupPending())
		serviceMust(t, z.Del(&foreign))
		serviceMust(t, m.core.CleanupValue(&p))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	var owned [3]util.Block
	sink := &polynomialDisplayTestSink{decimalHook: func(*PolynomialDecimalView) error {
		digits := make([]uint32, 40000)
		digits[39999] = 1
		serviceMust(t, r.SetMaxSize(1))
		cause := z.SetLimbs(&foreign, digits)
		if !numericPhysicalReservationRefusal(cause) {
			t.Fatal(cause)
		}
		owned, e = z.Allocator.PendingRoots()
		serviceMust(t, e)
		if owned[0] == (util.Block{}) {
			t.Fatal("no actual caller pending raw")
		}
		return cause
	}}
	cause := m.DisplaySMT2Into(p, sink)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal(cause)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&foreign}, 0)
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupDisplay())
	now, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if now != owned {
		t.Fatal("stream callback roots were attributed to parent display", now, owned)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&foreign}, 0)
}
