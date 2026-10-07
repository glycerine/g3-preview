// Source: src/ast/bv_decl_plugin.h:107-126.
package ast

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorOriginalDivZeroKindMapping(t *testing.T) {
	for _, test := range []struct{ in, out int32 }{{7, 12}, {8, 13}, {9, 14}, {10, 15}, {11, 16}} {
		got, e := BitVectorDivZeroKind(test.in)
		astTestMust(t, e)
		if got != test.out {
			t.Fatal("div0 kind", test.in, got)
		}
	}
	for _, kind := range []int32{-1, 0, 6, 12, 17, 18, 19, 20, 21, 60} {
		_, e := BitVectorDivZeroKind(kind)
		if e != ErrASTDomain {
			t.Fatal("invalid native precondition", kind, e)
		}
	}
}

func TestASTBitVectorOriginalDivZeroDeclaration(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, width := range []uint32{1, 8, 65, 4096} {
		sort, e := m.BitVectorSort(width)
		astTestMust(t, e)
		for _, test := range []struct{ in, out int32 }{{7, 12}, {8, 13}, {9, 14}, {10, 15}, {11, 16}} {
			input, e := m.MkFamilyFuncDecl(fid, test.in, nil, []ASTHandle{sort, sort}, ASTHandle{})
			astTestMust(t, e)
			astTestPin(t, m, input)
			got, e := m.BitVectorDivZeroDeclaration(input)
			astTestMust(t, e)
			astTestPin(t, m, got)
			expected, e := m.MkFamilyFuncDecl(fid, test.out, nil, []ASTHandle{sort}, ASTHandle{})
			astTestMust(t, e)
			if got != expected {
				t.Fatal("div0 declaration identity", width, test.in)
			}
			arity, e := m.GetArity(got)
			astTestMust(t, e)
			domain, e := m.GetDomain(got, 0)
			astTestMust(t, e)
			result, e := m.GetRange(got)
			astTestMust(t, e)
			uninterpreted, e := m.IsConsideredUninterpreted(got)
			astTestMust(t, e)
			if arity != 1 || domain != sort || result != sort || !uninterpreted {
				t.Fatal("div0 declaration shape", width, test.in)
			}
			before := m.Stats()
			m.Budget().FailAfter(0)
			again, e := m.BitVectorDivZeroDeclaration(input)
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			if again != got || m.Stats() != before {
				t.Fatal("warm declaration acquired state")
			}
		}
	}
}

func TestASTBitVectorDivZeroDeclarationOnlyUsesFirstDomain(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	p, e := m.NewParameterInt(123)
	astTestMust(t, e)
	for _, family := range []FamilyID{fid, 42} {
		info, e := m.NewFuncDeclInfo(family, 7, []ASTParameter{p}, ASTFuncFlags{}, false)
		astTestMust(t, e)
		// Raw metadata intentionally gives the original div a foreign second domain,
		// unrelated result sort and extra parameter. The helper uses none of these.
		input, e := m.MkFuncDecl(astTestSymbol(t, m, "raw-div0-input"), []ASTHandle{sort, boolSort}, boolSort, info)
		astTestMust(t, e)
		astTestMust(t, m.DelFuncDeclInfo(&info))
		astTestPin(t, m, input)
		got, e := m.BitVectorDivZeroDeclaration(input)
		astTestMust(t, e)
		expected, e := m.MkFamilyFuncDecl(family, 12, nil, []ASTHandle{sort}, ASTHandle{})
		astTestMust(t, e)
		if got != expected {
			t.Fatal("did not preserve family/first domain", family)
		}
		if family == 42 && !got.IsNull() {
			t.Fatal("unregistered family was replaced with BV")
		}
	}
	astTestMust(t, m.DelParameter(&p))
	info, e := m.NewFuncDeclInfo(fid, 7, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	empty, e := m.MkFuncDecl(astTestSymbol(t, m, "empty-div0-input"), nil, boolSort, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestPin(t, m, empty)
	_, e = m.BitVectorDivZeroDeclaration(empty)
	if e != ErrASTDomain {
		t.Fatal("empty native domain adapter", e)
	}
}

func TestASTBitVectorDivZeroEveryColdReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			sort, e := m.BitVectorSort(65)
			astTestMust(t, e)
			input, e := m.MkFamilyFuncDecl(fid, 7, nil, []ASTHandle{sort, sort}, ASTHandle{})
			astTestMust(t, e)
			m.Budget().FailAfter(ordinal)
			got, body := m.BitVectorDivZeroDeclaration(input)
			m.Budget().FailAfter(-1)
			if body != nil {
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal("unexpected refusal", body)
				}
				failures++
			} else {
				success = true
				arity, e := m.GetArity(got)
				astTestMust(t, e)
				if arity != 1 {
					t.Fatal("div0 arity", arity)
				}
			}
			// The fixture's owning closure checks the actual shared budget is zero,
			// including cache/header/factory prefixes retained by a refused operation.
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("cold reservation horizon", success, failures)
	}
}
