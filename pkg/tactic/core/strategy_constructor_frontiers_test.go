package core

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
)

// These strategies are composed repeatedly by QF_LIA. Each failed prefix must
// release its partial implementation without calling its normal rewrite API.
func TestCoreStrategyPrerequisiteConstructorFrontiers(t *testing.T) {
	for _, name := range []string{"simplify", "propagate-values", "solve-eqs"} {
		t.Run(name, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 256; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					m, c, g, services := solveEqFixture(t)
					simplifyMust(t, m.RegisterDeclarationPlugins())
					m.Budget().FailAfter(stop)
					var owner io.Closer
					var err error
					switch name {
					case "simplify":
						value, e := NewSimplifyTactic(m, c, g, nil)
						err = e
						if value != nil {
							owner = value
						}
					case "propagate-values":
						value, e := NewPropagateValuesTactic(m, c, g, nil)
						err = e
						if value != nil {
							owner = value
						}
					case "solve-eqs":
						value, e := NewSolveEquationsTactic(m, c, g, nil, services)
						err = e
						if value != nil {
							owner = value
						}
					}
					m.Budget().FailAfter(-1)
					if owner != nil {
						simplifyMust(t, owner.Close())
					}
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						simplifyMust(t, failure.Cleanup())
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if success {
					break
				}
			}
			if !success {
				t.Fatal("no successful constructor reservation frontier")
			}
		})
	}
}
