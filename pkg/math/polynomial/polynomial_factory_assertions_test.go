package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFactoryDirectoryAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C"} {
		t.Run(action, func(t *testing.T) {
			m := polynomialCellFixture(t)
			seven := unumeral.NewInt(7)
			p, e := m.MkConstantTakeCopy(&seven)
			serviceMust(t, e)
			info, e := m.Info(p)
			serviceMust(t, e)
			id := info.ID
			before := m.polynomials[id]
			polynomialAssertionActionsFixture(t, action)
			e = m.publishDirectory(p, id)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || m.polynomials[id] != before {
					t.Fatal("source directory assertion did not stop before publication", e, m.polynomials[id], before)
				}
			} else {
				serviceMust(t, e)
				if m.polynomials[id] != p {
					t.Fatal("Continue/release did not execute source publication", m.polynomials[id], p)
				}
			}
			serviceMust(t, m.CleanupValue(&p))
			polynomialPhysicalAudit(t, m)
		})
	}
}
