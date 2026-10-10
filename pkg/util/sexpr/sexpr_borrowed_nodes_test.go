package sexpr

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestSexprBorrowedNodesKeepTraversalAndRejectEarlierGeneration(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	a := sxNumber(t, q, "23")
	atom, e := m.MkNumeral(&a, 2, 3)
	sxMust(t, e)
	tree, e := m.MkComposite([]SexprHandle{atom}, 4, 5)
	sxMust(t, e)
	sxMust(t, m.IncRef(tree))
	var saved SexprBorrowedNode
	var retained *SexprBorrowedNodes
	sxMust(t, m.WithBorrowedNodes(func(b *SexprBorrowedNodes) error {
		retained = b
		copied := *b
		if copied.Valid() {
			t.Fatal("copied lock borrow accepted")
		}
		var e error
		saved, e = b.Record(tree)
		if e != nil {
			return e
		}
		if saved.Kind != SexprComposite || saved.Length != 1 || saved.Line != 4 || saved.Pos != 5 {
			t.Fatal("source metadata changed")
		}
		child, e := b.Child(saved, 0)
		if e != nil {
			return e
		}
		record, e := b.Record(child)
		if e != nil {
			return e
		}
		if child != atom || record.Kind != SexprNumeral || record.Numeral == nil {
			t.Fatal("source numeral borrow changed")
		}
		return q.WithNumericBorrow(func(r *unumeral.RationalNumericBorrow) error {
			n, ok, e := r.Exponent(record.Numeral)
			if e != nil {
				return e
			}
			if !ok || n != 23 {
				t.Fatal("source numeral changed", n, ok)
			}
			return nil
		})
	}))
	if retained.Valid() {
		t.Fatal("traversal lock borrow survived callback")
	}
	sxMust(t, m.WithBorrowedNodes(func(b *SexprBorrowedNodes) error {
		if _, e := b.Child(saved, 0); !errors.Is(e, ErrSexprOwner) {
			t.Fatal("expired node revived", e)
		}
		return nil
	}))
	sxMust(t, m.DecRef(tree))
	sxMust(t, q.Del(&a))
}
