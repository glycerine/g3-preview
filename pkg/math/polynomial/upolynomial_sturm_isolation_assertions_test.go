package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func sturmIsolationAssertionAction(t *testing.T, action string) {
	t.Helper()
	polynomialAssertionActionsFixture(t, action)
	if polynomialSourceDebug && action == "disabled" {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
}

func TestUPolynomialSturmIsolationFrameAssertionActions(t *testing.T) {
	for _, line := range []int32{2441, 2442, 2463, 2464} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(strconv.Itoa(int(line))+"/"+action, func(t *testing.T) {
				m, q, budget := upolySourceOwner(t)
				b := upolyIsolationBinaryOwner(t, q, budget)
				roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
				p := upolySourceNew(t, m, []int32{0, 1})
				v := mustUPolyView(t, m, p)
				zero := unumeral.NewBinaryInt(0)
				var stack upolySSStack
				sturmIsolationAssertionAction(t, action)
				used := budget.Used()
				body := m.pushSSFrame(v, b, &stack, roots, lo, hi, 1, 0, &zero, &zero, line)
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
						t.Fatal("frame assertion did not Throw", body)
					}
					if budget.Used() != used || roots.state.size != 0 || lo.state.size != 0 || hi.state.size != 0 || len(stack.values) != 0 {
						t.Fatal("Throw passed source frame assertion")
					}
				} else {
					serviceMust(t, body)
					if roots.state.size != 1 || lo.state.size != 0 || hi.state.size != 0 || len(stack.values) != 0 {
						t.Fatal("Continue skipped exact upper-root publication")
					}
					equal, e := b.Equal(&roots.state.values[0], &zero)
					serviceMust(t, e)
					if !equal {
						t.Fatal("independent exact root0")
					}
				}
			})
		}
	}
}

func TestUPolynomialSturmIsolationNoZeroAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			b := upolyIsolationBinaryOwner(t, q, budget)
			roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			p := upolySourceNew(t, m, []int32{0, 1})
			sturmIsolationAssertionAction(t, action)
			used := budget.Used()
			body := m.SturmIsolateRootsWithBounds(mustUPolyView(t, m, p), 1, 1, b, roots, lo, hi)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
					t.Fatal("no-zero assertion did not Throw", body)
				}
				if used != budget.Used() || roots.state.size != 0 {
					t.Fatal("Throw reached source sequence construction")
				}
			} else {
				serviceMust(t, body)
				if roots.state.size != 1 || lo.state.size != 0 || hi.state.size != 0 {
					t.Fatal("Continue lost root0", roots.state.size, lo.state.size, hi.state.size)
				}
				zero := unumeral.NewBinaryInt(0)
				equal, e := b.Equal(&roots.state.values[0], &zero)
				serviceMust(t, e)
				if !equal {
					t.Fatal("independent root0")
				}
			}
		})
	}
}

func TestUPolynomialSturmIsolationDisabledAssertionsSkipOperands(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	sturmIsolationAssertionAction(t, "disabled")
	used := budget.Used()
	budget.FailAfter(0)
	serviceMust(t, m.sturmIsolationNoZeroAssertion(UPolynomialView{}))
	for _, line := range []int32{2441, 2442, 2463, 2464} {
		serviceMust(t, m.pushSSFrame(UPolynomialView{}, nil, nil, nil, nil, nil, 4, 4, nil, nil, line))
	}
	budget.FailAfter(-1)
	if budget.Used() != used {
		t.Fatal("disabled assertions acquired storage")
	}
}

func TestUPolynomialSturmIsolationOutputSizeAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			m, q, budget := upolySourceOwner(t)
			b := upolyIsolationBinaryOwner(t, q, budget)
			roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			seven := unumeral.NewBinaryInt(7)
			serviceMust(t, upolyAppendBinaryFixture(lo, &seven))
			p := upolySourceNew(t, m, []int32{1, 0, 1})
			sturmIsolationAssertionAction(t, action)
			used := budget.Used()
			body := m.SturmIsolateRoots(mustUPolyView(t, m, p), b, roots, lo, hi)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
					t.Fatal("output-size assertion did not Throw", body)
				}
				if used != budget.Used() {
					t.Fatal("Throw constructed source normalized vector")
				}
			} else {
				serviceMust(t, body)
			}
			if roots.state.size != 0 || lo.state.size != 1 || hi.state.size != 0 {
				t.Fatal("source no-real-roots output prefix changed")
			}
			equal, e := b.Equal(&lo.state.values[0], &seven)
			serviceMust(t, e)
			if !equal {
				t.Fatal("source existing lower prefix changed")
			}
		})
	}
}
