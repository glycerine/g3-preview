// Source: algebraic_numbers.cpp:2721-2747; mpbq.cpp:426-458.
package polynomial

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicDecimalRootReferenceDoesNotReadFlags(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	defer func() { serviceMust(t, p.DelPolynomial(&poly)) }()
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for i, w := range []uint32{uint32(poly.origin), uint32(poly.origin >> 32), poly.slot, poly.generation} {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 4*uint64(i), w))
	}
	serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 20, 3))
	for i, w := range []uint32{1, 0, 0, 0, 2, 0, 0, 0} {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 24+4*uint64(i), w))
	}
	var out bytes.Buffer
	if e := c.DisplayDecimal(&out, &a, 0); e != nil || out.String() != "1.?" {
		t.Error("decimal root reached unrelated fields", e, out.String())
	}
	if _, e := c.config.Allocator.ReadUint32(l.raw, 16); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Error("raw view acquired packed flags", e)
	}
	serviceMust(t, c.CleanupRetained())
}

func TestAlgebraicDecimalRefinementSourceStopRetainsBounds(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 4096 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var c *AlgebraicRootCore
			var rt *umemory.Runtime
			var lane, pad umemory.Block
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || !c.decimalDisplayRunning || !c.decimalDisplay.refinement.executing {
					return f
				}
				z := c.config.Rationals.Integers()
				if z.AddMulInvocation.Tmp.Cell == 0 || z.IsFailureRef(z.AddMulInvocation.Tmp.Cell) {
					return f
				}
				armed = false
				reached = true
				serviceMust(t, rt.SetMaxSize(0))
				lane = algebraicCellDeletionTLSLane(t, rt)
				return f
			}
			c, _, rt = algebraicNegTLSFixture(t, transport)
			root := algebraicBigBinaryBoundsRoot(t, c)
			// This raw source view owns only its physical header; it claims no C++
			// constructor or numeral/polynomial member ownership.
			header, e := c.config.Allocator.Allocate(56)
			serviceMust(t, e)
			c.locations = append(c.locations, algebraicLocation{raw: header, kind: 1, generation: 1, live: true})
			prefix := Algebraic{owner: c.origin, slot: uint32(len(c.locations)), generation: 1}
			for i, w := range []uint32{7, 0, 0} {
				serviceMust(t, c.config.Allocator.WriteUint32(header, 24+4*uint64(i), w))
			}
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, c.CleanupRetained)
				contextTLSFinish(t, rt, func() error {
					if prefix.slot == 0 {
						return nil
					}
					l, e := c.locationCleanup(&prefix)
					if e != nil {
						return e
					}
					return c.deleteLocation(&prefix, l)
				})
				contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
				for _, block := range []*umemory.Block{&lane, &pad} {
					contextTLSFinish(t, rt, func() error {
						if *block == (umemory.Block{}) {
							return nil
						}
						return rt.Deallocate(block)
					})
				}
			}()
			n := int64(100000) - rt.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal(rt.State())
			}
			pad, e = rt.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			body := c.DisplayDecimal(io.Discard, &root, 0)
			armed = false
			serviceMust(t, rt.SetMaxSize(0))
			if !reached {
				return
			}
			s := &c.decimalDisplay
			if !errors.Is(body, arena.ErrBounds) || s.refinement.evaluationSourceCause(c) == nil || s.remaining != 2 || s.sourceStop == nil {
				t.Fatal("actual decimal Horner source stop", body, s.remaining)
			}
			z := c.config.Rationals.Integers()
			for i := range s.values {
				n := s.values[i].BorrowNumerator()
				if n.Cell == 0 || z.Cell(n.Cell).Released {
					t.Error("older decimal bound destroyed", i)
				}
			}
			values, stats, used := s.values, rt.Stats(), z.Budget().Used()
			if e := c.finishDecimalDisplay(false); !errors.Is(e, s.sourceStop) {
				t.Error("source replay lost younger Horner stop", e)
			}
			if values != s.values || stats != rt.Stats() || used != z.Budget().Used() {
				t.Error("source replay changed retained decimal graph")
			}
			contextTLSFinish(t, rt, c.CleanupRetained)
			invocation := s.refinement.invocation
			body = c.DisplayDecimal(io.Discard, &prefix, 0)
			if !errors.Is(body, umemory.ErrRuntimeUninitialized) || s.sourceStop != nil || s.remaining != 0 || invocation != s.refinement.invocation {
				t.Error("new partial bound copy adopted retired Horner source stop", body, s.sourceStop, s.remaining)
			}
			t.Logf("actual decimal refinement stop at offset %d", offset)
		})
	}
	if !reached {
		t.Fatal("actual decimal refinement transport not reached")
	}
}

func TestAlgebraicDecimalDisplayAndBoundDestructorStops(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(fmt.Sprint("raw=", raw), func(t *testing.T) {
			c, p, rt := algebraicNegTLSFixture(t, nil)
			var root Algebraic
			if raw {
				root = algebraicBigBinaryBoundsRoot(t, c)
			} else {
				z, b := c.config.Rationals.Integers(), c.config.Binary
				var n, negative unumeral.Int
				serviceMust(t, z.SetString(&n, "18446744073709551617"))
				serviceMust(t, z.Mul(&n, &n, &negative))
				serviceMust(t, z.Neg(&negative))
				poly, e := p.NewPolynomial()
				serviceMust(t, e)
				serviceMust(t, p.SetInts(poly, []unumeral.Int{negative, unumeral.NewInt(0), unumeral.NewInt(1)}))
				view, e := p.wholeView(poly)
				serviceMust(t, e)
				var lower, upper unumeral.BinaryRational
				one := unumeral.NewInt(1)
				serviceMust(t, b.SetInteger(&lower, &n))
				serviceMust(t, b.SetInteger(&upper, &n))
				serviceMust(t, b.SubInteger(&lower, &one, &lower))
				serviceMust(t, b.AddInteger(&upper, &one, &upper))
				serviceMust(t, c.SetRoot(&root, view, &lower, &upper, false))
				serviceMust(t, b.Del(&upper))
				serviceMust(t, b.Del(&lower))
				serviceMust(t, p.DelPolynomial(&poly))
				serviceMust(t, z.Del(&negative))
				serviceMust(t, z.Del(&n))
			}
			var lane umemory.Block
			called := false
			writer := &algebraicDisplayCallbackWriter{callback: func() { called = true; lane = algebraicCellDeletionTLSLane(t, rt) }}
			body := c.DisplayDecimal(writer, &root, 0)
			wantText := "18446744073709551617"
			if raw {
				wantText = "1.?"
			}
			if writer.String() != wantText {
				t.Error("selected source decimal bound", writer.String(), wantText)
			}
			defer func() {
				contextTLSFinish(t, rt, c.CleanupRetained)
				contextTLSFinish(t, rt, func() error { return c.CleanupValue(&root) })
				algebraicCellDeletionFinishLane(t, rt, &lane)
			}()
			s, z := &c.decimalDisplay, c.config.Rationals.Integers()
			if !called || !errors.Is(body, arena.ErrBounds) || s.remaining != 2 {
				t.Fatal("actual decimal release guard", called, body, s.remaining)
			}
			upper, lower := s.values[1].BorrowNumerator(), s.values[0].BorrowNumerator()
			if upper.Cell == 0 || lower.Cell == 0 || z.Cell(lower.Cell).Released {
				t.Fatal("older lower bound lost")
			}
			if raw {
				if z.Cell(upper.Cell).Released || s.displayStop == nil || !c.config.Binary.DisplayInvocationPending(&s.display) {
					t.Error("raw END stop crossed older upper bound")
				}
			} else if !z.Cell(upper.Cell).Released || s.displayStop != nil {
				t.Error("upper scoped DT did not stop before lower")
			}
			values, stats, used := s.values, rt.Stats(), z.Budget().Used()
			again := c.finishDecimalDisplay(false)
			want := s.displayStop
			if !raw {
				want = arena.ErrInvalidSlot
			}
			if !errors.Is(again, want) || values != s.values || stats != rt.Stats() || used != z.Budget().Used() {
				t.Error("literal source replay crossed release stop", again, want)
			}
		})
	}
}

func TestAlgebraicDecimalBodyFailureAndReplacementDisplay(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	defer func() { serviceMust(t, c.Del(&root)) }()
	b := c.config.Binary
	refusal := errors.New("writer body failure")
	writer := &algebraicDisplayCallbackWriter{callback: func() {
		s := &c.decimalDisplay
		before := s.values
		for _, call := range []func() error{func() error { return b.CleanupDisplayInvocation(&s.display) }, b.CleanupDisplay, b.CloseChecked, func() error { return b.Del(&s.values[0]) }} {
			if e := call(); !errors.Is(e, unumeral.ErrBinaryDisplayPending) {
				t.Error("writer callback acquired active numeric display", e)
			}
		}
		if before != s.values {
			t.Error("refused callback changed scoped bounds")
		}
	}}
	// This writer finishes normally, proving the refusal guards preserve output.
	serviceMust(t, c.DisplayDecimal(writer, &root, 1))
	if writer.String() != "1.4?" {
		t.Error("completed decimal output", writer.String())
	}
	stopped := &algebraicIntervalRefusingWriter{stop: 1, err: refusal}
	body := c.DisplayDecimal(stopped, &root, 1)
	s := &c.decimalDisplay
	if !errors.Is(body, refusal) || s.remaining != 0 || s.sourceStop != nil || !b.DisplayInvocationPending(&s.display) {
		t.Fatal("body error must destroy scoped bounds without executing raw END", body, s.remaining, s.sourceStop)
	}
	serviceMust(t, b.CleanupDisplay())
	value := unumeral.NewBinaryIntPower(3, 3)
	var replacement unumeral.BinaryDisplayInvocationReceipt
	body = b.DisplayDecimalToWithInvocation(&algebraicIntervalRefusingWriter{stop: 1, err: refusal}, &value, 1, &replacement)
	if !errors.Is(body, refusal) || !b.DisplayInvocationPending(&replacement) {
		t.Fatal("actual replacement display not reached", body)
	}
	state, used := b.DisplayState(), c.config.Rationals.Budget().Used()
	serviceMust(t, c.CleanupRetained())
	if !b.DisplayInvocationPending(&replacement) || state != b.DisplayState() || used != c.config.Rationals.Budget().Used() {
		t.Error("old decimal caller acquired replacement display")
	}
	serviceMust(t, b.CleanupDisplayInvocation(&replacement))
}
