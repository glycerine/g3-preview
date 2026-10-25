package smtcore

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTPBComplementVerifyRetainsNormalizationPrefix(t *testing.T) {
	for _, bound := range []int64{0, 3} {
		var args smtPBArgs
		args.bound.SetInt64(bound)
		for _, v := range []int32{1, 2} {
			args.terms = append(args.terms, smtPBTerm{NewSMTLiteral(v, false), *big.NewRat(1, 1)})
		}
		err := args.negate()
		if smtcoreSourceDebug {
			if err != ast.ErrASTDomain {
				t.Fatal("source VERIFY did not check the normalization result", bound, err)
			}
		} else {
			astTestMust(t, err)
		}
		// VERIFY evaluates normalize before the debug assertion. Both
		// constant complements clear terms, with false represented by 1.
		want := int64(1)
		if bound == 3 {
			want = 0
		}
		if len(args.terms) != 0 || args.bound.Cmp(big.NewRat(want, 1)) != 0 {
			t.Fatal("source normalization prefix was lost", bound, args.bound.String(), len(args.terms))
		}
	}
}

func TestSMTPBInternalizeTermSourceUnreachableGuard(t *testing.T) {
	_, s, pb, _, _ := smtPBWatchFixture(t, []int64{1, 1}, 2, false)
	before := len(s.boolExpr)
	accepted, err := pb.InternalizeTerm(ast.ASTHandle{})
	if accepted || len(s.boolExpr) != before {
		t.Fatal("unreachable term hook changed Boolean state")
	}
	if smtcoreSourceDebug {
		if err != ast.ErrASTDomain {
			t.Fatal("source UNREACHABLE was not checked", err)
		}
	} else {
		astTestMust(t, err)
	}
}
