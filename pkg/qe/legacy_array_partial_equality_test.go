package qe

// Source: src/muz/spacer/spacer_qe_project.cpp peq.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestLegacyPartialEqualityApplicationAndExpansion(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	family, err := m.RegisterArrayPlugin()
	fmMust(t, err)
	arraySort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	a := f.constant("legacy-peq-a", arraySort)
	b := f.constant("legacy-peq-b", arraySort)
	i := f.constant("legacy-peq-i", f.integer)
	j := f.constant("legacy-peq-j", f.integer)
	partial, err := newLegacyPartialEquality(m, family, a, b, []ast.ASTHandle{i, j})
	fmMust(t, err)
	defer func() { fmMust(t, partial.Close()) }()
	application, err := partial.Application()
	fmMust(t, err)
	defer func() { fmMust(t, m.DelRef(&application)) }()
	handle, err := m.RefGet(application)
	fmMust(t, err)
	count, err := m.GetNumArgs(handle)
	fmMust(t, err)
	if count != 4 {
		t.Fatal("partial equality argument count", count)
	}
	expansion, err := partial.Expand(true)
	fmMust(t, err)
	defer func() { fmMust(t, expansion.Close()) }()
	if len(expansion.constants) != 2 {
		t.Fatal("partial equality diff constant count", len(expansion.constants))
	}
	equality, err := m.RefGet(expansion.equality)
	fmMust(t, err)
	args, matched, err := (&ArrayProjection{manager: m}).operation(equality, m.BasicFamilyID(), ast.ASTOpEq)
	fmMust(t, err)
	if !matched || len(args) != 2 || args[0] != a {
		t.Fatal("partial equality expansion shape")
	}
	depth := 0
	current := args[1]
	for {
		storeArgs, store, err := (&ArrayProjection{manager: m, family: family}).operation(current, family, ast.ASTArrayOpStore)
		fmMust(t, err)
		if !store {
			break
		}
		depth++
		current = storeArgs[0]
	}
	if depth != 2 || current != b {
		t.Fatal("partial equality expansion store chain", depth)
	}
}
