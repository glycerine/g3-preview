package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTBitVectorDiv0FunctionsUseCoreCongruence(t *testing.T) {
	for kind := int32(12); kind <= 16; kind++ {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			m, _, solver := smtBVSolver(t)
			family, err := m.RegisterBitVectorPlugin()
			astTestMust(t, err)
			sort, err := m.BitVectorSort(4)
			astTestMust(t, err)
			var arguments []ast.ASTHandle
			for _, name := range []string{"div0-left", "div0-right"} {
				arg, err := m.MkConst(astTestSymbol(t, m, name), sort)
				astTestMust(t, err)
				astTestMust(t, m.IncRef(arg))
				t.Cleanup(func() { astTestMust(t, m.DecRef(arg)) })
				arguments = append(arguments, arg)
			}
			var applications []ast.ASTHandle
			for _, arg := range arguments {
				term, err := m.MkFamilyApp(family, kind, nil, []ast.ASTHandle{arg})
				astTestMust(t, err)
				astTestMust(t, m.IncRef(term))
				t.Cleanup(func() { astTestMust(t, m.DecRef(term)) })
				applications = append(applications, term)
				for bit := uint32(0); bit < 4; bit++ {
					expression, err := m.MkBitVectorBit2Bool(bit, arg)
					astTestMust(t, err)
					expression, err = m.MkNot(expression)
					astTestMust(t, err)
					astTestMust(t, solver.Assert(expression))
				}
			}
			equality, err := m.MkEq(applications[0], applications[1])
			astTestMust(t, err)
			different, err := m.MkNot(equality)
			astTestMust(t, err)
			astTestMust(t, solver.Assert(different))
			status, err := solver.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.False {
				t.Fatal("fixed-equal arguments yielded different div0 values", status)
			}
		})
	}
}
