package smtcore

import (
	"fmt"
	"math/big"
	"testing"
)

// Bellman-Ford recomputed from asserted edges is independent of the source
// dense incremental closure and checks both finite and disconnected cells.
func smtCheckDenseClosure(t *testing.T, g *smtDenseDiffMatrix) {
	t.Helper()
	for source := range g.rows {
		dist := make([]*smtInfRational, len(g.rows))
		dist[source] = &smtInfRational{}
		for pass := 1; pass < len(g.rows); pass++ {
			for _, e := range g.edges[1:] {
				if dist[e.source] == nil {
					continue
				}
				value := new(smtInfRational).Add(dist[e.source], &e.offset)
				if dist[e.target] == nil || value.Cmp(dist[e.target]) < 0 {
					dist[e.target] = value
				}
			}
		}
		for target := range g.rows {
			value, connected := g.distance(source, target)
			if connected != (dist[target] != nil) || connected && value.Cmp(dist[target]) != 0 {
				t.Fatal("incremental closure differs from Bellman-Ford", source, target)
			}
			if !connected {
				continue
			}
			antecedents, err := g.antecedents(source, target)
			astTestMust(t, err)
			var sum smtInfRational
			for _, literal := range antecedents {
				found := false
				for _, edge := range g.edges[1:] {
					if edge.literal == literal {
						sum.Add(&sum, &edge.offset)
						found = true
						break
					}
				}
				if !found {
					t.Fatal("distance explanation contains an unasserted literal", literal)
				}
			}
			if sum.Cmp(&value) != 0 {
				t.Fatal("distance explanation has the wrong total weight", source, target)
			}
		}
	}
	values := g.assignment()
	for _, edge := range g.edges[1:] {
		var difference smtInfRational
		difference.Sub(&values[edge.target], &values[edge.source])
		if difference.Cmp(&edge.offset) > 0 {
			t.Fatal("dense model violates an asserted edge", edge.source, edge.target)
		}
	}
}

func TestSMTDenseDifferenceClosureExplanationsAndRollback(t *testing.T) {
	for order := 0; order < 7; order++ {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			g := newSMTDenseDiffMatrix()
			for i := 0; i < 6; i++ {
				g.addVariable()
			}
			g.push()
			// Nonnegative reduced weights provide consistent rational and
			// infinitesimal inputs with negative original edge weights.
			for i := 0; i < 30; i++ {
				index := (i*7 + order*11) % 30
				source := index / 5
				target := index % 5
				if target >= source {
					target++
				}
				offset := newSMTInfRational(big.NewRat(int64((target-source)*7+index%4), 3), big.NewRat(int64(index%3), 1))
				ok, _, err := g.addEdge(source, target, &offset, SMTLiteral(index+2))
				astTestMust(t, err)
				if !ok {
					t.Fatal("consistent edge rejected", index)
				}
				smtCheckDenseClosure(t, g)
			}
			reverse, found := g.distance(1, 0)
			if !found {
				t.Fatal("test graph should be connected")
			}
			var conflict smtInfRational
			conflict.Neg(&reverse)
			conflict.infinitesimal.Sub(&conflict.infinitesimal, big.NewRat(1, 1))
			edges, trail := len(g.edges), len(g.trail)
			ok, reason, err := g.addEdge(0, 1, &conflict, SMTLiteral(100))
			astTestMust(t, err)
			if ok || len(reason) < 2 || reason[len(reason)-1] != SMTLiteral(100) || len(g.edges) != edges || len(g.trail) != trail {
				t.Fatal("negative infinitesimal cycle was not rejected atomically", reason)
			}
			g.push()
			v := g.addVariable()
			zero := smtInfRational{}
			ok, _, err = g.addEdge(v, 0, &zero, SMTLiteral(101))
			astTestMust(t, err)
			if !ok {
				t.Fatal("new-variable edge rejected")
			}
			smtCheckDenseClosure(t, g)
			astTestMust(t, g.pop(1))
			smtCheckDenseClosure(t, g)
			astTestMust(t, g.pop(1))
			smtCheckDenseClosure(t, g)
			if len(g.edges) != 1 || len(g.trail) != 0 || len(g.rows) != 6 {
				t.Fatal("scope pop retained child closure state")
			}
		})
	}
}
