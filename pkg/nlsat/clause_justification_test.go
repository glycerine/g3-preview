package nlsat

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestClausePlacementMutablePrefixCapacityAndAssumptions(t *testing.T) {
	m := assumptionFixture(t)
	as := assumptionLeaf(t, m, AttachedAssumption(&AssumptionAttachment{}))
	defer func() { assignmentMust(t, m.decRef(as)) }()
	input := []sat.Literal{sat.MakeLiteral(7, true), sat.MakeLiteral(11, false), sat.MakeLiteral(13, true)}
	storage := make([]sat.Literal, len(input))
	var c Clause
	assignmentMust(t, c.initialize(29, input, true, as, storage))
	if c.ID() != 29 || !c.IsLearned() || c.Size() != 3 || c.Assumptions() != as || as.refs != 1 || c.Activity() != 0 {
		t.Fatal("source constructor fields/reference ownership")
	}
	input[0] = sat.MakeLiteral(17, false)
	if !c.ContainsLiteral(sat.MakeLiteral(7, true)) || c.ContainsLiteral(sat.MakeLiteral(7, false)) || !c.ContainsVariable(7) {
		t.Fatal("source membership/sign semantics")
	}
	borrow, err := c.BorrowLiteral(1)
	assignmentMust(t, err)
	*borrow = sat.MakeLiteral(19, true)
	actual, err := c.Literal(1)
	assignmentMust(t, err)
	if actual != *borrow || storage[1] != *borrow || c.ContainsVariable(11) {
		t.Fatal("mutable literal was a snapshot")
	}
	before := c.storageBytes()
	assignmentMust(t, c.Shrink(1))
	if c.Size() != 1 || c.capacity != 3 || c.storageBytes() != before || c.ContainsVariable(19) || c.ContainsVariable(13) || len(c.Literals()) != 1 || c.Assumptions() != as {
		t.Fatal("shrink changed allocation or visible/hidden literals")
	}
	if storage[1] != sat.MakeLiteral(19, true) || storage[2] != sat.MakeLiteral(13, true) {
		t.Fatal("shrink erased retained storage")
	}
	c.SetActivity(math.MaxUint32)
	c.IncActivity()
	if c.Activity() != 0 {
		t.Fatal("activity must wrap at uint32")
	}
	assignmentMust(t, c.Shrink(0))
	if c.ContainsVariable(7) || c.ContainsLiteral(sat.MakeLiteral(7, true)) {
		t.Fatal("zero active prefix")
	}
}
func TestClauseIndependentMutationAndMembershipModels(t *testing.T) {
	rng := rand.New(rand.NewPCG(33893, 72997))
	for trial := 0; trial < 300; trial++ {
		n := rng.IntN(18)
		original := make([]sat.Literal, n)
		for i := range original {
			original[i] = sat.LiteralFromUint(rng.Uint32())
		}
		var c Clause
		assignmentMust(t, c.initialize(uint32(trial), original, trial%2 == 0, nil, make([]sat.Literal, n)))
		model := append([]sat.Literal(nil), original...)
		for step := 0; step < 16; step++ {
			if len(model) > 0 && rng.IntN(2) == 0 {
				idx := rng.IntN(len(model))
				value := sat.LiteralFromUint(rng.Uint32())
				borrow, err := c.BorrowLiteral(uint32(idx))
				assignmentMust(t, err)
				*borrow = value
				model[idx] = value
			} else {
				size := rng.IntN(len(model) + 1)
				assignmentMust(t, c.Shrink(uint32(size)))
				model = model[:size]
			}
			if c.Size() != uint32(len(model)) || c.capacity != uint32(n) {
				t.Fatal("model size/capacity")
			}
			for _, query := range append(append([]sat.Literal(nil), original...), sat.LiteralFromUint(rng.Uint32())) {
				wantLiteral, wantVariable := false, false
				for _, v := range model {
					wantLiteral = wantLiteral || v == query
					wantVariable = wantVariable || v.Variable() == query.Variable()
				}
				if c.ContainsLiteral(query) != wantLiteral || c.ContainsVariable(query.Variable()) != wantVariable {
					t.Fatal("independent clause membership", trial, step)
				}
			}
		}
	}
}
func TestJustificationTagsIdentityCopiesAndRetirement(t *testing.T) {
	m := assumptionFixture(t)
	var zero Justification
	decision, err := NewDecisionJustification()
	assignmentMust(t, err)
	var c Clause
	clause, err := NewClauseJustification(&c)
	assignmentMust(t, err)
	nullClause, err := NewClauseJustification(nil)
	assignmentMust(t, err)
	if !zero.IsNull() || !decision.IsDecision() || !clause.IsClause() || nullClause.IsNull() || !nullClause.IsClause() {
		t.Fatal("tagged nil/source kinds")
	}
	if clause == nullClause || decision == zero || clause != clause {
		t.Fatal("tagged identity")
	}
	borrow, err := clause.Clause()
	assignmentMust(t, err)
	if borrow != &c {
		t.Fatal("clause identity")
	}
	for _, j := range []Justification{zero, decision, clause, nullClause} {
		assignmentMust(t, DeleteJustification(nil, j))
	}
	for _, n := range []int{0, 1, 7, 128} {
		input := make([]sat.Literal, n)
		for i := range input {
			input[i] = sat.LiteralFromUint(uint32(i*7 + 3))
		}
		j, err := NewLazyJustification(m.allocator, input)
		assignmentMust(t, err)
		if !j.IsLazy() || j.Kind() != LazyJustificationKind || j.IsClause() || j.IsDecision() || j.IsNull() {
			t.Fatal("lazy tag")
		}
		l, err := j.Lazy()
		assignmentMust(t, err)
		copyJ := j
		if copyJ != j {
			t.Fatal("reason copy acquired different identity")
		}
		size, err := l.Size()
		assignmentMust(t, err)
		if size != uint32(n) {
			t.Fatal("lazy count")
		}
		want := append([]sat.Literal(nil), input...)
		if n > 0 {
			input[0] = sat.MakeLiteral(99, false)
		}
		got, err := l.Literals()
		assignmentMust(t, err)
		if !reflect.DeepEqual(got, want) && !(len(got) == 0 && len(want) == 0) {
			t.Fatal("lazy constructor borrowed input array")
		}
		for i := range got {
			v, err := l.Literal(uint32(i))
			assignmentMust(t, err)
			if v != want[i] {
				t.Fatal("lazy literal")
			}
		}
		copiedOwner := *l
		if _, err = copiedOwner.Size(); !errors.Is(err, umemory.ErrAllocatorOwner) {
			t.Fatal("copied lazy owner", err)
		}
		assignmentMust(t, DeleteJustification(m.allocator, j))
		assignmentMust(t, DeleteJustification(m.allocator, j))
		if _, err = l.Size(); !errors.Is(err, arena.ErrClosed) {
			t.Fatal("retired lazy owner", err)
		}
	}
}
func TestLazyJustificationAllocationReservationPrefixes(t *testing.T) {
	for _, stop := range []int64{0, 1, 2, 3, 4, 5, 8, 12, 16, 32, -1} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			m := assumptionFixture(t)
			input := []sat.Literal{3, 5, 7, 11, 13}
			m.budget.FailAfter(stop)
			j, err := NewLazyJustification(m.allocator, input)
			m.budget.FailAfter(-1)
			assignmentMust(t, DeleteJustification(m.allocator, j))
			var physical umemory.Failure
			if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
				t.Fatal("unexpected lazy allocation failure", err)
			}
			j, err = NewLazyJustification(m.allocator, input)
			assignmentMust(t, err)
			assignmentMust(t, DeleteJustification(m.allocator, j))
		})
	}
}
func TestClauseLazyAndAssumptionSourceGuards(t *testing.T) {
	atomDebugFixture(t)
	m := assumptionFixture(t)
	a := assumptionLeaf(t, m, Assumption{})
	if err := m.close(); !errors.Is(err, ErrAssumptionLive) {
		t.Fatal("accepted live set", err)
	}
	copyManager := *m
	if err := copyManager.incRef(a); !errors.Is(err, ErrAssumptionOwner) {
		t.Fatal("copied assumption owner", err)
	}
	a.refs = assumptionMaxRefs
	assignmentMust(t, m.incRef(a))
	if a.refs != 0 {
		t.Fatal("source 30-bit reference wrap")
	}
	a.refs = 1
	var c Clause
	assignmentMust(t, c.initialize(0, []sat.Literal{3}, false, a, make([]sat.Literal, 1)))
	want := arena.ErrBounds
	if diagnostic.DebugBuild {
		want = diagnostic.ErrDebugAssertion
	}
	if _, err := c.Literal(1); !errors.Is(err, want) {
		t.Fatal("const clause index", err)
	}
	if _, err := c.BorrowLiteral(1); !errors.Is(err, want) {
		t.Fatal("mutable clause index", err)
	}
	err := c.Shrink(2)
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) {
			t.Fatal("shrink assertion", err)
		}
	} else {
		assignmentMust(t, err)
	}
	if c.Size() != 1 {
		t.Fatal("oversized shrink grew clause")
	}
	j, err := NewLazyJustification(m.allocator, []sat.Literal{7})
	assignmentMust(t, err)
	l, err := j.Lazy()
	assignmentMust(t, err)
	if _, err = l.Literal(1); !errors.Is(err, want) {
		t.Fatal("lazy index", err)
	}
	assignmentMust(t, DeleteJustification(m.allocator, j))
	assignmentMust(t, m.decRef(a))
	if err = m.incRef(a); !errors.Is(err, ErrAssumptionOwner) {
		t.Fatal("retired assumption node", err)
	}
}
