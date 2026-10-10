package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func astTestSymbolGraph(s *usymbol.Manager) uint64 {
	x := s.Stats()
	d := x.Region.Descriptors
	return x.OwnerBytes + x.Table.OwnerBytes + x.Table.TableBytes + x.Table.DirectoryBytes + x.Region.OwnerBytes + x.Region.PayloadBytes + x.Region.PageHeaderBytes + x.Region.DirectoryBytes + x.Region.LogBytes + x.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func TestASTConstructorEveryPredictablePrefixOwnsNoLostGraph(t *testing.T) {
	var failures int
	successful := false
	for after := int64(0); after < 256; after++ {
		s, q, b := astTestOwners(t)
		other := b.Used() - astTestSymbolGraph(s)
		numBefore, e := q.Stats()
		astTestMust(t, e)
		b.FailAfter(after)
		m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
		if e == nil {
			b.FailAfter(-1)
			astTestMust(t, m.Close())
			if b.Used() != other+astTestSymbolGraph(s) {
				t.Fatal("successful owner teardown graph mismatch", after, b.Used())
			}
			successful = true
			break
		}
		failures++
		if m != nil {
			t.Fatal("failed owner published", after)
		}
		// Check before Close of either borrowed owner: caller symbol spelling/cold
		// allocation prefixes may persist, but no AST page/table/payload owner may.
		if b.Used() != other+astTestSymbolGraph(s) {
			t.Fatal("untracked constructor prefix", after, e, b.Used(), other+astTestSymbolGraph(s))
		}
		numAfter, e := q.Stats()
		astTestMust(t, e)
		if numAfter != numBefore {
			t.Fatal("constructor retained numeral roots", after, numBefore, numAfter)
		}
		b.FailAfter(-1)
	}
	if !successful || failures < 20 {
		t.Fatal("did not cross actual constructor reserve branches", failures)
	}
}
