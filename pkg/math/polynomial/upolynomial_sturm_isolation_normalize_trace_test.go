package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"math/big"
	"strings"
	"testing"
)

// Qualify the CTRACE refusal branch by changing a coefficient after the actual
// source GCD prefix. The valid normalization algorithm preserves divisibility;
// this deliberately inconsistent operand exercises its diagnostic contract.
func TestUPolynomialSturmIsolationNormalizeTraceTextAndAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		for _, closed := range []bool{false, true} {
			name := action + "/text"
			if closed {
				name = action + "/closed"
			}
			t.Run(name, func(t *testing.T) {
				c, m, _ := algebraicNegTLSFixture(t, nil)
				p := upolySourceNew(t, m, []int32{2, 4})
				v, e := m.wholeView(p)
				serviceMust(t, e)
				q, e := m.beginSturmIsolation(UPolynomialView{}, c.config.Binary)
				serviceMust(t, e)
				serviceMust(t, q.beginNormalization())
				serviceMust(t, m.gcdSpanWithSourceContext(v, &q.normalization.values[0], q.context()))
				x, e := m.read(v, 0)
				serviceMust(t, e)
				serviceMust(t, m.modular.SetInt64(x, 3))
				choices := action
				if action == "disabled" {
					choices = ""
				}
				polynomialAssertionActionsFixture(t, choices)
				if polynomialSourceDebug && action == "disabled" {
					serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
				}
				var text strings.Builder
				refused := &linearTraceFailWriter{}
				var out io.Writer = &text
				if closed {
					out = refused
				}
				tags := diagnostic.NewTraceTagsWithWriter(out)
				tags.Enable("div_bug")
				serviceMust(t, m.config.Integer.BindTrace(tags))
				body := q.normalizeCoefficient(v, 0)
				serviceMust(t, m.config.Integer.BindTrace(nil))
				if polynomialSourceDebug && action == "T" {
					if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
						t.Fatal("normalization SASSERT did not Throw", body)
					}
				} else {
					serviceMust(t, body)
				}
				if polynomialSourceDebug && tags.ShouldTrace("div_bug") {
					if closed {
						if refused.calls != 1 {
							t.Fatal("trace retried sticky stream", refused.calls)
						}
					} else if !strings.Contains(text.String(), "old(p[i]): 3, g: 2, p[i]: 1, tmp: 2\n") {
						t.Fatal("source normalization trace text", text.String())
					}
				} else if text.Len() != 0 || refused.calls != 0 {
					t.Fatal("erased trace emitted text")
				}
				body = q.endNormalization(body)
				_ = q.finish(body)
				serviceMust(t, m.CleanupSturmIsolation())
			})
		}
	}
}

func TestUPolynomialSturmIsolationNormalizeDisabledTraceSkipsOperands(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	budget := m.config.Integer.Budget()
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	serviceMust(t, m.config.Integer.BindTrace(tags))
	defer func() { serviceMust(t, m.config.Integer.BindTrace(nil)) }()
	q, e := m.beginSturmIsolation(UPolynomialView{}, c.config.Binary)
	serviceMust(t, e)
	used := budget.Used()
	budget.FailAfter(0)
	// No normalization scope exists, and no operand may be read.
	serviceMust(t, q.traceNormalization(nil))
	budget.FailAfter(-1)
	if text.Len() != 0 || used != budget.Used() {
		t.Fatal("disabled trace evaluated operands")
	}
	serviceMust(t, q.finish(nil))
}

// The stream fails on its first header insertion. The later source
// to_string operand still computes and retains its actual formatter receipt.
func TestUPolynomialSturmIsolationNormalizeTraceComputationAfterFailedStream(t *testing.T) {
	c, m, _ := algebraicNegTLSFixture(t, nil)
	z := m.config.Integer
	p := upolySourceNew(t, m, nil)
	var inputs [2]unumeral.Int
	common := new(big.Int).Lsh(big.NewInt(1), 512)
	for i, text := range []string{common.String(), new(big.Int).Lsh(new(big.Int).Set(common), 1).String()} {
		serviceMust(t, z.SetString(&inputs[i], text))
	}
	serviceMust(t, m.SetInts(p, inputs[:]))
	for i := range inputs {
		serviceMust(t, z.Del(&inputs[i]))
	}
	v, e := m.wholeView(p)
	serviceMust(t, e)
	q, e := m.beginSturmIsolation(UPolynomialView{}, c.config.Binary)
	serviceMust(t, e)
	serviceMust(t, q.beginNormalization())
	serviceMust(t, m.gcdSpanWithSourceContext(v, &q.normalization.values[0], q.context()))
	x, e := m.read(v, 0)
	serviceMust(t, e)
	serviceMust(t, z.SetString(x, new(big.Int).Add(common, big.NewInt(1)).String()))
	context := q.context()
	output := &upolyNormalizeFailedProbeWriter{probe: func() {
		used := z.Budget().Used()
		for _, e := range []error{m.CleanupSturmIsolation(), m.Close()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("TRACE callback acquired source owner", e)
			}
		}
		probe := unumeral.NewInt(0)
		if context.sturmIsolationFrame(z) != nil || z.WithAllocationContext(context.utility(z)).SetUint32(&probe, 7) == nil || probe.Val != 0 || z.Budget().Used() != used {
			t.Fatal("TRACE callback acquired numerical context")
		}
		// Qualify the formatter's actual out-of-range limb guard after
		// CTRACE has evaluated its condition and before to_string(old_p_i).
		old := &q.normalization.values[1]
		cell := z.Cell(old.Cell)
		size, e := cell.Backing.ReadWord(z.Allocator, 0)
		serviceMust(t, e)
		for i := size; i < cell.Capacity; i++ {
			serviceMust(t, z.RuntimeOwner().WriteUint32(cell.Backing.Raw, cell.Backing.Offset+8+4*uint64(i), 0))
		}
		serviceMust(t, z.RuntimeOwner().WriteUint32(cell.Backing.Raw, cell.Backing.Offset, cell.Capacity+1))
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("div_bug")
	serviceMust(t, z.BindTrace(tags))
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	body := q.normalizeCoefficient(v, 0)
	z.Budget().FailAfter(-1)
	serviceMust(t, z.BindTrace(nil))
	enabled := polynomialSourceDebug && tags.ShouldTrace("div_bug")
	if enabled {
		if !output.called || !errors.Is(body, arena.ErrBounds) || !q.normalization.insertions[0].Active() {
			t.Fatal("TRACE lost actual formatting refusal", body, output.called, q.normalization.insertions[0].Active())
		}
	} else {
		serviceMust(t, body)
		if output.called {
			t.Fatal("erased trace called writer")
		}
	}
	body = q.endNormalization(body)
	_ = q.finish(body)
	serviceMust(t, m.CleanupSturmIsolation())
	if q.active || q.normalizePending() {
		t.Fatal("TRACE retained acknowledged formatter")
	}
}
