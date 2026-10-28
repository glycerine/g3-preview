package nlsat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func TestUnivariateConjunctionRejectsMixedVariablesAndOwnsDomain(t *testing.T) {
	e := evaluatorFixture(t)
	mixed := evaluatorPolynomial(t, e, evaluatorTerm{coefficient: 1, y: 1}, evaluatorTerm{coefficient: 1, x: 1})
	square := evaluatorPolynomial(t, e, evaluatorTerm{coefficient: 1, x: 2}, evaluatorTerm{coefficient: -2})
	u, err := NewUnivariateConjunction(e.am, e.pm, 1)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, u.Close()) })
	if err = u.Add(mixed, AtomEQ, false); !errors.Is(err, ErrSearchOwner) {
		t.Fatal("mixed polynomial admitted", err)
	}
	assignmentMust(t, u.Add(square, AtomEQ, false))
	var witness polynomial.Algebraic
	defer func() { assignmentMust(t, e.am.Del(&witness)) }()
	sat, err := u.Check(&witness)
	assignmentMust(t, err)
	if !sat {
		t.Fatal("sqrt(2) domain unexpectedly empty")
	}
	assignmentMust(t, u.Add(square, AtomEQ, true))
	sat, err = u.Check(&witness)
	assignmentMust(t, err)
	if sat {
		t.Fatal("equality and disequality admitted a witness")
	}
	assignmentMust(t, u.Close())
	if _, err = u.Check(&witness); !errors.Is(err, ErrSearchOwner) {
		t.Fatal("closed owner accepted", err)
	}
	assignmentMust(t, u.Close())
}
