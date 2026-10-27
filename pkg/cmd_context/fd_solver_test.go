package cmd_context

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestCommandContextFiniteDomainSolverProofGate(t *testing.T) {
	for _, logic := range []string{"QF_FD"} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(logic, "/", proofs), func(t *testing.T) {
				c := contextFixture(t)
				contextMust(t, c.SetBoolOption("produce_proofs", proofs))
				contextMust(t, c.SetLogic(logic))
				_, err := c.Manager()
				contextMust(t, err)
				if !proofs {
					if _, ok := c.solver.(*portfolio.BoundedInt2BVSolver); !ok {
						t.Fatalf("missing retained FD stack: %T", c.solver)
					}
				} else {
					if _, ok := c.solver.(*smtcore.SMTSolver); !ok {
						t.Fatalf("FD path reached with manager proofs enabled: %T", c.solver)
					}
				}
			})
		}
	}
}

func TestCommandContextFiniteDomainStoppedConstructorPreservesManager(t *testing.T) {
	if !diagnostic.DebugBuild {
		t.Skip("SAT source destructor stop is debug only")
	}
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_FD"))
	m, err := c.Manager()
	contextMust(t, err)
	contextMust(t, c.solver.Close())
	c.solver = nil
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		m.Budget().FailAfter(ordinal)
		err = c.mkSolver(m)
		m.Budget().FailAfter(-1)
		var failure *portfolio.FDConstructionFailure
		if errors.As(err, &failure) {
			// Manager's construction boundary records this exact error on return.
			c.initializationError = err
			if stop := c.Close(); !errors.Is(stop, failure) {
				t.Fatal("context lost factory receipt", ordinal, stop)
			}
			if c.manager != m || m.Closed {
				t.Fatal("manager retired before stopped solver")
			}
			contextMust(t, failure.Cleanup())
			if failure.CleanupPending() {
				t.Fatal("receipt still owns stack")
			}
			contextMust(t, c.Close())
			return
		}
		if c.solver != nil {
			contextMust(t, c.solver.Close())
			c.solver = nil
		}
	}
	t.Fatal("no stopped constructor receipt reached")
}
