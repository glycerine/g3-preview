package nlsat

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"sort"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func assumptionFixture(t *testing.T) *assumptionManager {
	t.Helper()
	return assumptionFixtureWithProfile(t, diagnostic.DebugBuild)
}
func assumptionFixtureWithProfile(t *testing.T, debug bool) *assumptionManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	rt, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	assignmentMust(t, err)
	a, err := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: debug}})
	assignmentMust(t, err)
	m, err := newAssumptionManager(a)
	assignmentMust(t, err)
	t.Cleanup(func() {
		b.FailAfter(-1)
		assignmentMust(t, m.close())
		assignmentMust(t, a.Close())
		assignmentMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("assumption fixture retained bytes", b.Used())
		}
	})
	return m
}

func TestAssumptionDeletionSourceOrderThroughAllocatorReuse(t *testing.T) {
	m := assumptionFixtureWithProfile(t, false)
	a := assumptionLeaf(t, m, Assumption{})
	b := assumptionLeaf(t, m, Assumption{})
	c := assumptionLeaf(t, m, Assumption{})
	bc := assumptionJoin(t, m, b, c)
	root := assumptionJoin(t, m, a, bc)
	// del decrements children left-to-right, frees the current parent, then
	// takes queued children LIFO. Expected free order is root,bc,c,b,a; the
	// real allocator's reuse order is therefore a,b,c,bc,root.
	want := []uint64{a.raw.LogicalAddress(), b.raw.LogicalAddress(), c.raw.LogicalAddress(), bc.raw.LogicalAddress(), root.raw.LogicalAddress()}
	for _, d := range []*AssumptionSet{a, b, c, bc} {
		assignmentMust(t, m.decRef(d))
	}
	assignmentMust(t, m.decRef(root))
	var next []*AssumptionSet
	for i, address := range want {
		d := assumptionLeaf(t, m, Assumption{})
		next = append(next, d)
		if d.raw.LogicalAddress() != address {
			t.Fatal("source deletion order changed allocator reuse", i, d.raw.LogicalAddress(), address)
		}
	}
	for _, d := range next {
		assignmentMust(t, m.decRef(d))
	}
}
func assumptionLeaf(t *testing.T, m *assumptionManager, v Assumption) *AssumptionSet {
	t.Helper()
	d, err := m.mkLeaf(v)
	assignmentMust(t, err)
	assignmentMust(t, m.incRef(d))
	return d
}
func assumptionJoin(t *testing.T, m *assumptionManager, a, b *AssumptionSet) *AssumptionSet {
	t.Helper()
	d, err := m.mkJoin(a, b)
	assignmentMust(t, err)
	assignmentMust(t, m.incRef(d))
	return d
}

func TestAssumptionIdentitySourceBreadthFirstAndDuplicateLeaves(t *testing.T) {
	m := assumptionFixture(t)
	tokens := make([]*AssumptionAttachment, 1024)
	identities := map[*AssumptionAttachment]bool{}
	for i := range tokens {
		tokens[i] = &AssumptionAttachment{}
		if identities[tokens[i]] {
			t.Fatal("attachment identities alias")
		}
		identities[tokens[i]] = true
	}
	input := []sat.Literal{sat.MakeLiteral(7, false), sat.MakeLiteral(7, false)}
	x, y := literalAssumption(&input[0]), literalAssumption(&input[1])
	if x == y || x.InputLiteral() != &input[0] || y.InputLiteral() != &input[1] {
		t.Fatal("literal identity collapsed to value")
	}
	if !AttachedAssumption(nil).IsNull() || !literalAssumption(nil).IsNull() {
		t.Fatal("null pointer identity")
	}
	a := assumptionLeaf(t, m, AttachedAssumption(tokens[0]))
	b := assumptionLeaf(t, m, x)
	c := assumptionLeaf(t, m, y)
	duplicate := assumptionLeaf(t, m, a.value)
	ab := assumptionJoin(t, m, a, b)
	root := assumptionJoin(t, m, ab, c)
	var out []Assumption
	assignmentMust(t, m.linearize(root, &out))
	want := []Assumption{y, a.value, x}
	if !reflect.DeepEqual(out, want) {
		t.Fatal("source BFS order", out, want)
	}
	// A shared node appears once; a distinct leaf carrying the same value does
	// appear again. Existing output is retained, rather than globally deduplicated.
	diamond := assumptionJoin(t, m, ab, root)
	withDuplicate := assumptionJoin(t, m, diamond, duplicate)
	out = []Assumption{a.value}
	assignmentMust(t, m.linearize(withDuplicate, &out))
	want = []Assumption{a.value, a.value, a.value, x, y}
	if !reflect.DeepEqual(out, want) {
		t.Fatal("node/value identity or prefix", out, want)
	}
	for _, value := range []Assumption{a.value, x, y, AttachedAssumption(tokens[2])} {
		got, err := m.contains(withDuplicate, value)
		assignmentMust(t, err)
		if got != (value != AttachedAssumption(tokens[2])) {
			t.Fatal("contains", value, got)
		}
		for d := range m.live {
			if d.marked {
				t.Fatal("successful traversal left marks")
			}
		}
	}
	// Alias paths acquire no factory reference.
	before := a.refs
	for _, pair := range [][2]*AssumptionSet{{nil, a}, {a, nil}, {a, a}} {
		alias, err := m.mkJoin(pair[0], pair[1])
		assignmentMust(t, err)
		if alias != a || a.refs != before {
			t.Fatal("join alias ownership")
		}
	}
	input[1] = sat.MakeLiteral(19, true)
	if *y.InputLiteral() != input[1] {
		t.Fatal("input assumption stored a copied literal")
	}
	for _, d := range []*AssumptionSet{withDuplicate, diamond, root, ab, duplicate, c, b, a} {
		assignmentMust(t, m.decRef(d))
	}
}

type assumptionModelNode struct {
	value    Assumption
	children [2]*AssumptionSet
	leaf     bool
}
type assumptionModelPath struct {
	node *AssumptionSet
	path string
}

// The oracle enumerates root-to-leaf paths and sorts by (distance, edge path).
// It does not reproduce the implementation's marked-node queue.
func assumptionModelLeaves(root *AssumptionSet, model map[*AssumptionSet]assumptionModelNode) []Assumption {
	if root == nil {
		return nil
	}
	var paths []assumptionModelPath
	var visit func(*AssumptionSet, string)
	visit = func(d *AssumptionSet, path string) {
		n := model[d]
		if n.leaf {
			paths = append(paths, assumptionModelPath{d, path})
			return
		}
		visit(n.children[0], path+"0")
		visit(n.children[1], path+"1")
	}
	visit(root, "")
	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i].path) != len(paths[j].path) {
			return len(paths[i].path) < len(paths[j].path)
		}
		return paths[i].path < paths[j].path
	})
	seen := map[*AssumptionSet]bool{}
	var out []Assumption
	for _, p := range paths {
		if !seen[p.node] {
			seen[p.node] = true
			out = append(out, model[p.node].value)
		}
	}
	return out
}
func assumptionModelOwners(t *testing.T, m *assumptionManager, held []*AssumptionSet, model map[*AssumptionSet]assumptionModelNode) {
	t.Helper()
	reachable := map[*AssumptionSet]bool{}
	var visit func(*AssumptionSet)
	visit = func(d *AssumptionSet) {
		if d == nil || reachable[d] {
			return
		}
		reachable[d] = true
		n := model[d]
		if !n.leaf {
			visit(n.children[0])
			visit(n.children[1])
		}
	}
	refs := map[*AssumptionSet]uint32{}
	for _, r := range held {
		visit(r)
		if r != nil {
			refs[r]++
		}
	}
	for d := range reachable {
		n := model[d]
		if !n.leaf {
			refs[n.children[0]]++
			refs[n.children[1]]++
		}
	}
	if len(reachable) != len(m.live) {
		t.Fatal("live ownership differs", len(reachable), len(m.live))
	}
	for d := range m.live {
		if !reachable[d] || d.refs != refs[d] || d.marked {
			t.Fatal("DAG reference/mark invariant", d.refs, refs[d], d.marked)
		}
	}
}
func TestAssumptionDAGIndependentPathsAndReachability(t *testing.T) {
	m := assumptionFixture(t)
	rng := rand.New(rand.NewPCG(68291, 37907))
	values := []Assumption{AttachedAssumption(&AssumptionAttachment{}), AttachedAssumption(&AssumptionAttachment{})}
	lits := []sat.Literal{3, 3}
	values = append(values, literalAssumption(&lits[0]), literalAssumption(&lits[1]))
	for trial := 0; trial < 80; trial++ {
		model := map[*AssumptionSet]assumptionModelNode{}
		var held []*AssumptionSet
		for step := 0; step < 24; step++ {
			var d *AssumptionSet
			if len(held) < 2 || rng.IntN(3) == 0 {
				v := values[rng.IntN(len(values))]
				d = assumptionLeaf(t, m, v)
				model[d] = assumptionModelNode{value: v, leaf: true}
			} else {
				a, b := held[rng.IntN(len(held))], held[rng.IntN(len(held))]
				d = assumptionJoin(t, m, a, b)
				if d != a && d != b {
					model[d] = assumptionModelNode{children: [2]*AssumptionSet{a, b}}
				}
			}
			held = append(held, d)
			var got []Assumption
			assignmentMust(t, m.linearize(d, &got))
			want := assumptionModelLeaves(d, model)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("independent path order", trial, step, got, want)
			}
			assumptionModelOwners(t, m, held, model)
			if len(held) > 3 && rng.IntN(3) == 0 {
				idx := rng.IntN(len(held))
				assignmentMust(t, m.decRef(held[idx]))
				held = append(held[:idx], held[idx+1:]...)
				assumptionModelOwners(t, m, held, model)
			}
		}
		for len(held) > 0 {
			idx := rng.IntN(len(held))
			assignmentMust(t, m.decRef(held[idx]))
			held = append(held[:idx], held[idx+1:]...)
			assumptionModelOwners(t, m, held, model)
		}
	}
}

func TestAssumptionAllocationAndTraversalRecovery(t *testing.T) {
	for _, phase := range []string{"leaf", "join", "visit", "delete"} {
		for _, stop := range []int64{0, 1, 2, 3, 4, 8, 16, 32, -1} {
			t.Run(fmt.Sprintf("%s/%d", phase, stop), func(t *testing.T) {
				m := assumptionFixture(t)
				a := assumptionLeaf(t, m, AttachedAssumption(&AssumptionAttachment{}))
				b := assumptionLeaf(t, m, AttachedAssumption(&AssumptionAttachment{}))
				root := assumptionJoin(t, m, a, b)
				if phase == "delete" {
					assignmentMust(t, m.decRef(a))
					assignmentMust(t, m.decRef(b))
				}
				m.budget.FailAfter(stop)
				var made *AssumptionSet
				var err error
				switch phase {
				case "leaf":
					made, err = m.mkLeaf(a.value)
				case "join":
					made, err = m.mkJoin(a, b)
				case "visit":
					var out []Assumption
					err = m.linearize(root, &out)
				case "delete":
					err = m.decRef(root)
				}
				m.budget.FailAfter(-1)
				assignmentMust(t, m.cleanup())
				var physical umemory.Failure
				if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
					t.Fatal("unexpected refusal", err)
				}
				if made != nil {
					assignmentMust(t, m.incRef(made))
					assignmentMust(t, m.decRef(made))
				}
				if phase != "delete" {
					assignmentMust(t, m.decRef(root))
					assignmentMust(t, m.decRef(a))
					assignmentMust(t, m.decRef(b))
				}
				if len(m.live) != 0 {
					t.Fatal("retained allocation after recovery", len(m.live))
				}
			})
		}
	}
}

func assumptionComb(t *testing.T, m *assumptionManager, n int) (*AssumptionSet, []Assumption) {
	t.Helper()
	value := AttachedAssumption(&AssumptionAttachment{})
	root := assumptionLeaf(t, m, value)
	values := []Assumption{value}
	for i := 0; i < n; i++ {
		v := AttachedAssumption(&AssumptionAttachment{})
		leaf := assumptionLeaf(t, m, v)
		next := assumptionJoin(t, m, leaf, root)
		assignmentMust(t, m.decRef(leaf))
		assignmentMust(t, m.decRef(root))
		root = next
		values = append([]Assumption{v}, values...)
	}
	return root, values
}
func TestAssumptionTraversalReachedPrefixAndDeletionPushContinuation(t *testing.T) {
	for _, phase := range []string{"visit", "delete"} {
		for stop := int64(0); stop < 4; stop++ {
			t.Run(fmt.Sprintf("%s/%d", phase, stop), func(t *testing.T) {
				m := assumptionFixture(t)
				root, want := assumptionComb(t, m, 18)
				prefix := AttachedAssumption(&AssumptionAttachment{})
				out := []Assumption{prefix}
				m.budget.FailAfter(stop)
				var err error
				if phase == "visit" {
					err = m.linearize(root, &out)
				} else {
					err = m.decRef(root)
				}
				m.budget.FailAfter(-1)
				if err != nil {
					if m.mode == 0 {
						t.Fatal("interrupted operation not retained")
					}
					if phase == "visit" {
						if !root.marked || out[0] != prefix {
							t.Fatal("source traversal prefix erased")
						}
						if len(out) > 1 && !reflect.DeepEqual(out[1:], want[:len(out)-1]) {
							t.Fatal("wrong reached output prefix")
						}
						if _, e := m.contains(root, want[0]); !errors.Is(e, ErrAssumptionPending) {
							t.Fatal("reentered interrupted traversal", e)
						}
					} else if m.childToPush != nil && m.childToPush.refs != 0 {
						t.Fatal("pending child decrement was not consumed")
					}
				}
				assignmentMust(t, m.cleanup())
				if phase == "visit" {
					out = nil
					assignmentMust(t, m.linearize(root, &out))
					if !reflect.DeepEqual(out, want) {
						t.Fatal("recovered traversal", out, want)
					}
					assignmentMust(t, m.decRef(root))
				}
				if len(m.live) != 0 || m.current != nil || m.childToPush != nil || m.deleteRoot != nil {
					t.Fatal("retained deletion work after recovery")
				}
			})
		}
	}
}
