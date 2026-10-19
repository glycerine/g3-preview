package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strings"
	"testing"
)

func TestUPolynomialConvertEntryTraceSourceRecord(t *testing.T) {
	for _, fractional := range []bool{false, true} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(fractional, refused), func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				p := upolySourceNew(t, m, []int32{-1, 1})
				a, b := unumeral.NewRational(), unumeral.NewRational()
				an, ad, bn, bd := int64(0), uint64(1), int64(2), uint64(1)
				if fractional {
					an, ad, bn, bd = 2, 3, 5, 4
				}
				serviceMust(t, q.SetFraction(&a, an, ad))
				serviceMust(t, q.SetFraction(&b, bn, bd))
				c, d := unumeral.NewBinaryInt(99), unumeral.NewBinaryInt(88)
				t.Cleanup(func() {
					serviceMust(t, m.CleanupConversionEntryTrace())
					serviceMust(t, bqm.Del(&c))
					serviceMust(t, bqm.Del(&d))
					serviceMust(t, q.Del(&a))
					serviceMust(t, q.Del(&b))
				})
				var text strings.Builder
				bad := &linearTraceFailWriter{}
				var out io.Writer = &text
				if refused {
					out = bad
				}
				tags := diagnostic.NewTraceTagsWithWriter(out)
				tags.Enable("convert_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				ok, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &c, &d)
				serviceMust(t, e)
				if fractional {
					if ok || upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "1" || upolyIsolationBinaryRat(t, m, bqm, &d).RatString() != "88" {
						t.Fatal("independent exact-root publication", ok)
					}
				} else if !ok || upolyIsolationBinaryRat(t, m, bqm, &c).RatString() != "0" || upolyIsolationBinaryRat(t, m, bqm, &d).RatString() != "2" {
					t.Fatal("independent binary interval", ok)
				}
				if tags.ShouldTrace("convert_bug") {
					if refused {
						if bad.calls != 1 {
							t.Fatal("nonsticky stream", bad.calls)
						}
					} else {
						want := fmt.Sprintf("-------- [convert_bug] convert_q2bq_interval src/math/polynomial/upolynomial.cpp:2823 ---------\na: %d/%d\nb: %d/%d\nsign_a: -1\nsign_b: 1\n------------------------------------------------\n", an, ad, bn, bd)
						if fractional {
							want += "-------- [convert_bug] convert_q2bq_interval src/math/polynomial/upolynomial.cpp:2841 ---------\na: 2/3\nlower: 1/2, upper: 1\n------------------------------------------------\n"
						} else {
							want += "-------- [convert_bug] convert_q2bq_interval src/math/polynomial/upolynomial.cpp:2830 ---------\nfound c: 0\n------------------------------------------------\n"
						}
						if text.String() != want {
							t.Fatal("source entry record", text.String(), want)
						}
					}
				} else if text.Len() != 0 || bad.calls != 0 {
					t.Fatal("erased entry")
				}
				if m.convertEntry.active {
					t.Fatal("completed record retained child")
				}
			})
		}
	}
}
func TestUPolynomialConvertEntryTraceAfterAssertionsAndErasure(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	tags := diagnostic.NewTraceTagsWithWriter(nil)
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	used := budget.Used()
	budget.FailAfter(0)
	e := m.traceConvertEntry(nil, nil, nil, 0, 0, nil)
	budget.FailAfter(-1)
	serviceMust(t, e)
	if used != budget.Used() || m.convertEntry.invocation != 0 {
		t.Fatal("erased trace admitted/evaluated missing operands")
	}
	if !polynomialSourceDebug {
		return
	}
	p := upolySourceNew(t, m, []int32{-1, 1})
	a, b := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetInt64(&a, 1))
	serviceMust(t, q.SetInt64(&b, 2))
	t.Cleanup(func() { serviceMust(t, q.Del(&a)); serviceMust(t, q.Del(&b)) })
	c, d := unumeral.NewBinaryInt(99), unumeral.NewBinaryInt(88)
	t.Cleanup(func() { serviceMust(t, bqm.Del(&c)); serviceMust(t, bqm.Del(&d)) })
	var text strings.Builder
	tags = diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("convert_bug")
	serviceMust(t, m.modular.BindTrace(tags))
	polynomialAssertionActionsFixture(t, "CT")
	ok, e := m.ConvertRationalToBinaryInterval(mustUPolyView(t, m, p), q, &a, &b, bqm, &c, &d)
	if ok || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || text.Len() != 0 || m.convertEntry.invocation != 0 {
		t.Fatal("second entry assertion must precede trace admission", ok, e, text.String())
	}
}

type convertEntryTraceReadGuard struct {
	craTraceReadGuard
	qm      *unumeral.RationalManager
	binary  *unumeral.BinaryRationalManager
	values  [4]*unumeral.Int
	index   int
	slashes int
	refused bool
}

func (w *convertEntryTraceReadGuard) Write(p []byte) (int, error) {
	text := string(p)
	if text == "/" {
		w.slashes++
	}
	trigger := w.index == 0 && text == "a: " || w.index == 1 && text == "/" && w.slashes == 1 || w.index == 2 && text == "\nb: " || w.index == 3 && text == "/" && w.slashes == 2
	if trigger && w.ref == 0 {
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupConversionEntryTrace()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("entry callback acquired polynomial owner", e)
			}
		}
		for _, e := range []error{w.qm.CloseChecked(), w.binary.CloseChecked()} {
			if !errors.Is(e, unumeral.ErrNumericPending) {
				w.t.Fatal("entry callback closed borrowed numeric owner", e)
			}
		}
		n := w.values[w.index]
		if n.IsSmall() {
			w.t.Fatal("actual operand not BIG")
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
			return 0, errors.New("conversion entry outer trace refused")
		}
	}
	return len(p), nil
}
func TestUPolynomialConvertEntryTraceActualReadStops(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	for index := 0; index < 4; index++ {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(index, refused), func(t *testing.T) {
				vm, m, _ := algebraicNegTLSFixture(t, nil)
				q, bqm := vm.config.Rationals, vm.config.Binary
				z := m.config.Integer
				p := upolySourceNew(t, m, []int32{-1, 1})
				a, b := unumeral.NewRational(), unumeral.NewRational()
				serviceMust(t, q.SetString(&a, "18446744073709551617/36893488147419103233"))
				serviceMust(t, q.SetString(&b, "55340232221128654849/36893488147419103233"))
				c, d := unumeral.NewBinaryInt(99), unumeral.NewBinaryInt(88)
				t.Cleanup(func() {
					serviceMust(t, bqm.Del(&c))
					serviceMust(t, bqm.Del(&d))
					serviceMust(t, q.Del(&a))
					serviceMust(t, q.Del(&b))
				})
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := convertEntryTraceReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, qm: q, binary: bqm, values: [4]*unumeral.Int{a.BorrowNumerator(), a.BorrowDenominator(), b.BorrowNumerator(), b.BorrowDenominator()}, index: index, refused: refused}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, m.CleanupConversionEntryTrace())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				var ended upolyConvertEntryReceipt
				if index == 0 && !refused {
					warm := diagnostic.NewTraceTagsWithWriter(io.Discard)
					warm.Enable("convert_bug")
					serviceMust(t, m.modular.BindTrace(warm))
					// A completed concrete diagnostic invocation supplies its own receipt.
					serviceMust(t, m.traceConvertEntry(bqm, &a, &b, -1, 1, &ended))
				}
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("convert_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				var receipt upolyConvertEntryReceipt
				ok, body := m.convertRationalToBinaryIntervalWithEntryReceipt(mustUPolyView(t, m, p), q, &a, &b, bqm, &c, &d, &receipt)
				s := &m.convertEntry
				if ok || !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !m.convertEntryReceiptPending(&receipt) || !s.insertions[index].Active() {
					t.Fatal("actual entry operand read", body, w.ref)
				}
				if c.Num != unumeral.NewInt(99) || d.Num != unumeral.NewInt(88) {
					t.Fatal("entry failure crossed bound publication")
				}
				originalBody, originalSource := s.bodyCause, s.sourceCause
				if ended.owner != nil {
					character := s.insertions[index].Character
					serviceMust(t, m.cleanupConvertEntryReceipt(&ended))
					if !s.active || !s.insertions[index].Active() || s.insertions[index].Character != character {
						t.Fatal("ended entry receipt acquired later invocation")
					}
				}
				w.restore()
				// Reuse the genuine formatter frame independently before old cleanup.
				var value unumeral.Int
				serviceMust(t, z.SetString(&value, "18446744073709551619"))
				t.Cleanup(func() { serviceMust(t, z.Del(&value)) })
				original := z.Cell(value.Cell)
				size, e := original.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := original
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(value.Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				_, e = m.modular.SourceStringTracked(&value, &later)
				z.Store(value.Cell, original)
				t.Cleanup(func() { serviceMust(t, m.modular.CleanupSourceString(&later)) })
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("later accepted formatter", e)
				}
				character := later.Character
				serviceMust(t, m.cleanupConvertEntryReceipt(&receipt))
				if s.active || s.bodyCause != originalBody || s.sourceCause != originalSource {
					t.Fatal("owning cleanup changed source history")
				}
				if !later.Active() || later.Character != character {
					t.Fatal("cleanup adopted independent formatting generation")
				}
				serviceMust(t, m.modular.CleanupSourceString(&later))
			})
		}
	}
}

type inverseConvertEntryGuard struct {
	convertEntryTraceReadGuard
	core *AlgebraicRootCore
}

func (w *inverseConvertEntryGuard) Write(p []byte) (int, error) {
	if string(p) == "a: " && w.ref == 0 {
		for _, e := range []error{w.core.Close(), w.core.CleanupRetained()} {
			if !errors.Is(e, ErrAlgebraicPending) {
				w.t.Fatal("entry callback acquired actual algebraic caller", e)
			}
		}
		s := &w.core.inversion
		if s.remaining != 2 || !s.executing || !w.m.convertEntryReceiptPending(&s.convertEntry) {
			w.t.Fatal("actual inverse rational construction/receipt frontier")
		}
		w.values = [4]*unumeral.Int{s.values[0].BorrowNumerator(), s.values[0].BorrowDenominator(), s.values[1].BorrowNumerator(), s.values[1].BorrowDenominator()}
	}
	return w.convertEntryTraceReadGuard.Write(p)
}
func TestAlgebraicInverseConversionEntryActualSourceStop(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			vm, m, _ := algebraicNegTLSFixture(t, nil)
			z := m.config.Integer
			bqm := vm.config.Binary
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
			w := inverseConvertEntryGuard{convertEntryTraceReadGuard: convertEntryTraceReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, qm: vm.config.Rationals, binary: bqm, index: 0, refused: refused}, core: vm}
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
			if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || s.remaining != 1 || s.sourceRemaining != 1 || s.sourceCursor != 2 || !errors.Is(s.sourceStop, umemory.ErrRuntimeUninitialized) || !m.convertEntryReceiptPending(&s.convertEntry) {
				t.Fatal("actual caller reverse source destruction", body, s.remaining, s.sourceStop)
			}
			// The younger inv_upper really ended; inv_lower stops at its actual unreadable
			// numerator. No scalar value, construction count or quota is rewritten.
			if s.values[1].BorrowNumerator().NativeCellToken() != 0 || s.values[1].BorrowDenominator().NativeCellToken() != 0 {
				t.Fatal("younger rational survived source destruction")
			}
			original := s.sourceStop
			w.restore()
			serviceMust(t, vm.CleanupRetained())
			if m.convertEntry.active || s.remaining != 0 || s.sourceRemaining != 1 || s.sourceCursor != 2 || s.sourceStop != original {
				t.Fatal("owning caller cleanup lost exact child/history")
			}
		})
	}
}
