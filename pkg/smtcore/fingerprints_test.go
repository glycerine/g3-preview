package smtcore

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMTFingerprintExactThenRootIdentityAndScopes(t *testing.T) {
	m := astTestManager(t)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	a, an := smtEnodeConstant(t, m, 2100, boolean, 0)
	_, bn := smtEnodeConstant(t, m, 2101, boolean, 0)
	bn.root = an
	bound, err := m.MkVar(0, boolean)
	astTestMust(t, err)
	q1, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{symbol.Numerical(2102)}, Body: bound})
	astTestMust(t, err)
	q2, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{boolean}, Names: []symbol.Symbol{symbol.Numerical(2103)}, Body: bound})
	astTestMust(t, err)
	q1ID, err := m.GetASTID(q1)
	astTestMust(t, err)
	q2ID, err := m.GetASTID(q2)
	astTestMust(t, err)
	set := NewSMTFingerprintSet()
	first, err := set.Insert(q1, q1ID, []*SMTEnode{bn})
	astTestMust(t, err)
	if first == nil {
		t.Fatal("root-normalized insertion returned nil")
	}
	firstArg, err := first.Arg(0)
	astTestMust(t, err)
	if firstArg != an || first.Data() != q1 || first.DataHash() != q1ID {
		t.Fatal("root-normalized insertion", first)
	}
	if duplicate, err := set.Insert(q1, q1ID, []*SMTEnode{an}); err != nil || duplicate != nil {
		t.Fatal("duplicate normalized fingerprint", duplicate, err)
	}
	if found, err := set.Contains(q1, q1ID, []*SMTEnode{bn}); err != nil || !found {
		t.Fatal("two-stage contains", found, err)
	}
	astTestMust(t, set.PushScope())
	second, err := set.Insert(q2, q2ID, []*SMTEnode{an})
	astTestMust(t, err)
	if second == nil || set.Size() != 2 {
		t.Fatal("scoped insertion")
	}
	astTestMust(t, set.PopScopes(1))
	if set.Size() != 1 {
		t.Fatal("scope rollback size", set.Size())
	}
	if found, err := set.Contains(q2, q2ID, []*SMTEnode{an}); err != nil || found {
		t.Fatal("scope rollback membership", found, err)
	}
	var output bytes.Buffer
	astTestMust(t, set.Display(m, &output))
	aID, err := m.GetASTID(a)
	astTestMust(t, err)
	if !strings.HasPrefix(output.String(), "fingerprints:\n#") || !strings.Contains(output.String(), fmt.Sprintf(" #%d\n", aID)) || strings.Count(output.String(), "\n") != 2 {
		t.Fatal("fingerprint display", output.String())
	}
}

func TestSMTFingerprintResetAndGuards(t *testing.T) {
	set := NewSMTFingerprintSet()
	if _, err := set.Insert(ast.ASTHandle{}, 0, nil); err == nil {
		t.Fatal("null data accepted")
	}
	if err := set.PopScopes(0); err == nil {
		t.Fatal("zero-scope pop accepted")
	}
	set.Reset()
	if set.Size() != 0 {
		t.Fatal("reset retained fingerprints")
	}
}
