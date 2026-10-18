package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type berlekampConstructorProbe struct {
	t       *testing.T
	matrix  *upolyBerlekampMatrix
	f       UPolynomialView
	text    strings.Builder
	checked bool
}

func (w *berlekampConstructorProbe) Write(p []byte) (int, error) {
	if string(p) == "polynomial::berlekamp_matrix(" {
		w.checked = true
		q := w.matrix
		if q.values.size != 0 || q.values.backing != 0 || len(q.column) != 2 || len(q.row) != 2 || q.nullRow != 1 || q.column[0] != -1 || q.row[1] != -1 {
			w.t.Fatal("constructor trace crossed actual member/matrix frontier")
		}
		q.close()
		for _, e := range []error{q.init(q.m, w.f), q.m.Close(), q.m.SourceFinalize(), q.m.CleanupDisplay()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("constructor callback acquired active owner", e)
			}
		}
		if len(q.column) != 2 || len(q.row) != 2 {
			w.t.Fatal("callback destroyed real pivot members")
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialBerlekampConstructorTraceBeforeMatrixValues(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	w := berlekampConstructorProbe{t: t, matrix: &q, f: mustUPolyView(t, m, f)}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	serviceMust(t, q.init(m, w.f))
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		text := w.text.String()
		if !w.checked || !strings.Contains(text, "polynomial::berlekamp_matrix(x^2 + 1, 3)\n") || strings.Index(text, ":97 ") >= strings.Index(text, ":142 ") {
			t.Fatal("source constructor trace text/order", text)
		}
	} else if w.checked || w.text.Len() != 0 {
		t.Fatal("erased constructor trace executed")
	}
}

func TestUPolynomialBerlekampConstructorTraceExactDisplayRefusal(t *testing.T) {
	qn, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, qn.Integers()
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	// A deliberate integer-mode direct constructor prefix permits an actual
	// BIG display operand while preserving the genuine remembered p=3. No
	// finite-field arithmetic is reached or claimed for this invalid input.
	serviceMust(t, m.SetZ())
	var big unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
	f := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, unumeral.NewInt(0), unumeral.NewInt(1)}))
	matrix := upolyBerlekampMatrix{m: m}
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "polynomial::berlekamp_matrix("}
	t.Cleanup(func() {
		serviceMust(t, m.cleanupDisplayReceipt(&matrix.entryTrace))
		matrix.close()
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		t.Skip("trace compiled out")
	}
	e := matrix.init(m, mustUPolyView(t, m, f))
	if !errors.Is(e, arena.ErrSlotLimit) || !w.armed || matrix.values.size != 0 || matrix.values.backing != 0 || !m.displayReceiptPending(&matrix.entryTrace) {
		t.Fatal("actual constructor display child/frontier", e)
	}
	saved := matrix.entryTrace
	matrix.close()
	if len(matrix.column) != 2 || len(matrix.row) != 2 || matrix.entryTrace != saved {
		t.Fatal("older members crossed unfinished display child")
	}
	if e := matrix.init(m, UPolynomialView{}); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("reinitialization adopted unfinished constructor", e)
	}
	var untouched upolyVectorState
	_, diagonalErr := matrix.diagonalize()
	_, nextErr := matrix.next(&untouched)
	for _, e := range []error{diagonalErr, nextErr} {
		if !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Fatal("unfinished constructor accepted matrix operation", e)
		}
	}
	if untouched.size != 0 || untouched.backing != 0 || matrix.nullRow != 1 {
		t.Fatal("unfinished constructor changed null-vector frontier")
	}
	if !errors.Is(saved.frame.bodyCause, arena.ErrSlotLimit) {
		t.Fatal("exact display receipt lost content-copy body refusal")
	}
	serviceMust(t, m.cleanupDisplayReceipt(&saved))
	matrix.close()
	if matrix.column != nil || matrix.row != nil || m.displayReceiptPending(&saved) {
		t.Fatal("exact child cleanup did not release pivot members")
	}
	// A later independently accepted display may reuse the embedded frame.
	// The old constructor receipt must not gain authority over its new nonce.
	var independent upolyDisplayInvocationReceipt
	var output strings.Builder
	e = m.displayIntoWithReceipt(mustUPolyView(t, m, f), factorizationTraceStream{&output}, "x", false, &independent)
	if !errors.Is(e, arena.ErrSlotLimit) || !m.displayReceiptPending(&independent) {
		t.Fatal("independent actual display refusal", e)
	}
	serviceMust(t, m.cleanupDisplayReceipt(&saved))
	if !m.displayReceiptPending(&independent) {
		t.Fatal("stale constructor receipt cleaned replacement display")
	}
	serviceMust(t, m.cleanupDisplayReceipt(&independent))
}
