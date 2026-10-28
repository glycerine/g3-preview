package polynomial

// Source: src/util/mpz.cpp:1385-1410; src/math/polynomial/upolynomial.cpp:1108-1225.

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Native output text survives lexical cleanup and reuse of the embedded call.
// The actual insertion receipt still cannot end an executing lexical child.
func TestUPolynomialDisplayNativeTextSurvivesLaterEmbeddedCall(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a unumeral.Int
	serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []unumeral.Int{a}))
	serviceMust(t, r.Integers().Del(&a))
	defer func() {
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
	}()
	var old string
	first := &upolyDisplaySink{numeralHook: func(v string) error {
		old = v
		// Neither cleanup mode may end the still-executing lexical child.
		for _, owning := range []bool{false, true} {
			before := m.display.insertion
			used := r.Integers().Budget().Used()
			if e := unumeral.CompleteNumeralInsertion(&m.display.insertion, r.Integers().Naturals(), nil, owning); e != arena.ErrInvalidSlot {
				t.Fatalf("executing borrowed insertion cleanup: %v", e)
			}
			if before != m.display.insertion || used != r.Integers().Budget().Used() {
				t.Fatal("executing cleanup refusal changed the real receipt or charged state")
			}
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), first, "x", false))
	if old != "79228162514264337593543950343" {
		t.Fatal("completed native text", old)
	}
	called := false
	second := &upolyDisplaySink{numeralHook: func(v string) error {
		called = true
		if old != v {
			t.Fatal("later native value changed", old, v)
		}
		before := r.Integers().Budget().Used()
		if old != "79228162514264337593543950343" {
			t.Fatal("old native text changed", old)
		}
		if before != r.Integers().Budget().Used() {
			t.Fatal("native text read changed charged state")
		}
		return nil
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), second, "x", false))
	if !called || first.text.String() != second.text.String() {
		t.Fatal("later real display did not complete identically")
	}
	primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
}

// Copied receipts are values, but only their original issued invocation may
// authorize recovery. Reusing the embedded frame must not revive an old tuple.
func TestUPolynomialDisplayInsertionReceiptRefusesLaterEmbeddedCall(t *testing.T) {
	for _, owning := range []bool{false, true} {
		t.Run(strconv.FormatBool(owning), func(t *testing.T) {
			r, s := preparationPhysicalFixture(t)
			m := s.manager
			var a unumeral.Int
			serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []unumeral.Int{a}))
			serviceMust(t, r.Integers().Del(&a))
			defer func() {
				serviceMust(t, m.CleanupDisplay())
				serviceMust(t, m.reset(&p))
				m.freeBacking(&p)
			}()
			marker := errors.New("actual borrowed output failure")
			out := &upolyDisplaySink{numeralHook: func(string) error { return marker }}
			if e := m.DisplayInto(m.stateView(&p), out, "x", false); e != marker {
				t.Fatal("first output failure", e)
			}
			old := m.display.insertion
			if !old.Active() {
				t.Fatal("first actual insertion did not retain its recovery tuple")
			}
			serviceMust(t, m.CleanupDisplay())
			if e := m.DisplayInto(m.stateView(&p), out, "x", false); e != marker {
				t.Fatal("later output failure", e)
			}
			if old.Character != m.display.insertion.Character || !m.display.insertion.Active() {
				t.Fatal("fixture did not reuse actual pending embedded insertion")
			}
			used := r.Integers().Budget().Used()
			if e := unumeral.CompleteNumeralInsertion(&old, r.Integers().Naturals(), nil, owning); e != arena.ErrInvalidSlot {
				t.Errorf("old insertion receipt acquired later recovery authority: %v", e)
			}
			if used != r.Integers().Budget().Used() || !m.display.insertion.Active() {
				t.Fatal("old receipt changed later caller storage")
			}
			primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
		})
	}
}
