package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"strconv"
	"testing"
)

// Qualified malformed DEN requested-size guard reached after an actual mutable
// destination-copy refusal. The source NUM release must precede DEN's stop.
func TestAlgebraicConstRationalSourceDestructorJoinsBodyAndStops(t *testing.T) {
	reached := false
	for offset := int64(0); offset < 1024 && !reached; offset += 4 {
		t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
			var c *AlgebraicRootCore
			armed := false
			var denRequested uint64
			var numRef, denRef unumeral.NumericCellRef
			transport := func(f umemory.Failure) error {
				if !armed || c == nil || !c.constQ.running || c.constQ.executing {
					return f
				}
				z := c.config.Rationals.Integers()
				num, den := c.localQ.BorrowNumerator().Cell, c.localQ.BorrowDenominator().Cell
				if !unumeral.ValidNumericRef(z.Cells, num) || !unumeral.ValidNumericRef(z.Cells, den) {
					return f
				}
				armed = false
				reached = true
				numRef, denRef = num, den
				for _, e := range []error{c.Close(), c.CleanupRetained(), c.cleanupConstQ(), c.finishConstQ(nil)} {
					if !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("mutable setter callback obtained const scope lifecycle", e)
					}
				}
				cell := z.Cell(den)
				denRequested = cell.Backing.Requested
				cell.Backing.Requested = uint64(^uint32(0)) + 1
				z.Store(den, cell)
				return f
			}
			var runtime *umemory.Runtime
			c, _, runtime = algebraicNegTLSFixture(t, transport)
			q, z := c.config.Rationals, c.config.Rationals.Integers()
			input := unumeral.NewRational()
			serviceMust(t, q.SetString(&input, "18446744073709551617/18446744073709551619"))
			original := input
			var out Algebraic
			one := unumeral.NewInt(1)
			serviceMust(t, c.SetInteger(&out, &one))
			n := int64(100000) - runtime.State().LocalSize - offset - 8
			if n <= 0 {
				t.Fatal("TLS extent")
			}
			pad, e := runtime.Allocate(uint64(n))
			serviceMust(t, e)
			serviceMust(t, runtime.SetMaxSize(1))
			armed = true
			body := c.SetRational(&out, &input)
			armed = false
			serviceMust(t, runtime.SetMaxSize(0))
			s := &c.constQ
			if input != original {
				t.Fatal("const input changed")
			}
			if reached {
				if !errors.Is(body, arena.ErrBounds) || !numericPhysicalReservationRefusal(body) || s.bodyCause == nil || !errors.Is(body, s.bodyCause) || s.sourceStop == nil || s.sourceCursor != 1 || !s.active || !c.localActive || unumeral.ValidNumericRef(z.Cells, numRef) || c.localQ.BorrowNumerator().Cell != 0 || c.localQ.BorrowDenominator().Cell != denRef || z.Cell(denRef).Released {
					t.Fatal("const source NUM/DEN frontier", body, s.sourceCursor)
				}
				state, used, stats := *s, z.Budget().Used(), runtime.Stats()
				if e := c.finishConstQ(nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("const source stop lost", e)
				}
				if e := c.finishQ(nil); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("generic finalizer acquired const scope", e)
				}
				if !reflect.DeepEqual(state, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
					t.Fatal("const source finalizer replayed")
				}
				var untouched Algebraic
				if e := c.SetRational(&untouched, &input); !errors.Is(e, ErrAlgebraicPending) || untouched != (Algebraic{}) {
					t.Fatal("stopped const scope replaced", e)
				}
				cell := z.Cell(denRef)
				cell.Backing.Requested = denRequested
				z.Store(denRef, cell)
			}
			serviceMust(t, runtime.Deallocate(&pad))
			serviceMust(t, c.CleanupRetained())
			if reached && (s.active || c.localActive || s.sourceCursor != 1 || s.ownCursor != 2 || s.sourceStop == nil) {
				t.Fatal("own cleanup moved const source cursor")
			}
			serviceMust(t, c.CleanupValue(&out))
			serviceMust(t, q.Del(&input))
		})
	}
	if !reached {
		t.Fatal("actual mutable copy/refused source DEN destruction not reached")
	}
}

func TestAlgebraicConstRationalSignedHostSourceStop(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		t.Run(strconv.FormatBool(wrapped), func(t *testing.T) {
			reached := false
			for offset := int64(0); offset < 2048 && !reached; offset += 4 {
				t.Run(strconv.FormatInt(offset, 10), func(t *testing.T) {
					var c *AlgebraicRootCore
					var runtime *umemory.Runtime
					var lane umemory.Block
					armed := false
					var numRef unumeral.NumericCellRef
					transport := func(f umemory.Failure) error {
						if !armed || c == nil || !c.constQ.running || c.constQ.executing {
							return f
						}
						z := c.config.Rationals.Integers()
						ref := c.localQ.BorrowNumerator().Cell
						if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
							return f
						}
						armed = false
						reached = true
						numRef = ref
						serviceMust(t, runtime.SetMaxSize(0))
						contextTLSNearMinimum(t, runtime)
						var e error
						lane, e = runtime.Allocate(8)
						serviceMust(t, e)
						delta := int64(-1000000)
						if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
							t.Fatal(e)
						}
						return f
					}
					c, _, runtime = algebraicNegTLSFixture(t, transport)
					q, z := c.config.Rationals, c.config.Rationals.Integers()
					input := unumeral.NewRational()
					serviceMust(t, q.SetString(&input, "18446744073709551617/18446744073709551619"))
					original := input
					var out Algebraic
					one := unumeral.NewInt(1)
					serviceMust(t, c.SetInteger(&out, &one))
					n := int64(100000) - runtime.State().LocalSize - offset - 8
					if n <= 0 {
						t.Fatal("TLS extent")
					}
					pad, e := runtime.Allocate(uint64(n))
					serviceMust(t, e)
					serviceMust(t, runtime.SetMaxSize(1))
					armed = true
					var body error
					if wrapped {
						body = c.SetInteger(&out, &z.Two64)
					} else {
						body = c.SetRational(&out, &input)
					}
					armed = false
					serviceMust(t, runtime.SetMaxSize(0))
					s := &c.constQ
					if input != original {
						t.Fatal("const input changed")
					}
					if reached {
						if !errors.Is(body, arena.ErrBounds) || !numericPhysicalReservationRefusal(body) || s.bodyCause == nil || s.sourceStop == nil || s.sourceCursor != 0 || !s.active || !z.Cell(numRef).Released {
							t.Fatal("actual const NUM deallocation guard", body, s.sourceCursor)
						}
						if wrapped {
							outer := &c.integerSet
							ref := outer.value.BorrowNumerator().Cell
							if !outer.active || outer.sourceStop == nil || outer.sourceCursor != 0 || !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
								t.Fatal("const stop destroyed older integer wrapper")
							}
							if e := c.cleanupIntegerSet(); !errors.Is(e, ErrAlgebraicPending) {
								t.Fatal("outer integer wrapper cleaned before const child", e)
							}
						} else {
							ref := c.localQ.BorrowDenominator().Cell
							if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
								t.Fatal("DEN destroyed after NUM stop")
							}
						}
						before, used, stats := *s, z.Budget().Used(), runtime.Stats()
						if e := c.finishConstQ(nil); !errors.Is(e, arena.ErrBounds) {
							t.Fatal(e)
						}
						if !reflect.DeepEqual(before, *s) || used != z.Budget().Used() || stats != runtime.Stats() {
							t.Fatal("const NUM source stop replayed physical release")
						}
					}
					contextTLSFinish(t, runtime, c.CleanupRetained)
					if reached && (s.active || c.localActive || s.sourceCursor != 0 || s.ownCursor != 2 || s.sourceStop == nil || wrapped && c.integerSet.active) {
						t.Fatal("const owning cleanup moved source cursor")
					}
					contextTLSFinish(t, runtime, func() error { return c.CleanupValue(&out) })
					contextTLSFinish(t, runtime, func() error { return q.Del(&input) })
					for _, block := range []*umemory.Block{&lane, &pad} {
						contextTLSFinish(t, runtime, func() error {
							if *block == (umemory.Block{}) {
								return nil
							}
							return runtime.Deallocate(block)
						})
					}
				})
			}
			if !reached {
				t.Fatal("actual const NUM accounting stop not reached")
			}
		})
	}
}

func TestAlgebraicConstRationalCopyCallbackAndExactAcknowledgement(t *testing.T) {
	var c *AlgebraicRootCore
	reached := false
	transport := func(f umemory.Failure) error {
		if c == nil || !c.constQ.running || !c.constQ.executing {
			return f
		}
		reached = true
		if !c.config.Rationals.Integers().MatchesCopyAllocationInvocation(&c.constQ.copy) {
			t.Fatal("copy callback lacks accepted const receipt")
		}
		var untouched Algebraic
		one := unumeral.NewInt(1)
		for _, e := range []error{c.Close(), c.CleanupRetained(), c.cleanupConstQ(), c.finishConstQ(nil), c.SetInteger(&untouched, &one)} {
			if !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("const copy callback obtained lifecycle", e)
			}
		}
		if untouched != (Algebraic{}) {
			t.Fatal("const copy callback published output")
		}
		return f
	}
	var runtime *umemory.Runtime
	c, _, runtime = algebraicNegTLSFixture(t, transport)
	q, z := c.config.Rationals, c.config.Rationals.Integers()
	input := unumeral.NewRational()
	serviceMust(t, q.SetString(&input, "18446744073709551617/18446744073709551619"))
	original := input
	var out Algebraic
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.SetRational(&out, &input)
	s := &c.constQ
	if body == nil || !reached || !s.active || s.sourceCursor != 2 || !z.MatchesCopyAllocationInvocation(&s.copy) || input != original || out != (Algebraic{}) {
		t.Fatal("actual const copy allocation ownership absent", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	invocation := s.invocation
	serviceMust(t, c.CleanupRetained())
	if s.active || c.localActive || s.sourceCursor != 2 || s.ownCursor != 2 || z.MatchesCopyAllocationInvocation(&s.copy) {
		t.Fatal("own const copy acknowledgement changed source cursor")
	}
	serviceMust(t, c.SetRational(&out, &input))
	if s.invocation != invocation+1 {
		t.Fatal("new const invocation did not advance")
	}
	equal, e := c.EqualRational(&out, &input)
	serviceMust(t, e)
	if !equal {
		t.Fatal("const setter retry changed input value")
	}
	serviceMust(t, c.Del(&out))
	serviceMust(t, q.Del(&input))
}
