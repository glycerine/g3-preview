package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type algebraicIsolationSignsDestructorMap struct {
	algebraicIsolationAssignments
	core    *AlgebraicRootCore
	runtime *umemory.Runtime
	t       *testing.T
	lane    umemory.Block
	marker  error
	armed   bool
}

func (v *algebraicIsolationSignsDestructorMap) Contains(x uint32) (bool, error) {
	if !v.armed && v.core.isolationSigns.constructed && v.core.signEvaluationRunning {
		v.armed = true
		// A real getter Throw ends the body. The qualified signed-host TLS guard
		// then stops the actual source destructor of the BIG scoped sample.
		serviceMust(v.t, v.runtime.SetMaxSize(0))
		contextTLSNearMinimum(v.t, v.runtime)
		var e error
		v.lane, e = v.runtime.Allocate(8)
		serviceMust(v.t, e)
		delta := int64(-1000000)
		if e = v.runtime.Reallocate(&v.lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			v.t.Fatal(e)
		}
		serviceMust(v.t, v.runtime.SetMaxSize(1))
		return false, v.marker
	}
	return v.algebraicIsolationAssignments.Contains(x)
}
func TestAlgebraicIsolationSignsActualSampleDestructorStop(t *testing.T) {
	c, _, runtime := algebraicNegTLSFixture(t, nil)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}})
	values := algebraicIsolationSignsDestructorMap{core: c, runtime: runtime, t: t, marker: errors.New("sample assignment getter Throw")}
	n := unumeral.NewInt(0)
	serviceMust(t, c.config.Rationals.Integers().SetLimbs(&n, []uint32{1, 0, 1}))
	values.assigned[0] = true
	serviceMust(t, c.SetInteger(&values.values[0], &n))
	serviceMust(t, c.config.Rationals.Integers().Del(&n))
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() {
		serviceMust(t, runtime.SetMaxSize(0))
		contextTLSFinish(t, runtime, c.CleanupRetained)
		contextTLSFinish(t, runtime, func() error {
			if values.lane == (umemory.Block{}) {
				return nil
			}
			return runtime.Deallocate(&values.lane)
		})
		if e := roots.Close(); e != nil {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("source vector close frontier", e)
			}
			contextTLSFinish(t, runtime, roots.CleanupRetained)
			serviceMust(t, roots.Close())
		}
		contextTLSFinish(t, runtime, func() error { return c.CleanupValue(&values.values[0]) })
		// Take the fixture's real caller INC into an owning reference so a
		// stopped polynomial destructor resumes instead of repeating DEC.
		reference := polynomialReference{value: p}
		contextTLSFinish(t, runtime, func() error { return reference.close(pm.core) })
		contextTLSFinish(t, runtime, pm.CloseChecked)
	}()
	signs := []int{71}
	body := c.IsolateMultivariateRootsWithSigns(pm, p, &values, roots, &signs)
	if !values.armed || !errors.Is(body, values.marker) {
		t.Fatal("actual getter did not throw", values.armed, body)
	}
	s := &c.isolationSigns
	if s.sourceStop == nil || !s.active || !s.constructed || s.sample.slot == 0 {
		t.Fatal("source sample destructor did not stop", body, s.sourceStop, s.active, s.constructed)
	}
	if len(signs) != 1 || signs[0] != 71 || roots.Len() != 1 {
		t.Fatal("source stop changed caller outputs", signs, roots.Len())
	}
	if e := pm.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("retained sample lost its borrowed manager", e)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	contextTLSFinish(t, runtime, c.CleanupRetained)
	if s.active || s.constructed || s.sample != (Algebraic{}) || pm.algebraicIsolationBorrowers != 0 {
		t.Fatal("owning cleanup retained sample")
	}
}
