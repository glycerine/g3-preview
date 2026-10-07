// Licensed under the MIT license.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTQuantifierSourcePatternDiagnosticBeforeAllocation(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	truth, e := m.MkFamilyApp(0, ASTOpTrue, nil, nil)
	astTestMust(t, e)
	spec := ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{b}, Names: []util.Symbol{util.NumericalSymbol(0)}, Body: truth, Patterns: []ASTHandle{truth}}
	before := m.Stats()
	next := m.nextNode
	ids := m.ExprIDs.Next
	rc, e := m.GetRefCount(truth)
	astTestMust(t, e)
	q, e := m.MkQuantifier(spec)
	if ASTSourceDebug {
		if e != ErrASTDomain {
			t.Fatal("source is_pattern diagnostic missing", e)
		}
		if m.Stats() != before || m.nextNode != next || m.ExprIDs.Next != ids {
			t.Fatal("diagnostic happened after candidate allocation")
		}
		after, e := m.GetRefCount(truth)
		astTestMust(t, e)
		if after != rc {
			t.Fatal("diagnostic changed RC")
		}
	} else {
		astTestMust(t, e)
		astTestPin(t, m, q)
	}
	pattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{truth})
	astTestMust(t, e)
	astTestPin(t, m, pattern)
	spec.Patterns = []ASTHandle{pattern}
	q, e = m.MkQuantifier(spec)
	astTestMust(t, e)
	astTestPin(t, m, q)
	variable, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, variable)
	badPattern, e := m.MkFamilyApp(2, 0, nil, []ASTHandle{variable})
	astTestMust(t, e)
	astTestPin(t, m, badPattern)
	spec.Patterns = []ASTHandle{badPattern}
	q, e = m.MkQuantifier(spec)
	if ASTSourceDebug {
		if e != ErrASTDomain {
			t.Fatal("source is_pattern requires App children", e)
		}
	} else {
		astTestMust(t, e)
		astTestPin(t, m, q)
	}
	spec.Patterns = nil
	spec.NoPatterns = []ASTHandle{truth}
	q, e = m.MkQuantifier(spec)
	astTestMust(t, e)
	astTestPin(t, m, q)
}
