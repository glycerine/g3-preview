package nlsat

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func atomDebugFixture(t *testing.T) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	r, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	assignmentMust(t, err)
	d, err := diagnostic.NewDebugContext(r, b)
	assignmentMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(d)
	assignmentMust(t, err)
	t.Cleanup(func() {
		_, err := diagnostic.BindProcessDebugContext(previous)
		assignmentMust(t, err)
		assignmentMust(t, d.Close())
		assignmentMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("debug storage retained", b.Used())
		}
	})
}

func TestAtomSourceAssertionsAndReachedHeaders(t *testing.T) {
	atomDebugFixture(t)
	var a Atom
	a.initialize(AtomEQ, 7)
	err := a.DecRef()
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) || a.refs != 0 {
			t.Fatal("source decrement assertion", a.refs, err)
		}
	} else {
		assignmentMust(t, err)
		if a.refs != math.MaxUint32 {
			t.Fatal("release source decrement", a.refs)
		}
	}
	_, err = AtomRootEQ.Flip()
	if diagnostic.DebugBuild && !errors.Is(err, diagnostic.ErrDebugAssertion) {
		t.Fatal("flip assertion", err)
	}
	if !diagnostic.DebugBuild {
		assignmentMust(t, err)
	}
	var ineq InequalityAtom
	storage := []InequalityFactor{{}}
	err = ineq.initialize(AtomRootLT, []polynomial.Polynomial{{}}, []bool{true}, 17, storage)
	if diagnostic.DebugBuild && !errors.Is(err, diagnostic.ErrDebugAssertion) {
		t.Fatal("inequality constructor assertion", err)
	}
	if !diagnostic.DebugBuild {
		assignmentMust(t, err)
	}
	if ineq.maxVar != 17 || len(ineq.factors) != 1 || !ineq.factors[0].Even {
		t.Fatal("constructor lost reached fields")
	}
	_, err = ineq.Polynomial(1)
	want := arena.ErrBounds
	if diagnostic.DebugBuild {
		want = diagnostic.ErrDebugAssertion
	}
	if !errors.Is(err, want) {
		t.Fatal("polynomial bounds", err)
	}
	_, err = ineq.IsOdd(1)
	if !errors.Is(err, want) {
		t.Fatal("parity bounds", err)
	}
	var root RootAtom
	err = root.initialize(AtomGT, 23, 3, polynomial.Polynomial{})
	if diagnostic.DebugBuild && !errors.Is(err, diagnostic.ErrDebugAssertion) {
		t.Fatal("root constructor assertion", err)
	}
	if !diagnostic.DebugBuild {
		assignmentMust(t, err)
	}
	if root.Variable() != 23 || root.Index() != 3 || root.MaxVariable() != 23 {
		t.Fatal("root reached fields")
	}
}

func TestSignTableSourceAssertionsAndOwnerGuards(t *testing.T) {
	atomDebugFixture(t)
	table := signTableFixture(t)
	roots := signRoots(t, table.am, []int{1, 2, 3, 4, 5, 6, 7, 8, 9})
	assignmentMust(t, table.add(roots, []int{1, -1, 1, -1, 1, -1, 1, -1, 1, -1}))
	assignmentMust(t, table.checkInvariant())
	copy := *table
	if _, err := copy.signAt(0, 0); !errors.Is(err, polynomial.ErrAlgebraicOwner) {
		t.Fatal("copied table", err)
	}
	if diagnostic.DebugBuild {
		table.sections[table.sorted[0]].position = 99
		if err := table.checkInvariant(); !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("position invariant", err)
		}
		table.sections[table.sorted[0]].position = 0
	}
	want := arena.ErrBounds
	if diagnostic.DebugBuild {
		want = diagnostic.ErrDebugAssertion
	}
	if _, err := table.rootID(0); !errors.Is(err, want) {
		t.Fatal("section guard", err)
	}
	if _, err := table.rootID(table.numCells()); !errors.Is(err, want) {
		t.Fatal("cell guard", err)
	}
	if _, err := table.root(123); !errors.Is(err, want) {
		t.Fatal("root guard", err)
	}
	zero, err := table.root(math.MaxUint32)
	assignmentMust(t, err)
	assignmentWantInteger(t, table.am, zero, 0)
	assignmentMust(t, table.close())
	assignmentMust(t, table.close())
	if _, err := table.signAt(0, 0); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed table", err)
	}
}
