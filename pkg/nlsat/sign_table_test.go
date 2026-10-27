package nlsat

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func signTableFixture(t *testing.T) *signTable {
	t.Helper()
	am, _ := assignmentFixture(t)
	table, err := newSignTable(am)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, table.close()) })
	return table
}
func signRoots(t *testing.T, am *polynomial.AlgebraicManager, values []int) *polynomial.AlgebraicVector {
	t.Helper()
	roots, err := am.NewVector()
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, roots.Close()) })
	for _, n := range values {
		var value polynomial.Algebraic
		num := unumeral.NewInt(int32(n))
		assignmentMust(t, am.SetInteger(&value, &num))
		assignmentMust(t, roots.PushCopy(&value))
		assignmentMust(t, am.Del(&value))
	}
	return roots
}

type signModel struct{ roots, signs []int }

func checkSignTable(t *testing.T, table *signTable, models []signModel) {
	t.Helper()
	all := map[int]bool{}
	for _, m := range models {
		for _, r := range m.roots {
			all[r] = true
		}
	}
	var ordered []int
	for r := range all {
		ordered = append(ordered, r)
	}
	sort.Ints(ordered)
	if table.numCells() != uint32(2*len(ordered)+1) {
		t.Fatal("distinct root count", len(ordered), table.numCells())
	}
	for c := uint32(0); c < table.numCells(); c++ {
		// Represent each sector by a half-integer adjacent to its lower root;
		// represent each section by the integer root itself.
		x := 0
		if c == 0 {
			x = -1000
		} else if c%2 == 1 {
			x = 2 * ordered[c/2]
		} else {
			x = 2*ordered[c/2-1] + 1
		}
		for i, m := range models {
			idx := sort.Search(len(m.roots), func(i int) bool { return 2*m.roots[i] >= x })
			want := 0
			if idx == len(m.roots) || 2*m.roots[idx] != x {
				if m.signs[idx] < 0 {
					want = -1
				} else if m.signs[idx] > 0 {
					want = 1
				}
			}
			got, err := table.signAt(uint32(i), c)
			assignmentMust(t, err)
			if got != want {
				t.Fatalf("cell %d point %d/2 polynomial %d: got %d want %d roots %v", c, x, i, got, want, m.roots)
			}
		}
		if c%2 == 1 {
			id, err := table.rootID(c)
			assignmentMust(t, err)
			r, err := table.root(id)
			assignmentMust(t, err)
			assignmentWantInteger(t, table.am, r, int32(ordered[c/2]))
			if table.sections[id].position != c/2 {
				t.Fatal("section position")
			}
		}
	}
}

func TestSignTableMergedRootsIndependentCellSigns(t *testing.T) {
	table := signTableFixture(t)
	rng := rand.New(rand.NewPCG(71213, 41947))
	for trial := 0; trial < 16; trial++ {
		var models []signModel
		for step := 0; step < 12; step++ {
			n := []int{0, 1, 7, 8, 9, 16}[step%6]
			perm := rng.Perm(32)
			rs := append([]int(nil), perm[:n]...)
			sort.Ints(rs)
			signs := make([]int, n+1)
			for i := range signs {
				signs[i] = []int{math.MinInt, -113, -1, 0, 1, 719, math.MaxInt}[rng.IntN(7)]
			}
			roots := signRoots(t, table.am, rs)
			assignmentMust(t, table.add(roots, signs))
			assignmentMust(t, roots.Close()) // table must own independent roots
			models = append(models, signModel{rs, signs})
			checkSignTable(t, table, models)
		}
		assignmentMust(t, table.addConst(math.MinInt))
		models = append(models, signModel{nil, []int{-1}})
		checkSignTable(t, table, models)
		assignmentMust(t, table.reset())
		if table.numCells() != 1 || len(table.info) != 0 {
			t.Fatal("reset did not clear cells/info")
		}
	}
}

func TestSignTableStableIDsDisplayAndIrrationalCopies(t *testing.T) {
	table := signTableFixture(t)
	roots := signRoots(t, table.am, []int{2, 4})
	assignmentMust(t, table.add(roots, []int{-100, 17, -93}))
	assignmentMust(t, roots.Close())
	roots = signRoots(t, table.am, []int{1, 2, 3})
	assignmentMust(t, table.add(roots, []int{5, -7, 0, 32}))
	assignmentMust(t, roots.Close())
	if fmt.Sprint(table.sorted) != "[2 0 3 1]" {
		t.Fatal("stable section IDs", table.sorted)
	}
	var out bytes.Buffer
	assignmentMust(t, table.display(&out, true))
	want := "sections:\n  1 < 2 < 3 < 4\npoly_info:\n  roots: 0 1, signs: - + -\n  roots: 2 0 3, signs: + - 0 +\n"
	if out.String() != want {
		t.Fatalf("raw display\n%q\nwant\n%q", out.String(), want)
	}
	out.Reset()
	assignmentMust(t, table.display(&out, false))
	want = "sections:\n  1 < 2 < 3 < 4\nsign variations:\n  - - - 0 + + + 0 -\n  + 0 - 0 0 0 + + +\n"
	if out.String() != want {
		t.Fatalf("display\n%q\nwant\n%q", out.String(), want)
	}
	// An algebraic root contributed by two different polynomial root arrays
	// has one section and survives retirement of the input owners.
	assignmentMust(t, table.reset())
	root := assignmentInteger(t, table.am, 2)
	assignmentMust(t, table.am.Root(root, 2, root))
	for i := 0; i < 2; i++ {
		v, err := table.am.NewVector()
		assignmentMust(t, err)
		assignmentMust(t, v.PushCopy(root))
		assignmentMust(t, table.add(v, []int{-1, 1}))
		assignmentMust(t, v.Close())
	}
	assignmentMust(t, table.am.Del(root))
	if table.numCells() != 3 || len(table.sections) != 1 {
		t.Fatal("irrational duplicate not merged")
	}
	var square polynomial.Algebraic
	assignmentMust(t, table.am.Power(&table.sections[0].root, 2, &square))
	assignmentWantInteger(t, table.am, &square, 2)
	assignmentMust(t, table.am.Del(&square))
}

func TestSignTableAllocationRefusalsResetAndRetry(t *testing.T) {
	for _, stop := range []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 16, 24, 32, 64, -1} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			table := signTableFixture(t)
			a := signRoots(t, table.am, []int{0, 4, 8})
			assignmentMust(t, table.add(a, []int{-9, 3, -7, 101}))
			b := signRoots(t, table.am, []int{1, 2, 4, 5, 6, 7, 9, 10})
			table.budget.FailAfter(stop)
			err := table.add(b, []int{1, -1, 1, -1, 1, -1, 1, -1, 1})
			table.budget.FailAfter(-1)
			assignmentMust(t, table.am.CleanupRetained())
			var physical umemory.Failure
			if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
				t.Fatal("unexpected refusal", err)
			}
			if err != nil {
				if _, e := table.signAt(0, 0); !errors.Is(e, polynomial.ErrAlgebraicPending) {
					t.Fatal("partial table usable", e)
				}
			}
			assignmentMust(t, table.reset())
			assignmentMust(t, table.add(b, []int{1, -1, 1, -1, 1, -1, 1, -1, 1}))
			checkSignTable(t, table, []signModel{{[]int{1, 2, 4, 5, 6, 7, 9, 10}, []int{1, -1, 1, -1, 1, -1, 1, -1, 1}}})
		})
	}
}
