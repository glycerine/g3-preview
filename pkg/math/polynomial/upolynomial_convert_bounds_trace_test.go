package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func TestUPolynomialConvertNegativeBoundsTraceAfterSwap(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	p := upolySourceNew(t, m, []int32{1, 1})
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetFraction(&a, -5, 3))
	serviceMust(t, q.SetFraction(&b, -1, 2))
	c, d := unumeral.NewBinaryRational(), unumeral.NewBinaryRational()
	t.Cleanup(func() {
		serviceMust(t, m.CleanupIntervalConversion())
		serviceMust(t, bqm.Del(&c))
		serviceMust(t, bqm.Del(&d))
		serviceMust(t, q.Del(&a))
		serviceMust(t, q.Del(&b))
	})
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("convert_bug")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	ok, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &c, &d)
	serviceMust(t, e)
	if !ok || upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "-5/4" || upolyIsolationBinaryRat(t, m, bqm, &d).RatString() != "-1/2" {
		t.Fatal("independent negative interval", ok)
	}
	if tags.ShouldTrace("convert_bug") && !strings.Contains(text.String(), "a: -5/3\nlower: -5/2, upper: -5/2^2\n") {
		t.Fatal("trace preceded negative bound swap", text.String())
	}
	if m.intervalBounds.active || m.intervalBounds.constructed != 0 {
		t.Fatal("completed bound scope retained")
	}
}

type convertBoundsReadGuard struct {
	craTraceReadGuard
	binary         *unumeral.BinaryRationalManager
	input          *unumeral.Rational
	kind           int // 0 exact lower; 1 numerator; 2 denominator; 3 lower; 4 upper
	armed, refused bool
}

func (w *convertBoundsReadGuard) Write(p []byte) (int, error) {
	text := string(p)
	line := "2841"
	if w.kind == 0 {
		line = "2830"
	}
	if text == line {
		w.armed = true
	}
	trigger := w.armed && (w.kind == 0 && text == "found c: " || w.kind == 1 && text == "a: " || w.kind == 2 && text == "/" || w.kind == 3 && text == "\nlower: " || w.kind == 4 && text == ", upper: ")
	if trigger && w.ref == 0 {
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupIntervalConversion()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("bound callback acquired executing owner", e)
			}
		}
		if e := w.binary.CloseChecked(); !errors.Is(e, unumeral.ErrNumericPending) {
			w.t.Fatal("bound callback closed borrowed binary manager", e)
		}
		q := &w.m.intervalBounds
		if q.constructed != 2 || !q.executing {
			w.t.Fatal("actual lower/upper declaration frontier")
		}
		n := q.values[0].BorrowNumerator()
		switch w.kind {
		case 1:
			n = w.input.BorrowNumerator()
		case 2:
			n = w.input.BorrowDenominator()
		case 4:
			n = q.values[1].BorrowNumerator()
		}
		if n.IsSmall() {
			w.t.Fatal("actual selected operand not BIG")
		}
		z := w.m.config.Integer
		w.ref = n.Cell
		w.original = z.Cell(w.ref)
		size, e := w.original.Backing.ReadWord(z.Allocator, 0)
		serviceMust(w.t, e)
		serviceMust(w.t, z.RuntimeOwner().WriteUint32(w.raw, w.offset, size))
		projected := w.original
		projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = w.query, w.raw, w.offset
		z.Store(w.ref, projected)
		if w.refused {
			return 0, errors.New("bound trace outer stream refused")
		}
	}
	return len(p), nil
}
func TestUPolynomialConvertBoundsActualReadStops(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("TRACE erased")
	}
	for kind := 0; kind < 5; kind++ {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, refused), func(t *testing.T) {
				vm, m, _ := algebraicNegTLSFixture(t, nil)
				qm, bqm := vm.config.Rationals, vm.config.Binary
				z := m.config.Integer
				constant := -1
				left := "18446744073709551617/36893488147419103233"
				right := "55340232221128654849/36893488147419103233"
				if kind == 0 {
					constant = -2
					left = "18446744073709551617/18446744073709551616"
					right = "3"
				}
				p := upolySourceNew(t, m, []int32{int32(constant), 1})
				a, b := unumeral.NewRational(), unumeral.NewRational()
				serviceMust(t, qm.SetString(&a, left))
				serviceMust(t, qm.SetString(&b, right))
				c, d := unumeral.NewBinaryInt(99), unumeral.NewBinaryInt(88)
				t.Cleanup(func() {
					serviceMust(t, bqm.Del(&c))
					serviceMust(t, bqm.Del(&d))
					serviceMust(t, qm.Del(&a))
					serviceMust(t, qm.Del(&b))
				})
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := convertBoundsReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, binary: bqm, input: &a, kind: kind, refused: refused}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, m.CleanupIntervalConversion())
					serviceMust(t, m.CleanupConversionEntryTrace())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("convert_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				var receipt upolyConvertEntryReceipt
				ok, body := m.convertRationalToBinaryIntervalWithEntryReceipt(mustUPolyView(t, m, p), qm, &a, &b, bqm, &c, &d, &receipt)
				q := &m.intervalBounds
				slot := kind - 1
				want := uint8(0)
				if kind == 0 {
					slot = 0
					want = 1
				}
				if kind == 3 {
					want = 1
				}
				if kind == 4 {
					want = 2
				}
				if ok || !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !m.convertEntryReceiptPending(&receipt) || !q.insertions[slot].Active() || q.constructed != want {
					t.Fatal("actual bound trace/source suffix", body, q.constructed, want)
				}
				if want != 0 && !errors.Is(q.sourceCause, umemory.ErrRuntimeUninitialized) {
					t.Fatal("actual bound NUM destructor stop missing", q.sourceCause)
				}
				if want == 0 && q.sourceCause != nil {
					t.Fatal("unjoined input read failure blocked older bounds", q.sourceCause)
				}
				if c.Num != unumeral.NewInt(99) || d.Num != unumeral.NewInt(88) {
					t.Fatal("diagnostic failure crossed first bound publication")
				}
				source, original := q.sourceCause, q.bodyCause
				w.restore()
				// Later independent formatting can reuse the actual character frame.
				var laterValue unumeral.Int
				serviceMust(t, z.SetString(&laterValue, "18446744073709551619"))
				t.Cleanup(func() { serviceMust(t, z.Del(&laterValue)) })
				originalCell := z.Cell(laterValue.Cell)
				size, e := originalCell.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := originalCell
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(laterValue.Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				_, e = m.modular.SourceStringTracked(&laterValue, &later)
				z.Store(laterValue.Cell, originalCell)
				t.Cleanup(func() { serviceMust(t, m.modular.CleanupSourceString(&later)) })
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("later accepted formatter", e)
				}
				character := later.Character
				serviceMust(t, m.cleanupConvertEntryReceipt(&receipt))
				if q.active || q.constructed != want || q.sourceCause != source || q.bodyCause != original {
					t.Fatal("owning cleanup rewrote bound source history")
				}
				if !later.Active() || later.Character != character {
					t.Fatal("old bound cleanup acquired later formatter")
				}
				serviceMust(t, m.modular.CleanupSourceString(&later))
			})
		}
	}
}

type inverseBoundsReadGuard struct {
	convertBoundsReadGuard
	core *AlgebraicRootCore
}

func (w *inverseBoundsReadGuard) Write(p []byte) (int, error) {
	if w.armed && string(p) == "/" && w.ref == 0 {
		for _, e := range []error{w.core.Close(), w.core.CleanupRetained()} {
			if !errors.Is(e, ErrAlgebraicPending) {
				w.t.Fatal("bound callback acquired actual inverse caller", e)
			}
		}
		if w.core.inversion.remaining != 2 || !w.core.inversion.executing {
			w.t.Fatal("inverse source rational frontier")
		}
		w.input = &w.core.inversion.values[0]
	}
	return w.convertBoundsReadGuard.Write(p)
}
func TestAlgebraicInverseBoundsTraceActualSourceStop(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("TRACE erased")
	}
	vm, m, _ := algebraicNegTLSFixture(t, nil)
	z, bqm := m.config.Integer, vm.config.Binary
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	lo, hi := unumeral.NewBinaryIntPower(0, 64), unumeral.NewBinaryIntPower(0, 64)
	serviceMust(t, z.SetString(lo.BorrowNumerator(), "18446744073709551617"))
	serviceMust(t, z.SetString(hi.BorrowNumerator(), "36893488147419103231"))
	t.Cleanup(func() { serviceMust(t, bqm.Del(&lo)); serviceMust(t, bqm.Del(&hi)) })
	var root Algebraic
	serviceMust(t, vm.SetRoot(&root, mustUPolyView(t, m, p), &lo, &hi, true))
	t.Cleanup(func() { serviceMust(t, vm.Del(&root)) })
	allocator := z.SourceAllocator()
	query, e := allocator.Allocate(4096)
	serviceMust(t, e)
	raw, offset, e := allocator.RawLocation(query)
	serviceMust(t, e)
	w := inverseBoundsReadGuard{convertBoundsReadGuard: convertBoundsReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, binary: bqm, kind: 2}, core: vm}
	t.Cleanup(func() {
		w.restore()
		serviceMust(t, vm.CleanupRetained())
		serviceMust(t, allocator.Deallocate(4096, query))
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("convert_bug")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	body := vm.Inv(&root)
	s := &vm.inversion
	if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || m.intervalBounds.constructed != 0 || s.remaining != 1 || s.sourceRemaining != 1 || s.sourceCursor != 3 || !errors.Is(s.sourceStop, umemory.ErrRuntimeUninitialized) {
		t.Fatal("actual inverse/bound source order", body, m.intervalBounds.constructed, s.remaining, s.sourceCursor, s.sourceStop)
	}
	if !m.convertEntryReceiptPending(&s.convertEntry) || s.convertEntry.boundsInvocation == 0 {
		t.Fatal("missing exact bound child receipt")
	}
	source := s.sourceStop
	w.restore()
	serviceMust(t, vm.CleanupRetained())
	if m.intervalBounds.active || s.remaining != 0 || s.sourceCursor != 3 || s.sourceStop != source {
		t.Fatal("owning inverse cleanup changed source history")
	}
}
