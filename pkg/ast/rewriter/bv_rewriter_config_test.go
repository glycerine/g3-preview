// Source: src/ast/rewriter/bv_rewriter.cpp:26-51;
// src/ast/rewriter/bv_rewriter_params.pyg:1-20.
package rewriter

import (
	"errors"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"testing"
)

func TestBVRewriterConfigurationDefaultsAndLocalOnlyOption(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	p, e := NewBVPolynomialConfiguration(c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	module, e := g.SymbolsField.Intern("rewriter")
	astTestMust(t, e)
	cell, e := g.GetParams(module)
	astTestMust(t, e)
	k := "mkbv2num"
	astTestMust(t, c.SetBoolName(cell, &k, true))
	astTestMust(t, p.UpdateBitVectorLocal(&local))
	s, e := p.BitVectorState()
	astTestMust(t, e)
	want := BVRewriterConfigurationState{HiDiv0: true, EliminateSignExtension: true, BitToBool: true}
	if s != want {
		t.Fatal(s, want)
	}
	k = "mkbv2num"
	astTestMust(t, c.SetBoolName(&local, &k, true))
	astTestMust(t, p.UpdateBitVectorLocal(&local))
	s, e = p.BitVectorState()
	astTestMust(t, e)
	if !s.MkBVToNumeral {
		t.Fatal("local-only option ignored")
	}
}

func TestBVRewriterConfigurationModuleFallbackAndSorting(t *testing.T) {
	c, g, b := bvPolynomialParamsFixture(t)
	p, e := NewBVPolynomialConfiguration(c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	module, e := g.SymbolsField.Intern("rewriter")
	astTestMust(t, e)
	cell, e := g.GetParams(module)
	astTestMust(t, e)
	for _, k := range []string{"mul2concat", "bv_trailing", "bv_urem_simpl", "blast_eq_value", "split_concat_eq", "udiv2mul", "bvnot2arith", "bv_not_simpl", "bv_sort_ac", "bv_extract_prop", "bv_ite2id", "bv_le_extra"} {
		key := k
		astTestMust(t, c.SetBoolName(cell, &key, true))
	}
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	k := "bit2bool"
	astTestMust(t, c.SetBoolName(&local, &k, false))
	k = "bv_trailing"
	astTestMust(t, c.SetUIntName(&local, &k, 0))
	k = "flat"
	astTestMust(t, c.SetBoolName(&local, &k, false))
	astTestMust(t, p.UpdateBitVector(&local))
	s, e := p.BitVectorState()
	astTestMust(t, e)
	want := BVRewriterConfigurationState{HiDiv0: true, EliminateSignExtension: true, MultiplyToConcat: true, Trailing: true, RemainderSimplification: true, BlastEqualityValue: true, SplitConcatEquality: true, UnsignedDivisionToMultiply: true, NotToArithmetic: true, NotSimplification: true, SortAC: true, ExtractPropagation: true, IteToIdentity: true, LEExtra: true}
	if s != want {
		t.Fatal(s, want)
	}
	base, e := p.State()
	astTestMust(t, e)
	if base.Flat || !base.SortSums {
		t.Fatal("base update / final sort assignment", base)
	}
	// All key names are now interned; the warm update has no new allocation.
	b.FailAfter(0)
	e = p.UpdateBitVector(&local)
	b.FailAfter(-1)
	astTestMust(t, e)
	if p.placed[3] {
		t.Fatal("generated BV snapshot still live")
	}
	k = "bv_sort_ac"
	astTestMust(t, c.SetBoolName(&local, &k, false))
	astTestMust(t, p.UpdateBitVectorLocal(&local))
	base, e = p.State()
	astTestMust(t, e)
	if base.SortSums {
		t.Fatal("sort_sums must follow final bv_sort_ac lookup")
	}
}

func TestBVRewriterConfigurationAllocationFreeSnapshotAndStoppedOwner(t *testing.T) {
	c, g, b := bvPolynomialParamsFixture(t)
	p, e := NewBVPolynomialConfiguration(c, g, nil)
	astTestMust(t, e)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	module, e := g.SymbolsField.Intern("rewriter")
	astTestMust(t, e)
	cell, e := g.GetParams(module)
	astTestMust(t, e)
	key := "mul2concat"
	astTestMust(t, c.SetBoolName(cell, &key, true))
	key = "unrelated_local_option"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	refs := c.Stats().References
	// Even first-time option-name reads compare strings without interning.
	b.FailAfter(0)
	body := p.UpdateBitVectorLocal(&local)
	b.FailAfter(-1)
	astTestMust(t, body)
	if p.placed[3] || c.Stats().References != refs {
		t.Fatal("generated snapshot not retired", c.Stats())
	}
	old := p.bitVector
	if !old.MultiplyToConcat {
		t.Fatal(old)
	}
	copied := *p
	if e = copied.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("copied owner", e)
	}
	// An actual unavailable borrowed global owner stops the reached suffix.
	astTestMust(t, g.Close())
	body = p.UpdateBitVectorLocal(&local)
	if body == nil {
		t.Fatal("closed borrowed global accepted")
	}
	if p.cause != body || !p.stopped {
		t.Fatal("failure prefix", body, p.cause)
	}
	if e = p.UpdateBitVectorLocal(&local); e != body {
		t.Fatal("stopped local update", e, body)
	}
	if e = p.UpdateBitVector(&local); e != body {
		t.Fatal("stopped combined update", e, body)
	}
	if _, e = p.BitVectorState(); e != body {
		t.Fatal("stopped state", e, body)
	}
	if p.bitVector != old {
		t.Fatal("retry changed prefix")
	}
	astTestMust(t, p.Close())
	astTestMust(t, p.Close())
	if p.cause != body {
		t.Fatal("cleanup changed failure history")
	}
}

func TestBVRewriterConfigurationDescriptionsAndIndependentSetter(t *testing.T) {
	c, g, b := bvPolynomialParamsFixture(t)
	p, e := NewBVPolynomialConfiguration(c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	astTestMust(t, p.UpdateBitVectorLocal(&local))
	astTestMust(t, p.SetMkBVToNumeral(true))
	s, e := p.BitVectorState()
	astTestMust(t, e)
	if !s.MkBVToNumeral {
		t.Fatal("source setter ignored")
	}
	astTestMust(t, p.UpdateBitVectorLocal(&local))
	s, e = p.BitVectorState()
	astTestMust(t, e)
	if s.MkBVToNumeral {
		t.Fatal("local update must overwrite source setter")
	}
	for _, external := range []bool{false, true} {
		d, e := uparams.NewParamDescrs(g.SymbolsField, uparams.ParamDescrsConfig{Budget: b})
		astTestMust(t, e)
		astTestMust(t, BVRewriterParameterDescriptions(d, external))
		for _, key := range []string{"flat", "som", "hoist_mul", "som_blowup", "hi_div0", "bit2bool", "mul2concat", "bv_sort_ac", "bv_urem_simpl"} {
			found, e := d.ContainsName(key)
			astTestMust(t, e)
			if !found {
				t.Fatal("missing original descriptor", key)
			}
		}
		found, e := d.ContainsName("mkbv2num")
		astTestMust(t, e)
		if found == external {
			t.Fatal("external-release descriptor guard", external, found)
		}
		found, e = d.ContainsName("arith_ineq_lhs")
		astTestMust(t, e)
		if found {
			t.Fatal("unrelated arithmetic descriptor included")
		}
		astTestMust(t, d.Close())
	}
}
