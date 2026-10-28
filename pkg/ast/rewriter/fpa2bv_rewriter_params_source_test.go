package rewriter

// Source: src/ast/fpa/fpa2bv_rewriter_params.hpp:5-17.

import (
	"testing"

	fparams "github.com/glycerine/g3/pkg/ast/fpa"
)

func TestFPA2BVRewriterParamsConstructorLookupAndLifetime(t *testing.T) {
	c, g, local, _ := dynAckSourceFixture(t)
	p, e := fparams.NewFPA2BVRewriterParams(c, g, nil)
	paramsMust(t, e)
	value, e := p.HighFPUnspecified()
	paramsMust(t, e)
	if value {
		t.Fatal("source false default")
	}
	paramsMust(t, p.Close())

	paramsMust(t, g.SourceSetName("rewriter.hi_fp_unspecified", "true"))
	p, e = fparams.NewFPA2BVRewriterParams(c, g, local)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, p.Close()) })
	value, e = p.HighFPUnspecified()
	paramsMust(t, e)
	if !value {
		t.Fatal("source module fallback")
	}
	// The constructor owns a snapshot: later global changes do not alter g.
	paramsMust(t, g.SourceSetName("rewriter.hi_fp_unspecified", "false"))
	value, e = p.HighFPUnspecified()
	paramsMust(t, e)
	if !value {
		t.Fatal("constructor module snapshot changed")
	}
	key := "hi_fp_unspecified"
	paramsMust(t, c.SetBoolName(local, &key, false))
	value, e = p.HighFPUnspecified()
	paramsMust(t, e)
	if value {
		t.Fatal("borrowed local did not override module snapshot")
	}
	copyOwner := *p
	if _, e = copyOwner.HighFPUnspecified(); e == nil {
		t.Fatal("copied facade accepted")
	}
	paramsMust(t, p.Close())
	if _, e = p.HighFPUnspecified(); e == nil {
		t.Fatal("closed facade accepted")
	}
}
