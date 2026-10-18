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

func TestUPolynomialRefinableTraceSourceRecords(t *testing.T) {
	for _, f := range []struct {
		name   string
		p      []int32
		a, b   int32
		want   []string
		count  int
		ok     bool
		wa, wb string
	}{
		{"bracket", []int32{-1, 0, 1}, 0, 2, []string{"sign_a: -1, sign_b: 1\n"}, 1, true, "0", "2"},
		{"left", []int32{0, -1, 1}, 0, 3, []string{"sign_a: 0, sign_b: 1\n", "CASE 2, a: 0, b: 3, new_a: 3/2\n", "CASE 2, a: 0, b: 3/2, new_a: 3/2^2\n"}, 3, true, "3/4", "3/2"},
		{"right", []int32{0, -1, 1}, -2, 1, []string{"sign_a: 1, sign_b: 0\n", "CASE 3, a: -2, b: 1, new_b: -1/2\n", "CASE 3, a: -1/2, b: 1, new_b: 1/2^2\n"}, 3, true, "-1/2", "1/4"},
		{"both", []int32{0, 87, -59, 10}, 0, 3, []string{"sign_a: 0, sign_b: 0\n", "CASE 4\na1: 0, b1: 3/2, new_a1: 3/2^2\na2: 3/2, b2: 3, new_b2: 9/2^2\n", "CASE 4\na1: 0, b1: 3/2^4, new_a1: 3/2^5\na2: 45/2^4, b2: 3, new_b2: 93/2^5\n"}, 5, true, "45/16", "93/32"},
		{"mid-root", []int32{-6, 11, -6, 1}, 1, 3, []string{"sign_a: 0, sign_b: 0\n"}, 1, false, "2", "3"},
	} {
		for _, refused := range []bool{false, true} {
			t.Run(f.name+fmt.Sprint(refused), func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				bqm := upolyIsolationBinaryOwner(t, q, budget)
				p := upolySourceNew(t, m, f.p)
				a, b := unumeral.NewBinaryInt(f.a), unumeral.NewBinaryInt(f.b)
				t.Cleanup(func() { serviceMust(t, m.CleanupRefinable()); serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
				var text strings.Builder
				bad := &linearTraceFailWriter{}
				var out io.Writer = &text
				if refused {
					out = bad
				}
				tags := diagnostic.NewTraceTagsWithWriter(out)
				tags.Enable("upolynomial")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				ok, e := m.IsolatingToRefinable(mustUPolyView(t, m, p), bqm, &a, &b)
				serviceMust(t, e)
				if ok != f.ok || upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != f.wa || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != f.wb {
					t.Fatal("independent endpoint oracle", ok)
				}
				if tags.ShouldTrace("upolynomial") {
					if refused {
						if bad.calls != 1 {
							t.Fatal("nonsticky output", bad.calls)
						}
					} else {
						for _, want := range f.want {
							if !strings.Contains(text.String(), want) {
								t.Fatal("missing source record", want, text.String())
							}
						}
						if strings.Count(text.String(), "isolating2refinable src/math/polynomial/upolynomial.cpp:") != f.count {
							t.Fatal("record count", f.count, text.String())
						}
					}
				} else if text.Len() != 0 || bad.calls != 0 {
					t.Fatal("erased TRACE")
				}
				if m.refinable.active || m.refinable.constructed != 0 {
					t.Fatal("completed source locals retained")
				}
			})
		}
	}
}
func TestUPolynomialRefinableActualReservationPrefixes(t *testing.T) {
	success, failed := false, false
	for n := int64(0); n < 512 && !success; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			vm, m, _ := algebraicNegTLSFixture(t, nil)
			q := m.config.Integer
			budget := q.Budget()
			bqm := vm.config.Binary
			p := upolySourceNew(t, m, nil)
			var negativeRoot unumeral.Int
			serviceMust(t, q.SetString(&negativeRoot, "-18446744073709551616"))
			serviceMust(t, m.SetInts(p, []unumeral.Int{unumeral.NewInt(0), negativeRoot, unumeral.NewInt(1)}))
			serviceMust(t, q.Del(&negativeRoot))
			a, b := unumeral.NewBinaryInt(0), unumeral.NewBinaryRational()
			serviceMust(t, q.SetString(b.BorrowNumerator(), "55340232221128654848"))
			t.Cleanup(func() {
				budget.FailAfter(-1)
				serviceMust(t, m.CleanupRefinable())
				serviceMust(t, bqm.Del(&a))
				serviceMust(t, bqm.Del(&b))
			})
			v := mustUPolyView(t, m, p)
			budget.FailAfter(n)
			ok, body := m.IsolatingToRefinable(v, bqm, &a, &b)
			budget.FailAfter(-1)
			if body == nil {
				success = true
				if !ok || upolyIsolationBinaryRat(t, m, bqm, &a).RatString() != "13835058055282163712" || upolyIsolationBinaryRat(t, m, bqm, &b).RatString() != "27670116110564327424" {
					t.Fatal("independent scaled interval")
				}
			} else {
				failed = true
				if !numericStorageReservationRefusal(body) {
					t.Fatal("unexpected refusal", body)
				}
			}
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{BinaryManagers: []*unumeral.BinaryRationalManager{bqm}, Integers: []*unumeral.Int{a.BorrowNumerator(), b.BorrowNumerator()}})
			s := &m.refinable
			constructed, source, original := s.constructed, s.sourceCause, s.bodyCause
			serviceMust(t, m.CleanupRefinable())
			if s.active || s.constructed != constructed || s.sourceCause != source || s.bodyCause != original {
				t.Fatal("cleanup rewrote source history")
			}
		})
	}
	if !success || !failed {
		t.Fatal("prefix campaign incomplete", success, failed)
	}
}

type refinableTraceReadGuard struct {
	craTraceReadGuard
	binary    *unumeral.BinaryRationalManager
	endpoint  *unumeral.BinaryRational
	temporary bool
	refused   bool
}

func (w *refinableTraceReadGuard) Write(p []byte) (int, error) {
	caption := ", b: "
	if w.temporary {
		caption = ", new_a: "
	}
	if string(p) == caption && w.ref == 0 {
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupRefinable()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("callback acquired refinable owner", e)
			}
		}
		if e := w.binary.CloseChecked(); !errors.Is(e, unumeral.ErrNumericPending) {
			w.t.Fatal("callback closed borrowed binary owner", e)
		}
		n := w.endpoint.BorrowNumerator()
		if w.temporary {
			n = w.m.refinable.values[0].BorrowNumerator()
		}
		if n.IsSmall() || w.m.refinable.constructed != 1 {
			w.t.Fatal("actual case2 constructor/operand frontier")
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
			return 0, errors.New("refinable outer trace refused")
		}
	}
	return len(p), nil
}
func TestUPolynomialRefinableTraceActualReadStops(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("TRACE erased")
	}
	for _, temporary := range []bool{false, true} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(temporary, refused), func(t *testing.T) {
				vm, m, _ := algebraicNegTLSFixture(t, nil)
				z := m.config.Integer
				bqm := vm.config.Binary
				p := upolySourceNew(t, m, nil)
				var negativeRoot unumeral.Int
				serviceMust(t, z.SetString(&negativeRoot, "-18446744073709551617"))
				serviceMust(t, m.SetInts(p, []unumeral.Int{unumeral.NewInt(0), negativeRoot, unumeral.NewInt(1)}))
				serviceMust(t, z.Del(&negativeRoot))
				a, b := unumeral.NewBinaryInt(0), unumeral.NewBinaryRational()
				serviceMust(t, z.SetString(b.BorrowNumerator(), "36893488147419103234"))
				t.Cleanup(func() { serviceMust(t, bqm.Del(&a)); serviceMust(t, bqm.Del(&b)) })
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := refinableTraceReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, binary: bqm, endpoint: &b, temporary: temporary, refused: refused}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, m.CleanupRefinable())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("upolynomial")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				_, body := m.IsolatingToRefinable(mustUPolyView(t, m, p), bqm, &a, &b)
				s := &m.refinable
				index := 1
				want := uint8(0)
				if temporary {
					index = 2
					want = 1
				}
				if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !s.active || !s.insertions[index].Active() || s.constructed != want {
					t.Fatal("actual operand/source suffix", body, s.constructed, want)
				}
				if temporary && !errors.Is(s.sourceCause, umemory.ErrRuntimeUninitialized) {
					t.Fatal("temporary NUM destructor did not stop", s.sourceCause)
				}
				if !temporary && s.sourceCause != nil {
					t.Fatal("unjoined endpoint failure stopped older source NUM", s.sourceCause)
				}
				originalBody, originalSource := s.bodyCause, s.sourceCause
				w.restore()
				// A later independent numeral string may reuse the formatter frame. Old
				// refinable cleanup has authority only over its own accepted receipt.
				var laterValue unumeral.Int
				serviceMust(t, z.SetString(&laterValue, "18446744073709551619"))
				t.Cleanup(func() { serviceMust(t, z.Del(&laterValue)) })
				original := z.Cell(laterValue.Cell)
				size, e := original.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := original
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(laterValue.Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				var text strings.Builder
				e = z.InsertSourceIntegerTracked(factorizationTraceStream{&text}, &laterValue, &later, nil)
				z.Store(laterValue.Cell, original)
				t.Cleanup(func() { serviceMust(t, m.modular.CleanupSourceString(&later)) })
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("later accepted formatter", e)
				}
				character := later.Character
				serviceMust(t, m.CleanupRefinable())
				if s.active || s.constructed != want || s.bodyCause != originalBody || s.sourceCause != originalSource {
					t.Fatal("owning cleanup changed source history")
				}
				if !later.Active() || later.Character != character {
					t.Fatal("cleanup acquired independent character generation")
				}
				serviceMust(t, m.modular.CleanupSourceString(&later))
			})
		}
	}
}

func TestUPolynomialRefinableTraceErasureAndEndedReceipt(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	bqm := upolyIsolationBinaryOwner(t, q, budget)
	tags := diagnostic.NewTraceTagsWithWriter(nil)
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	var old upolyRefinableInvocationReceipt
	first, e := m.beginRefinable(bqm, &old)
	serviceMust(t, e)
	used := budget.Used()
	budget.FailAfter(0)
	// No source temporaries or operands are present; erased TRACE must not read them.
	e = first.traceInterval(2693, [6]*unumeral.BinaryRational{})
	budget.FailAfter(-1)
	serviceMust(t, e)
	if budget.Used() != used || first.constructed != 0 {
		t.Fatal("erased trace constructed/evaluated operands")
	}
	serviceMust(t, first.finish(nil))
	later, e := m.beginRefinable(bqm, nil)
	serviceMust(t, e)
	generation := later.invocation
	serviceMust(t, m.cleanupRefinableReceipt(&old))
	if !later.active || !later.executing || later.invocation != generation {
		t.Fatal("ended receipt acquired later source invocation")
	}
	serviceMust(t, later.finish(nil))
}
