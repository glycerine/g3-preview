package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialBerlekampGetRuntimeAssertionActions(t *testing.T) {
	for _, constant := range []bool{false, true} {
		for _, action := range []string{"T", "C", "disabled"} {
			name := "mutable/" + action
			if constant {
				name = "const/" + action
			}
			t.Run(name, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				three := unumeral.NewInt(3)
				serviceMust(t, m.SetZP(&three))
				f := upolySourceNew(t, m, []int32{1, 0, 1})
				q := upolyBerlekampMatrix{m: m}
				t.Cleanup(q.close)
				serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
				// A caller index crosses into the next real constructed row. Continue
				// reaches that same physical numeral; no matrix ownership/count is changed.
				want, e := m.read(m.stateView(&q.values), 2)
				serviceMust(t, e)
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var got *unumeral.Int
				if constant {
					got, e = q.getConst(0, 2)
				} else {
					got, e = q.get(0, 2)
				}
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || got != nil {
						t.Fatal("source Throw did not precede physical read", got, e)
					}
				} else {
					serviceMust(t, e)
					if got != want {
						t.Fatal("Continue/disabled/release changed reached physical field")
					}
				}
				if q.displaying || q.traceBorrow.executing || q.traceBorrow.bound != 0 {
					t.Fatal("assertion left borrowed owners bound")
				}
			})
		}
	}
}
func TestUPolynomialBerlekampGetClosedRuntimeSwitchPrecedesRead(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	polynomialAssertionActionsFixture(t, "")
	debug := diagnostic.BorrowProcessDebugContext()
	serviceMust(t, debug.Close())
	for _, constant := range []bool{false, true} {
		var value *unumeral.Int
		var e error
		if constant {
			value, e = q.getConst(0, 0)
		} else {
			value, e = q.get(0, 0)
		}
		if value != nil || !errors.Is(e, arena.ErrClosed) {
			t.Fatal("valid predicate bypassed closed switch owner", value, e)
		}
	}
}

func TestUPolynomialBerlekampNextAssertionBeforeResizeAndCursor(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			three := unumeral.NewInt(3)
			serviceMust(t, m.SetZP(&three))
			f := upolySourceNew(t, m, []int32{1, 0, 1})
			q := upolyBerlekampMatrix{m: m}
			t.Cleanup(q.close)
			serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
			rank, e := q.diagonalize()
			serviceMust(t, e)
			if rank != 1 {
				t.Fatal("actual matrix null rank", rank)
			}
			out := upolySourceNew(t, m, []int32{1, 1, 1})
			d, e := m.descriptor(out)
			serviceMust(t, e)
			before, cursor := d.state, q.nullRow
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			found, e := q.next(&d.state)
			if polynomialSourceDebug && action == "T" {
				if found || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || d.state != before || q.nullRow != cursor {
					t.Fatal("Throw passed resize/cursor frontier", found, e, d.state, q.nullRow)
				}
			} else {
				serviceMust(t, e)
				if found || d.state.size != 2 || q.nullRow != q.size {
					t.Fatal("Continue/disabled/release source resize/search", found, d.state, q.nullRow)
				}
				for i := uint32(0); i < 2; i++ {
					value, e := m.read(m.stateView(&d.state), i)
					serviceMust(t, e)
					one, e := m.config.Integer.IsOne(value)
					serviceMust(t, e)
					if !one {
						t.Fatal("resized prefix changed")
					}
				}
			}
			if q.displaying || q.traceBorrow.executing || q.traceBorrow.bound != 0 {
				t.Fatal("null-vector assertion retained borrow")
			}
		})
	}
}
func TestUPolynomialBerlekampNextClosedAssertionSwitchBeforeResize(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	q := upolyBerlekampMatrix{m: m}
	t.Cleanup(q.close)
	serviceMust(t, q.init(m, mustUPolyView(t, m, f)))
	out := upolySourceNew(t, m, []int32{1})
	d, e := m.descriptor(out)
	serviceMust(t, e)
	before, cursor := d.state, q.nullRow
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	found, e := q.next(&d.state)
	if found || !errors.Is(e, arena.ErrClosed) || d.state != before || q.nullRow != cursor {
		t.Fatal("valid predicate bypassed closed switch before resize", found, e)
	}
}

func TestUPolynomialBerlekampEntryAssertionActionsAndPublication(t *testing.T) {
	for _, route := range []string{"degree", "monic"} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(route+"/"+action, func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				three := unumeral.NewInt(3)
				serviceMust(t, m.SetZP(&three))
				coefficients := []int32{1, 1}
				if route == "monic" {
					coefficients = []int32{1, 0, -1}
				}
				f := upolySourceNew(t, m, coefficients)
				fs, e := m.NewFactors()
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, fs.Close()) })
				choice := action
				if action == "disabled" {
					choice = ""
				}
				polynomialAssertionActionsFixture(t, choice)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
				count, ce := fs.DistinctFactors()
				serviceMust(t, ce)
				if polynomialSourceDebug && action == "T" {
					if split || !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) || count != 0 {
						t.Fatal("entry Throw passed factor publication", split, e, count)
					}
				} else {
					serviceMust(t, e)
					if split || count != 1 {
						t.Fatal("Continue/disabled/release changed rank-one source publication", split, count)
					}
					product := upolySourceNew(t, m, nil)
					serviceMust(t, fs.Multiply(product))
					same, e := m.Equal(f, product)
					serviceMust(t, e)
					if !same {
						t.Fatal("source factor copy changed")
					}
				}
				if m.factorTrace != nil {
					t.Fatal("entry assertion retained manager borrow")
				}
			})
		}
	}
}
func TestUPolynomialBerlekampDisabledAssertionsSkipEmptyLeadingRead(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, nil)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	result, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	count, e := fs.DistinctFactors()
	serviceMust(t, e)
	if !result || count != 1 {
		t.Fatal("disabled source assertions did not reach literal empty/tail continuation", result, count)
	}
}

func TestUPolynomialSquareFreeEntryAssertionConstantPublication(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			five := unumeral.NewInt(5)
			serviceMust(t, m.SetZP(&five))
			f := upolySourceNew(t, m, []int32{2})
			old := upolySourceNew(t, m, []int32{1, 1})
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			serviceMust(t, fs.PushCopy(mustUPolyView(t, m, old), 1))
			choice := action
			if action == "disabled" {
				choice = ""
			}
			polynomialAssertionActionsFixture(t, choice)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			e = m.SquareFreeFactorZP(mustUPolyView(t, m, f), fs)
			expected := int64(2)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
					t.Fatal("square-free source Throw", e)
				}
				expected = 1
			} else {
				serviceMust(t, e)
			}
			count, e := fs.DistinctFactors()
			serviceMust(t, e)
			constant, e := m.config.Integer.Int64(&fs.constant)
			serviceMust(t, e)
			if count != 1 || constant != expected {
				t.Fatal("source constant/factor publication frontier", count, constant, expected)
			}
			factor, e := fs.FactorView(0)
			serviceMust(t, e)
			same, e := m.EqualViews(factor, mustUPolyView(t, m, old))
			serviceMust(t, e)
			if !same {
				t.Fatal("older factor changed")
			}
			if m.factorTrace != nil {
				t.Fatal("square-free entry retained assertion borrow")
			}
		})
	}
}
func TestUPolynomialSquareFreeClosedAssertionSwitchBeforeConstantCopy(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	m, _, _ := upolySourceOwner(t)
	five := unumeral.NewInt(5)
	serviceMust(t, m.SetZP(&five))
	f := upolySourceNew(t, m, []int32{1, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	before := m.config.Integer.Budget().Used()
	polynomialAssertionActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	e = m.SquareFreeFactorZP(mustUPolyView(t, m, f), fs)
	if !errors.Is(e, arena.ErrClosed) || fs.count != 0 || m.config.Integer.Budget().Used() != before {
		t.Fatal("valid predicate bypassed closed switch before copy", e, fs.count)
	}
}
