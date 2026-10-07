// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1385-1410; src/math/polynomial/upolynomial.cpp:1108-1225.
// Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// The old c_str belongs to an exited lexical buffer, even when the next
// actual display reuses the embedded call metadata and prints identical bytes.
func TestUPolynomialDisplayBorrowRefusesLaterLiveEmbeddedCall(t *testing.T) {
	r, s := preparationPhysicalFixture(t)
	m := s.manager
	var a util.Int
	serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
	var p upolyVectorState
	serviceMust(t, m.set(&p, []util.Int{a}))
	serviceMust(t, r.Integers().Del(&a))
	defer func() {
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, m.reset(&p))
		m.freeBacking(&p)
	}()
	var old util.NumeralCStringView
	first := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error {
		old = v
		// Neither cleanup mode may end the still-executing lexical child.
		for _, owning := range []bool{false, true} {
			before := m.display.insertion
			used := r.Integers().Budget().Used()
			if e := util.CompleteNumeralInsertion(&m.display.insertion, r.Integers().Naturals(), nil, owning); e != arena.ErrInvalidSlot {
				t.Fatalf("executing borrowed insertion cleanup: %v", e)
			}
			if before != m.display.insertion || used != r.Integers().Budget().Used() {
				t.Fatal("executing cleanup refusal changed the real receipt or charged state")
			}
		}
		_, e := v.Byte(0)
		return e
	}}
	serviceMust(t, m.DisplayInto(m.stateView(&p), first, "x", false))
	if _, e := old.Byte(0); e != arena.ErrInvalidSlot {
		t.Fatalf("exited lexical borrow: %v", e)
	}
	called := false
	second := &upolyDisplaySink{numeralHook: func(v util.NumeralCStringView) error {
		called = true
		if old.Character != v.Character {
			t.Fatal("fixture did not reuse actual embedded character metadata")
		}
		before := r.Integers().Budget().Used()
		if b, e := old.Byte(0); e != arena.ErrInvalidSlot {
			t.Errorf("old lexical borrow retargeted to later live call: byte %d, error %v", b, e)
		}
		if before != r.Integers().Budget().Used() {
			t.Fatal("stale view read changed charged state")
		}
		_, e := v.Byte(0)
		return e
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
			var a util.Int
			serviceMust(t, r.Integers().SetLimbs(&a, []uint32{7, 0, 0, 1}))
			var p upolyVectorState
			serviceMust(t, m.set(&p, []util.Int{a}))
			serviceMust(t, r.Integers().Del(&a))
			defer func() {
				serviceMust(t, m.CleanupDisplay())
				serviceMust(t, m.reset(&p))
				m.freeBacking(&p)
			}()
			marker := errors.New("actual borrowed output failure")
			out := &upolyDisplaySink{numeralHook: func(util.NumeralCStringView) error { return marker }}
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
			if e := util.CompleteNumeralInsertion(&old, r.Integers().Naturals(), nil, owning); e != arena.ErrInvalidSlot {
				t.Errorf("old insertion receipt acquired later recovery authority: %v", e)
			}
			if used != r.Integers().Budget().Used() || !m.display.insertion.Active() {
				t.Fatal("old receipt changed later caller storage")
			}
			primeTrialPhysicalAuditOwners(t, r, s, nil, &m.values(&p)[0])
		})
	}
}
