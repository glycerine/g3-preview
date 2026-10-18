package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialFactorsDisplaySourceOrderAndCallbacks(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.CleanupDisplay()); serviceMust(t, fs.Close()) })
	constant := unumeral.NewInt(-2)
	serviceMust(t, fs.SetConstant(&constant))
	for i, coeffs := range [][]int32{{1, 1}, {-3, 1}} {
		p := upolySourceNew(t, m, coeffs)
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), uint32(2-i)))
	}
	called := false
	out := &upolyDisplaySink{literalHook: func(string) error {
		if called {
			return nil
		}
		called = true
		for _, e := range []error{fs.Close(), fs.Clear(), fs.CleanupDisplay(), m.Close(), m.SourceFinalize(), m.CleanupDisplay()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Error("display callback acquired borrowed owner", e)
			}
		}
		for _, e := range []error{m.config.Integer.CloseChecked(), m.modular.Close()} {
			if !errors.Is(e, unumeral.ErrNumericPending) {
				t.Error("display callback acquired numeral owner", e)
			}
		}
		return nil
	}}
	serviceMust(t, fs.DisplayInto(out))
	if !called || out.text.String() != "-2 * (x + 1)^2 * (x - 3)^1" {
		t.Fatal("source factor display", called, out.text.String())
	}
	if fs.displayScope.active || m.factorsDisplay != nil || m.pendingFactorsDisplay != nil {
		t.Fatal("completed factor display retained scope")
	}
}

func TestUPolynomialFactorsDisplayExactPendingChildAndForeignReplacement(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned", true: "foreign"}[replace], func(t *testing.T) {
			target := algebraicCopyBoundedTargetSlots(t, 10)
			m, z := target.config.Polynomials, target.config.Rationals.Integers()
			p, e := m.NewPolynomial()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)) })
			serviceMust(t, m.SetInts(p, []unumeral.Int{z.Two64}))
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.CleanupDisplay()); serviceMust(t, fs.Close()) })
			view, e := m.wholeView(p)
			serviceMust(t, e)
			serviceMust(t, fs.PushCopy(view, 1))
			out := &upolyDisplaySink{}
			body := fs.DisplayInto(out)
			if !errors.Is(body, arena.ErrSlotLimit) || !fs.displayScope.pending() || out.text.String() != "1 * (" {
				t.Fatal("actual child refusal/prefix", body, out.text.String())
			}
			for _, e := range []error{fs.Close(), m.SourceFinalize(), m.Close()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("pending display lost owner", e)
				}
			}
			var foreign upolyDisplayInvocationReceipt
			if replace {
				serviceMust(t, m.CleanupDisplay())
				if e = m.displayIntoWithReceipt(view, &upolyDisplaySink{}, "x", false, &foreign); !errors.Is(e, arena.ErrSlotLimit) {
					t.Fatal("foreign actual display refusal", e)
				}
			}
			physical := z.PendingCell
			serviceMust(t, fs.CleanupDisplay())
			if replace {
				if !m.displayReceiptPending(&foreign) || z.PendingCell != physical {
					t.Fatal("factor display cleanup acquired foreign child")
				}
				serviceMust(t, m.cleanupDisplayReceipt(&foreign))
			} else if fs.displayScope.pending() {
				t.Fatal("owned child survived acknowledgement")
			}
		})
	}
}
