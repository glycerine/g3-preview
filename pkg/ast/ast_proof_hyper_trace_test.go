package ast

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTHyperResolveSourceTracePremisesAndParameters(t *testing.T) {
	c, globals, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() { _, e := uparams.BindProcessGlobalParamsService(previous); astTestMust(t, e) })
	premises, conclusion, substitutions := hyperTestInputs(t, m)
	positions := []ASTProofPosition{{First: math.MaxUint32, Second: 1 << 31}}
	_, err = m.MkProofHyperResolve(premises, conclusion, positions, substitutions)
	astTestMust(t, err)
	if output.Len() != 0 {
		t.Fatalf("disabled tag wrote %q", output.String())
	}
	tags.Enable("hyper_res")
	_, err = m.MkProofHyperResolve(premises, conclusion, positions, substitutions)
	astTestMust(t, err)
	if tags.ShouldTrace("hyper_res") {
		ids := make([]uint32, 3)
		for i, h := range []ASTHandle{substitutions[0][0], substitutions[0][1], substitutions[1][0]} {
			ids[i], err = m.GetASTID(h)
			astTestMust(t, err)
		}
		premiseHeader := "-------- [hyper_res] mk_hyper_resolve src/ast/ast.cpp:3152 ---------\n"
		paramsHeader := "-------- [hyper_res] mk_hyper_resolve src/ast/ast.cpp:3167 ---------\n"
		footer := "------------------------------------------------\n"
		want := premiseHeader + "(hypothesis hyper-a)\n" + footer +
			premiseHeader + "(hypothesis hyper-b)\n" + footer +
			paramsHeader + fmt.Sprintf("#%d\n#%d\n-1\n-2147483648\n#%d\n", ids[0], ids[1], ids[2]) + footer
		if got := output.String(); got != want {
			t.Fatalf("hyper-res trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased trace wrote %q", output.String())
	}
	output.Reset()
	_, err = m.MkProofHyperResolve(nil, conclusion, nil, [][]ASTHandle{nil})
	astTestMust(t, err)
	if tags.ShouldTrace("hyper_res") {
		want := "-------- [hyper_res] mk_hyper_resolve src/ast/ast.cpp:3167 ---------\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("empty parameter trace: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased empty trace wrote %q", output.String())
	}
}
