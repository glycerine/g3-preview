package rewriter

// Source: src/ast/rewriter/rewriter_params.hpp:5-37.

import "testing"

func TestRewriterParamsDefaultsSnapshotAndLocalPrecedence(t *testing.T) {
	c, g, local, _ := dynAckSourceFixture(t)
	p, e := NewRewriterParams(c, g, nil)
	paramsMust(t, e)
	u1, e := p.MaxMemory()
	paramsMust(t, e)
	u2, e := p.MaxSteps()
	paramsMust(t, e)
	b1, e := p.Flat()
	paramsMust(t, e)
	b2, e := p.PushITEArith()
	paramsMust(t, e)
	b3, e := p.PushITEBV()
	paramsMust(t, e)
	b4, e := p.PullCheapITE()
	paramsMust(t, e)
	u3, e := p.BVIneqConsistencyTestMax()
	paramsMust(t, e)
	b5, e := p.CacheAll()
	paramsMust(t, e)
	b6, e := p.RewritePatterns()
	paramsMust(t, e)
	b7, e := p.IgnorePatternsOnGroundQBody()
	paramsMust(t, e)
	if u1 != ^uint32(0) || u2 != ^uint32(0) || !b1 || b2 || b3 || b4 || u3 != 0 || b5 || b6 || !b7 {
		t.Fatal("generated defaults", u1, u2, b1, b2, b3, b4, u3, b5, b6, b7)
	}
	paramsMust(t, p.Close())

	for key, value := range map[string]string{
		"rewriter.max_memory":                      "17",
		"rewriter.max_steps":                       "19",
		"rewriter.flat":                            "false",
		"rewriter.push_ite_arith":                  "true",
		"rewriter.push_ite_bv":                     "true",
		"rewriter.pull_cheap_ite":                  "true",
		"rewriter.bv_ineq_consistency_test_max":    "23",
		"rewriter.cache_all":                       "true",
		"rewriter.rewrite_patterns":                "true",
		"rewriter.ignore_patterns_on_ground_qbody": "false",
	} {
		paramsMust(t, g.SourceSetName(key, value))
	}
	p, e = NewRewriterParams(c, g, local)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, p.Close()) })
	paramsMust(t, g.SourceSetName("rewriter.max_steps", "29"))
	u2, e = p.MaxSteps()
	paramsMust(t, e)
	b1, e = p.Flat()
	paramsMust(t, e)
	b2, e = p.PushITEArith()
	paramsMust(t, e)
	b3, e = p.PushITEBV()
	paramsMust(t, e)
	b4, e = p.PullCheapITE()
	paramsMust(t, e)
	u3, e = p.BVIneqConsistencyTestMax()
	paramsMust(t, e)
	b5, e = p.CacheAll()
	paramsMust(t, e)
	b6, e = p.RewritePatterns()
	paramsMust(t, e)
	b7, e = p.IgnorePatternsOnGroundQBody()
	paramsMust(t, e)
	if u2 != 19 || b1 || !b2 || !b3 || !b4 || u3 != 23 || !b5 || !b6 || b7 {
		t.Fatal("module snapshot", u2, b1, b2, b3, b4, u3, b5, b6, b7)
	}
	key := "max_steps"
	paramsMust(t, c.SetUIntName(local, &key, 31))
	key = "flat"
	paramsMust(t, c.SetBoolName(local, &key, true))
	u2, e = p.MaxSteps()
	paramsMust(t, e)
	b1, e = p.Flat()
	paramsMust(t, e)
	if u2 != 31 || !b1 {
		t.Fatal("local precedence", u2, b1)
	}
	copyOwner := *p
	if _, e = copyOwner.CacheAll(); e == nil {
		t.Fatal("copied facade accepted")
	}
	paramsMust(t, p.Close())
	if _, e = p.MaxMemory(); e == nil {
		t.Fatal("closed facade accepted")
	}
}
