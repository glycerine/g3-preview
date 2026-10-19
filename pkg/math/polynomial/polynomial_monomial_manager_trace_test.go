package polynomial

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialManagerDestructorLeakTraceAndAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			_, e := m.MkVar()
			serviceMust(t, e)
			leaked, e := m.MkVariable(0, 2)
			serviceMust(t, e)
			action := choice
			if choice == "disabled" {
				action = "T"
			}
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("polynomial")
			m.config.Trace = tags
			e = m.SourceFinalize()
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) || m.sourceStage != 1 {
					t.Fatal("table assertion did not stop after leak trace", e, m.sourceStage)
				}
				serviceMust(t, m.CleanupValue(&leaked))
				serviceMust(t, m.SourceFinalize())
			} else {
				serviceMust(t, e)
			}
			if leaked != (Monomial{}) {
				serviceMust(t, m.CleanupValue(&leaked))
			}
			want := ""
			if tags.ShouldTrace("polynomial") {
				want = "-------- [polynomial] ~monomial_manager src/math/polynomial/polynomial.cpp:796 ---------\n" +
					"monomials leaked\nx0^2\n------------------------------------------------\n"
			}
			if output.String() != want {
				t.Fatalf("source destructor trace: got %q, want %q", output.String(), want)
			}
			if m.managerTraceRunning {
				t.Fatal("destructor trace span retained")
			}
		})
	}
}

func TestMonomialManagerDecrementAssertionActions(t *testing.T) {
	for _, choice := range []string{"T", "C", "disabled"} {
		t.Run(choice, func(t *testing.T) {
			m := monomialFixture(t)
			action := choice
			if choice == "disabled" {
				action = "T"
			}
			polynomialAssertionActionsFixture(t, action)
			if polynomialSourceDebug && choice == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e := m.DecManagerRef()
			if polynomialSourceDebug && choice == "T" {
				if !errors.Is(e, ErrMonomialAssertion) || m.managerRefs != 0 {
					t.Fatal("manager decrement assertion mutated count", e, m.managerRefs)
				}
			} else {
				serviceMust(t, e)
				if m.managerRefs != ^uint32(0) {
					t.Fatal("continued source unsigned manager decrement", m.managerRefs)
				}
				m.managerRefs = 0
			}
		})
	}
}
