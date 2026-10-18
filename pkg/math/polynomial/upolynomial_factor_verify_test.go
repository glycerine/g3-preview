package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"os"
	"strings"
	"testing"
)

func polynomialVerifyActionsFixture(t *testing.T, choices string) *os.File {
	t.Helper()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	debug, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	inputFile, e := os.CreateTemp(t.TempDir(), "polynomial-assert-input-*")
	serviceMust(t, e)
	_, e = inputFile.WriteString(choices)
	serviceMust(t, e)
	_, e = inputFile.Seek(0, 0)
	serviceMust(t, e)
	outputFile, e := os.CreateTemp(t.TempDir(), "polynomial-assert-output-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outputFile, Stderr: outputFile})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, inputFile)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, e)
	serviceMust(t, debug.BindAssertionStreams(streams))
	serviceMust(t, debug.BindAssertionActions(actions))
	previous, e := diagnostic.BindProcessDebugContext(debug)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, inputFile.Close())
		serviceMust(t, outputFile.Close())
		serviceMust(t, debug.Close())
		serviceMust(t, region.Close())
	})
	return outputFile
}

func TestUPolynomialFactorExactDivisionVerifyActions(t *testing.T) {
	for _, line := range []int32{3074, 3085, 3100} {
		for _, action := range []string{"T", "C", "disabled"} {
			t.Run(fmt.Sprint(line)+"/"+action, func(t *testing.T) {
				_, s := preparationPhysicalFixture(t)
				m := s.manager
				a, b, out := upolySourceNew(t, m, []int32{1, 0, 1}), upolySourceNew(t, m, []int32{1, 1}), upolySourceNew(t, m, []int32{99})
				av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
				if line == 3100 {
					out = a
				}
				d, e := m.descriptor(out)
				serviceMust(t, e)
				choice := action
				if action == "disabled" {
					choice = "C"
				}
				file := polynomialVerifyActionsFixture(t, choice)
				if action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				// Actual nonexact division computes its private remainder before VERIFY's
				// false leaf. These are direct call-site adapters, not fabricated frontends.
				e = s.exactDivVerifyWithSourceContext(av, bv, &d.state, nil, integerAllocationInvocationContext{}, line)
				if !polynomialSourceDebug {
					if !errors.Is(e, diagnostic.ErrDebugNativeUnreachable) {
						t.Fatal("GNU release false VERIFY domain", e)
					}
				} else if action == "T" {
					if !errors.Is(e, ErrUPolynomialAssertion) || !linearThrownAssertion(e) {
						t.Fatal("VERIFY Throw", e)
					}
				} else {
					serviceMust(t, e)
				}
				upolyGCDExpect(t, m, out, nil) // source failure after division resets r

				if m.div1.size != 1 || m.values(&m.div1)[0].Val != 2 {
					t.Fatal("VERIFY skipped actual division/remainder", m.div1)
				}
				data, e := os.ReadFile(file.Name())
				serviceMust(t, e)
				expr := "exact_div(C, B, A)"
				if line == 3085 {
					expr = "exact_div(A, D, C)"
				}
				if line == 3100 {
					expr = "exact_div(B, D, B)"
				}
				if !strings.HasPrefix(string(data), "Failed to verify: "+expr+"\n") {
					t.Fatal("VERIFY message", string(data))
				}
				if polynomialSourceDebug && !strings.Contains(string(data), fmt.Sprintf("Line: %d\nUNREACHABLE CODE WAS REACHED.", line)) {
					t.Fatal("VERIFY source site", string(data))
				}
			})
		}
	}
}
func TestUPolynomialFactorExactDivisionVerifySuccessfulOperand(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	a, b, out := upolySourceNew(t, m, []int32{-1, 0, 1}), upolySourceNew(t, m, []int32{-1, 1}), upolySourceNew(t, m, []int32{99})
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	d, e := m.descriptor(out)
	serviceMust(t, e)
	file := polynomialVerifyActionsFixture(t, "")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	serviceMust(t, s.exactDivVerifyWithSourceContext(av, bv, &d.state, nil, integerAllocationInvocationContext{}, 3074))
	upolyGCDExpect(t, m, out, []int64{1, 1})
	data, e := os.ReadFile(file.Name())
	serviceMust(t, e)
	if len(data) != 0 {
		t.Fatal("successful VERIFY emitted diagnostic", string(data))
	}
}
