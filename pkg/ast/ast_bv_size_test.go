// Source: src/ast/bv_decl_plugin.h:302,390-394.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorOriginalSizeSortAndExpression(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, width := range []uint32{1, 8, 65, 257} {
		s, e := m.BitVectorSort(width)
		astTestMust(t, e)
		x, e := m.MkConst(astTestSymbol(t, m, "bv-size-x"), s)
		astTestMust(t, e)
		astTestPin(t, m, x)
		v, e := m.MkVar(0, s)
		astTestMust(t, e)
		astTestPin(t, m, v)
		for _, h := range []ASTHandle{s, x, v} {
			before := m.Stats()
			m.Budget().FailAfter(0)
			got, e := m.BitVectorSize(h)
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			if got != width || m.Stats() != before {
				t.Fatal("borrowed width getter", width, got)
			}
		}
		for _, h := range []ASTHandle{x, v} {
			m.Budget().FailAfter(0)
			isBV, e := m.IsBitVector(h)
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			if !isBV {
				t.Fatal("expression sort recognizer", width)
			}
		}
	}
	truth, e := m.MkTrue()
	astTestMust(t, e)
	isBV, e := m.IsBitVector(truth)
	astTestMust(t, e)
	if isBV {
		t.Fatal("Bool expression recognized as BV")
	}
}

func TestASTBitVectorSizeSourceDebugGuardAndReleaseRead(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterInt(-3)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&p)) }()
	for _, s := range []ASTHandle{
		bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{p, p}),
		bvRecognizerTestSort(t, m, fid, 1, []ASTParameter{p}),
	} {
		m.Budget().FailAfter(0)
		width, body := m.BitVectorSize(s)
		m.Budget().FailAfter(-1)
		if ASTSourceDebug {
			if !errors.Is(body, util.ErrGlobalParamsAssertion) {
				t.Fatal("source sort assertion", body)
			}
		} else if body != nil || width != ^uint32(0)-2 {
			t.Fatal("release must erase recognition and preserve unsigned conversion", width, body)
		}
	}
}
