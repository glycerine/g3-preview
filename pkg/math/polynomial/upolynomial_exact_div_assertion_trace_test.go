package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strings"
	"testing"
)

func TestUPolynomialExactDivCopiedSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a := upolySourceNew(t, m, []int32{1})
			serviceMust(t, m.Reserve(a, 2))
			b := upolySourceNew(t, m, []int32{1})
			out := upolySourceNew(t, m, []int32{99})
			// The caller's real initialized trailing-zero slot survives normalizing
			// the copied size. Continue reaches that actual slot; no scratch repair.
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := m.ExactDiv(a, b, out)
			if polynomialSourceDebug && action == "T" {
				if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || m.exactDiv.size != 2 || m.div1.size != 1 || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{99}) {
					t.Fatal("post-copy Throw changed source prepared buffers or caller output", ok, e)
				}
			} else {
				serviceMust(t, e)
				if !ok || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{1}) {
					t.Fatal("literal initialized-slot continuation", ok, upolyDivisionValues(t, m, out))
				}
			}
			if m.factorTrace != nil {
				t.Fatal("exact division assertion retained manager borrow")
			}
		})
	}
}

type exactDivPointerProbe struct {
	t        *testing.T
	m        *UPolynomialManager
	observed bool
	text     strings.Builder
}

func (w *exactDivPointerProbe) Write(p []byte) (int, error) {
	if string(p) == "sz1: " {
		w.observed = true
		if w.m.div1.size != 1 || w.m.values(&w.m.div1)[0] != unumeral.NewInt(72) || w.m.exactDiv.size != 2 {
			w.t.Fatal("pointer record crossed source copy/preparation frontier")
		}
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupDisplay()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("pointer trace callback acquired active owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialExactDivPointerRecordUsesActualPreCopyStorage(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	// A real preceding call leaves the caller's former remainder in m_div_tmp1.
	previousA, previousB := upolySourceNew(t, m, []int32{6, 1}), upolySourceNew(t, m, []int32{1, 1})
	previousQ, previousR := upolySourceNew(t, m, nil), upolySourceNew(t, m, []int32{72})
	var d uint32
	serviceMust(t, m.DivRem(previousA, previousB, previousQ, previousR, &d))
	if m.div1.size != 1 || m.values(&m.div1)[0] != unumeral.NewInt(72) {
		t.Fatal("actual preceding div_rem private-member state")
	}
	a, b := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, []int32{-1, 1})
	out := upolySourceNew(t, m, nil)
	input, e := m.read(mustUPolyView(t, m, a), 0)
	serviceMust(t, e)
	expected := fmt.Sprintf("sz1: 3 p1: %p, _p1.c_ptr(): %p, _p1.size(): 1\n", input, &m.values(&m.div1)[0])
	w := exactDivPointerProbe{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("factor_bug")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	ok, e := m.ExactDiv(a, b, out)
	serviceMust(t, e)
	if !ok || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{1, 1}) {
		t.Fatal("pointer record changed exact quotient", ok)
	}
	if tags.ShouldTrace("factor_bug") {
		if !w.observed || !strings.Contains(w.text.String(), expected) || !strings.Contains(w.text.String(), ":624 ") {
			t.Fatal("native storage pointer record/source location", expected, w.text.String())
		}
	} else if w.observed || w.text.Len() != 0 {
		t.Fatal("erased pointer record executed")
	}
}
func TestUPolynomialExactDivPointerRecordStickyStreamFailure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a, b := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, []int32{-1, 1})
	out := upolySourceNew(t, m, nil)
	w := factorDegreeRefusedTrace{cause: errors.New("pointer record stream refusal")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("factor_bug")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	ok, e := m.ExactDiv(a, b, out)
	serviceMust(t, e)
	if !ok || !reflect.DeepEqual(upolyDivisionValues(t, m, out), []int64{1, 1}) {
		t.Fatal("pointer stream refusal changed quotient", ok)
	}
	if tags.ShouldTrace("factor_bug") {
		if w.calls != 1 || w.flushes != 1 {
			t.Fatal("sticky pointer record host writes/footer flush", w.calls, w.flushes)
		}
	} else if w.calls != 0 {
		t.Fatal("erased pointer record executed")
	}
}
