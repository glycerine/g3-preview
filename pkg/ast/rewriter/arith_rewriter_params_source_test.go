// Source: src/ast/rewriter/arith_rewriter_params.hpp:5-42.
package rewriter

import "testing"

func arithRewriterBoolValues(t *testing.T, p *ArithRewriterParams) []bool {
	t.Helper()
	getters := []func() (bool, error){
		p.AlgebraicNumberEvaluator, p.MulToPower, p.ExpandPower, p.ExpandTan,
		p.SortSums, p.GCDRounding, p.ArithLHS, p.ArithIneqLHS, p.ElimToReal,
		p.PushToReal, p.Eq2Ineq, p.ElimRem,
	}
	values := make([]bool, len(getters))
	for i, get := range getters {
		var e error
		values[i], e = get()
		paramsMust(t, e)
	}
	return values
}

func TestArithRewriterParamsDefaultsSnapshotAndLocalPrecedence(t *testing.T) {
	c, g, local, _ := dynAckSourceFixture(t)
	p, e := NewArithRewriterParams(c, g, nil)
	paramsMust(t, e)
	degree, e := p.MaxDegree()
	paramsMust(t, e)
	values := arithRewriterBoolValues(t, p)
	want := []bool{true, false, false, false, false, false, false, false, false, true, false, false}
	if degree != 64 {
		t.Fatal("max_degree default", degree)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatal("generated bool default", i, values[i], want[i])
		}
	}
	paramsMust(t, p.Close())

	for key, value := range map[string]string{
		"rewriter.algebraic_number_evaluator": "false",
		"rewriter.mul_to_power":               "true",
		"rewriter.expand_power":               "true",
		"rewriter.expand_tan":                 "true",
		"rewriter.max_degree":                 "17",
		"rewriter.sort_sums":                  "true",
		"rewriter.gcd_rounding":               "true",
		"rewriter.arith_lhs":                  "true",
		"rewriter.arith_ineq_lhs":             "true",
		"rewriter.elim_to_real":               "true",
		"rewriter.push_to_real":               "false",
		"rewriter.eq2ineq":                    "true",
		"rewriter.elim_rem":                   "true",
	} {
		paramsMust(t, g.SourceSetName(key, value))
	}
	p, e = NewArithRewriterParams(c, g, local)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, p.Close()) })
	paramsMust(t, g.SourceSetName("rewriter.max_degree", "19"))
	degree, e = p.MaxDegree()
	paramsMust(t, e)
	values = arithRewriterBoolValues(t, p)
	if degree != 17 {
		t.Fatal("module snapshot degree", degree)
	}
	for i, value := range values {
		if value != !want[i] {
			t.Fatal("module snapshot bool", i, value, !want[i])
		}
	}
	key := "max_degree"
	paramsMust(t, c.SetUIntName(local, &key, 23))
	key = "push_to_real"
	paramsMust(t, c.SetBoolName(local, &key, true))
	degree, e = p.MaxDegree()
	paramsMust(t, e)
	push, e := p.PushToReal()
	paramsMust(t, e)
	if degree != 23 || !push {
		t.Fatal("local precedence", degree, push)
	}
	copyOwner := *p
	if _, e = copyOwner.ElimRem(); e == nil {
		t.Fatal("copied facade accepted")
	}
	paramsMust(t, p.Close())
	if _, e = p.MaxDegree(); e == nil {
		t.Fatal("closed facade accepted")
	}
}
