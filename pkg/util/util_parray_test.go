// Licensed under the MIT license.
package util

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func pArrayNew(t *testing.T, preserve bool) (*Int32PArrayManager, *arena.SmallBytes, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 28)
	a, e := arena.NewSmallBytes(arena.SmallBytesConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewInt32PArrayManager(Int32PArrayConfig{Budget: b, Allocator: a, PreserveRoots: preserve})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
		a.Close()
		if b.Used() != 0 {
			t.Error("leak", b.Used())
		}
	})
	return m, a, b
}
func pArrayOK(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func pArraySize(t *testing.T, m *Int32PArrayManager, r Int32PArrayRef) uint32 {
	t.Helper()
	n, e := m.Size(r)
	pArrayOK(t, e)
	return n
}
func pArrayGet(t *testing.T, m *Int32PArrayManager, r *Int32PArrayRef, i uint32) int32 {
	t.Helper()
	v, e := m.Get(r, i)
	pArrayOK(t, e)
	return v
}
func pArrayGraph(t *testing.T, m *Int32PArrayManager, a *arena.SmallBytes, b *arena.Budget, refs ...Int32PArrayRef) {
	t.Helper()
	counts := map[arena.SmallBytesRef]uint32{}
	cells := map[arena.SmallBytesRef]pArrayCell{}
	blocks := map[arena.SmallBytesRef]bool{}
	var todo []arena.SmallBytesRef
	for _, r := range refs {
		if !PArrayNull(r.cell) {
			counts[r.cell]++
			todo = append(todo, r.cell)
		}
	}
	for len(todo) > 0 {
		r := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if _, ok := cells[r]; ok {
			continue
		}
		c, e := m.cell(r)
		pArrayOK(t, e)
		cells[r] = c
		if c.kind == PArrayRoot {
			if !PArrayNull(c.link) {
				if blocks[c.link] {
					t.Fatal("shared value-block owner")
				}
				blocks[c.link] = true
			}
		} else {
			counts[c.link]++
			todo = append(todo, c.link)
		}
	}
	for r, c := range cells {
		if c.rc != counts[r] {
			t.Fatal("cell roots", r, c.rc, counts[r])
		}
	}
	s := m.Stats()
	as := a.Stats()
	if s.LiveCells != uint32(len(cells)) || s.LiveValueBlocks != uint32(len(blocks)) || as.Live != uint32(len(cells)+len(blocks)) || as.RequestedBytes != s.CellBytes+s.ValueBytes {
		t.Fatal("unreachable allocation before cleanup", s, as)
	}
	if b.Used() != s.OwnerBytes+s.ScratchBytes+as.OwnerBytes+as.PayloadBytes+as.MetadataBytes+as.DirectoryBytes {
		t.Fatal("complete graph charge", b.Used(), s, as)
	}
}
func TestPArraySource712Assertions(t *testing.T) {
	total := 0
	ensure := func(ok bool) {
		t.Helper()
		total++
		if !ok {
			t.Fatalf("source ENSURE %d", total)
		}
	}
	for _, preserve := range []bool{true, false} {
		m, _, _ := pArrayNew(t, preserve)
		var a1, a2, a3 Int32PArrayRef
		pArrayOK(t, m.Mk(&a1))
		ensure(pArraySize(t, m, a1) == 0)
		pArrayOK(t, m.PushBackFrom(&a1, 10, &a2))
		ensure(pArraySize(t, m, a1) == 0)
		ensure(pArraySize(t, m, a2) == 1)
		pArrayOK(t, m.PushBackFrom(&a1, 20, &a1))
		pArrayOK(t, m.PushBackFrom(&a1, 30, &a1))
		ensure(pArrayGet(t, m, &a1, 0) == 20)
		ensure(pArrayGet(t, m, &a1, 1) == 30)
		ensure(pArrayGet(t, m, &a2, 0) == 10)
		ensure(pArraySize(t, m, a1) == 2)
		ensure(pArraySize(t, m, a2) == 1)
		ensure(pArraySize(t, m, a3) == 0)
		pArrayOK(t, m.PushBackFrom(&a2, 100, &a3))
		ensure(pArraySize(t, m, a3) == 2)
		ensure(pArrayGet(t, m, &a3, 0) == 10)
		ensure(pArrayGet(t, m, &a3, 1) == 100)
		pArrayOK(t, m.PushBack(&a2, 50))
		ensure(pArrayGet(t, m, &a2, 0) == 10)
		ensure(pArrayGet(t, m, &a2, 1) == 50)
		ensure(pArraySize(t, m, a2) == 2)
		for _, r := range []*Int32PArrayRef{&a1, &a2, &a3} {
			pArrayOK(t, m.Del(r))
		}
		// tst2: exactly100+111 value-loop assertions.
		for i := int32(0); i < 100; i++ {
			pArrayOK(t, m.PushBack(&a1, i))
		}
		ensure(pArraySize(t, m, a1) == 100)
		pArrayOK(t, m.PushBackFrom(&a1, 100, &a2))
		for i := int32(0); i < 10; i++ {
			pArrayOK(t, m.PushBack(&a2, i+101))
		}
		ensure(pArrayGet(t, m, &a1, 0) == 0)
		for i := uint32(0); i < pArraySize(t, m, a1); i++ {
			ensure(uint32(pArrayGet(t, m, &a1, i)) == i)
		}
		for i := uint32(0); i < pArraySize(t, m, a2); i++ {
			ensure(uint32(pArrayGet(t, m, &a2, i)) == i)
		}
		pArrayOK(t, m.Unshare(&a1))
		pArrayOK(t, m.Del(&a1))
		pArrayOK(t, m.Del(&a2))
		// tst3: preserve the actual guarded getter routes, not eager reads.
		var a4 Int32PArrayRef
		for i := int32(0); i < 20; i++ {
			pArrayOK(t, m.PushBack(&a1, i))
		}
		ensure(pArraySize(t, m, a1) == 20)
		pArrayOK(t, m.SetFrom(&a1, 0, 1, &a2))
		for i := uint32(1); i < 20; i++ {
			if i == 6 {
				pArrayOK(t, m.Copy(&a2, &a3))
				pArrayOK(t, m.PopBack(&a3))
				pArrayOK(t, m.PopBack(&a3))
				pArrayOK(t, m.PushBack(&a3, 40))
			}
			pArrayOK(t, m.Set(&a2, i, int32(i+1)))
		}
		pArrayOK(t, m.PopBackFrom(&a2, &a4))
		pArrayOK(t, m.PopBack(&a4))
		pArrayOK(t, m.PushBack(&a4, 30))
		for i := uint32(0); i < 20; i++ {
			ensure(uint32(pArrayGet(t, m, &a2, i)) == i+1)
		}
		ensure(pArrayGet(t, m, &a1, 10) == 10)
		ensure(pArraySize(t, m, a1) == 20)
		ensure(pArraySize(t, m, a2) == 20)
		ensure(pArraySize(t, m, a3) == 19)
		ensure(pArraySize(t, m, a4) == 19)
		for i := uint32(0); i < 20; i++ {
			ensure(uint32(pArrayGet(t, m, &a1, i)) == i)
			ensure(uint32(pArrayGet(t, m, &a2, i)) == i+1)
			ensure(i >= 18 || uint32(pArrayGet(t, m, &a4, i)) == i+1)
			ensure(i >= 6 || uint32(pArrayGet(t, m, &a3, i)) == i+1)
			ensure(!(6 <= i && i <= 17) || uint32(pArrayGet(t, m, &a3, i)) == i)
		}
		ensure(pArrayGet(t, m, &a4, 18) == 30)
		ensure(pArrayGet(t, m, &a3, 18) == 40)
		pArrayOK(t, m.Unshare(&a1))
		pArrayOK(t, m.Reroot(&a4))
		pArrayOK(t, m.Unshare(&a2))
		for _, r := range []*Int32PArrayRef{&a1, &a2, &a3, &a4} {
			pArrayOK(t, m.Del(r))
		}
	}
	if total != 712 {
		t.Fatal(total)
	}
}
func TestPArrayIndependentVersionOracle(t *testing.T) {
	for _, preserve := range []bool{false, true} {
		m, _, _ := pArrayNew(t, preserve)
		refs := make([]Int32PArrayRef, 7)
		oracle := make([][]int32, 7)
		state := uint32(8181)
		for k := 0; k < 2000; k++ {
			state = state*1664525 + 1013904223
			s := int(state % 7)
			state = state*1664525 + 1013904223
			d := int(state % 7)
			state = state*1664525 + 1013904223
			op := state % 5
			copyO := append([]int32(nil), oracle[s]...)
			switch {
			case op == 0:
				pArrayOK(t, m.Copy(&refs[s], &refs[d]))
				oracle[d] = copyO
			case op == 1 || len(copyO) == 0:
				pArrayOK(t, m.PushBackFrom(&refs[s], int32(state), &refs[d]))
				oracle[d] = append(copyO, int32(state))
			case op == 2:
				pArrayOK(t, m.PopBackFrom(&refs[s], &refs[d]))
				oracle[d] = copyO[:len(copyO)-1]
			case op == 3:
				i := uint32(state % uint32(len(copyO)))
				pArrayOK(t, m.SetFrom(&refs[s], i, int32(state), &refs[d]))
				copyO[i] = int32(state)
				oracle[d] = copyO
			default:
				if len(copyO) > 0 {
					pArrayOK(t, m.Unshare(&refs[s]))
				}
			}
			for j := range refs {
				// Materialize before inspection: source's PUSH reroot inverse has
				// a finite off-by-one defect, frozen separately below.
				if len(oracle[j]) > 0 {
					pArrayOK(t, m.Unshare(&refs[j]))
				}
				if pArraySize(t, m, refs[j]) != uint32(len(oracle[j])) {
					t.Fatal("size", preserve, k, j, "op", op, "s/d", s, d, "got/want", pArraySize(t, m, refs[j]), len(oracle[j]))
				}
				for i, v := range oracle[j] {
					if pArrayGet(t, m, &refs[j], uint32(i)) != v {
						t.Fatal("value", k, j, i)
					}
				}
			}
		}
		for i := range refs {
			pArrayOK(t, m.Del(&refs[i]))
		}
		if m.Stats().LiveCells != 0 || m.Stats().LiveValueBlocks != 0 {
			t.Fatal(m.Stats())
		}
	}
}
func TestPArrayRootTrailOwnershipAndWidths(t *testing.T) {
	m, _, _ := pArrayNew(t, true)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 5))
	pArrayOK(t, m.Copy(&a, &b))
	pArrayOK(t, m.Set(&a, 0, 7))
	c, e := m.cell(b.cell)
	pArrayOK(t, e)
	if c.kind != PArraySet || c.elem != 5 || a.updates != 1 {
		t.Fatal(c, a)
	}
	root, e := m.cell(a.cell)
	pArrayOK(t, e)
	if root.rc != 2 {
		t.Fatal(root)
	}
	pArrayOK(t, m.Copy(&a, &a))
	if a.updates != 0 {
		t.Fatal("self Copy")
	}
	pArrayOK(t, m.SetFrom(&a, 0, 8, &a))
	if a.updates != 1 {
		t.Fatal("self From")
	}
	pArrayOK(t, m.Reroot(&b))
	if pArrayGet(t, m, &b, 0) != 5 || pArrayGet(t, m, &a, 0) != 8 {
		t.Fatal("reverse")
	}
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
	pArrayOK(t, m.Mk(&a))
	c, e = m.cell(a.cell)
	pArrayOK(t, e)
	c.rc = PArrayMaxRC
	pArrayOK(t, m.store(a.cell, c))
	pArrayOK(t, m.inc(a.cell))
	c, e = m.cell(a.cell)
	pArrayOK(t, e)
	if c.rc != 0 || m.Stats().LiveCells != 1 {
		t.Fatal(c, m.Stats())
	}
	if e = m.Del(&a); !errors.Is(e, ErrPArrayRef) {
		t.Fatal(e)
	}
	c.rc = 1
	pArrayOK(t, m.store(a.cell, c))
	pArrayOK(t, m.Del(&a))
}
func TestPArrayLifetimeCapacityAndWarmPlateau(t *testing.T) {
	m, a, b := pArrayNew(t, true)
	var r, s Int32PArrayRef
	pArrayOK(t, m.Mk(&r))
	pArrayOK(t, m.Copy(&r, &s))
	pArrayOK(t, m.Unshare(&r))
	if m.Stats().LiveValueBlocks != 1 {
		t.Fatal("zero header", m.Stats())
	}
	pArrayOK(t, m.Del(&r))
	pArrayOK(t, m.Del(&s))
	for i := int32(0); i < 6; i++ {
		pArrayOK(t, m.PushBack(&r, i))
	}
	c, e := m.cell(r.cell)
	pArrayOK(t, e)
	cap, e := m.capacity(c.link)
	pArrayOK(t, e)
	if cap != 8 {
		t.Fatal(cap)
	}
	before := b.Used()
	if e = m.Close(); !errors.Is(e, ErrPArrayLive) {
		t.Fatal(e)
	}
	if b.Used() != before || !a.IsLive(r.cell) {
		t.Fatal("failed close")
	}
	if n := testing.AllocsPerRun(100, func() {
		if e := m.Set(&r, 0, 9); e != nil {
			panic(e)
		}
		if _, e := m.Get(&r, 0); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	if b.Used() != before {
		t.Fatal("plateau")
	}
	pArrayOK(t, m.Del(&r))
	pArrayOK(t, m.Close())
	if e = m.Mk(&r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestPArrayPredictableFailurePrefixes(t *testing.T) {
	// Direct allocator makes each source allocation its own predictable failure
	// point; retained scratch/pool capacity prefixes are separately budgeted.
	for fail := int64(0); fail < 12; fail++ {
		b := arena.NewBudget(1 << 24)
		a, e := arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
		pArrayOK(t, e)
		m, e := NewInt32PArrayManager(Int32PArrayConfig{Budget: b, Allocator: a, PreserveRoots: true})
		pArrayOK(t, e)
		var r, s Int32PArrayRef
		pArrayOK(t, m.PushBack(&r, 10))
		pArrayOK(t, m.PushBack(&r, 20))
		pArrayOK(t, m.Copy(&r, &s))
		b.FailAfter(fail)
		e = m.PushBack(&r, 30)
		b.FailAfter(-1)
		// Allowed source publication prefix: shared root counter advances before
		// allocation; root/trail can already be published when payload growth fails.
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		if e != nil && r.updates != 1 {
			t.Fatal("counter prefix", fail, r.updates)
		}
		pArrayGraph(t, m, a, b, r, s)
		if pArraySize(t, m, s) != 2 || pArrayGet(t, m, &s, 0) != 10 || pArrayGet(t, m, &s, 1) != 20 {
			t.Fatal("old version", fail)
		}
		n := pArraySize(t, m, r)
		if n != 2 && n != 3 {
			t.Fatal(n)
		}
		st := m.Stats()
		as := a.Stats()
		if as.Live != st.LiveCells+st.LiveValueBlocks || as.RequestedBytes != st.CellBytes+st.ValueBytes {
			t.Fatal("unpublished leak before cleanup", fail, st, as)
		}
		pArrayOK(t, m.Del(&r))
		pArrayOK(t, m.Del(&s))
		pArrayOK(t, m.Close())
		a.Close()
		if b.Used() != 0 {
			t.Fatal("leak", fail, b.Used())
		}
	}
}
func TestPArrayInvalidAndDisplay(t *testing.T) {
	m, _, _ := pArrayNew(t, false)
	other, _, _ := pArrayNew(t, true)
	var r, z Int32PArrayRef
	s, e := m.DisplayInfo(z)
	pArrayOK(t, e)
	if s != "<null>" {
		t.Fatal(s)
	}
	pArrayOK(t, m.PushBack(&r, -3))
	if _, e = other.Size(r); !errors.Is(e, ErrPArrayRef) {
		t.Fatal(e)
	}
	if _, e = m.Get(&r, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if e = m.PopBack(&z); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	old := r
	pArrayOK(t, m.Del(&r))
	if _, e = m.Size(old); !errors.Is(e, ErrPArrayRef) {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(r, Int32PArrayRef{}) {
		t.Fatal(r)
	}
}
func TestPArraySourcePushRerootOffByOne(t *testing.T) {
	m, _, _ := pArrayNew(t, false)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 10))
	pArrayOK(t, m.Copy(&a, &b))
	pArrayOK(t, m.PushBack(&b, 20))
	pArrayOK(t, m.Reroot(&b))
	c, e := m.cell(a.cell)
	pArrayOK(t, e)
	if c.kind != PArrayPop || c.idx != 1 || pArraySize(t, m, a) != 0 || pArraySize(t, m, b) != 2 {
		t.Fatal("source inverse prefix", c)
	}
	if pArrayGet(t, m, &b, 0) != 10 || pArrayGet(t, m, &b, 1) != 20 {
		t.Fatal("new root")
	}
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
}
func TestPArrayUnaffectedRerootRoutesBothPolicies(t *testing.T) {
	for _, preserve := range []bool{true, false} {
		m, pool, budget := pArrayNew(t, preserve)
		var a, b Int32PArrayRef
		for i := int32(0); i < 20; i++ {
			pArrayOK(t, m.PushBack(&a, i))
		}
		pArrayOK(t, m.Copy(&a, &b))
		for i := uint32(0); i < 20; i++ {
			pArrayOK(t, m.Set(&b, i, int32(i+1)))
		}
		pArrayOK(t, m.Reroot(&a))
		pArrayOK(t, m.Reroot(&b))
		for i := uint32(0); i < 20; i++ {
			if pArrayGet(t, m, &a, i) != int32(i) || pArrayGet(t, m, &b, i) != int32(i+1) {
				t.Fatal("SET reroot changed versions", preserve, i)
			}
		}
		pArrayGraph(t, m, pool, budget, a, b)
		pArrayOK(t, m.Del(&a))
		pArrayOK(t, m.Del(&b))
	}
	m, pool, budget := pArrayNew(t, false)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 10))
	pArrayOK(t, m.PushBack(&a, 20))
	pArrayOK(t, m.Copy(&a, &b))
	pArrayOK(t, m.PopBack(&b))
	pArrayOK(t, m.Reroot(&b))
	if pArraySize(t, m, a) != 2 || pArraySize(t, m, b) != 1 || pArrayGet(t, m, &a, 1) != 20 || pArrayGet(t, m, &b, 0) != 10 {
		t.Fatal("POP inverse")
	}
	pArrayGraph(t, m, pool, budget, a, b)
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
}
func TestPArraySourcePushRerootDefectPreservedWithRootPolicy(t *testing.T) {
	m, _, _ := pArrayNew(t, true)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 10))
	pArrayOK(t, m.PushBack(&a, 20))
	pArrayOK(t, m.Copy(&a, &b))
	pArrayOK(t, m.PopBack(&b))
	if pArraySize(t, m, b) != 1 {
		t.Fatal("mathematical old version size before reroot should1")
	}
	pArrayOK(t, m.Reroot(&a))
	if pArraySize(t, m, b) != 0 || pArraySize(t, m, a) != 2 {
		t.Fatal("source PUSH inverse deliberately produces0 instead of mathematical1")
	}
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
}
func TestPArrayMaterializationAndRerootFailureGraphs(t *testing.T) {
	for _, operation := range []string{"unshare", "unfold", "reroot"} {
		for fail := int64(0); fail < 14; fail++ {
			budget := arena.NewBudget(1 << 24)
			pool, e := arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
			pArrayOK(t, e)
			m, e := NewInt32PArrayManager(Int32PArrayConfig{Budget: budget, Allocator: pool})
			pArrayOK(t, e)
			var a, b Int32PArrayRef
			pArrayOK(t, m.PushBack(&a, 10))
			pArrayOK(t, m.PushBack(&a, 20))
			pArrayOK(t, m.Copy(&a, &b))
			pArrayOK(t, m.PushBack(&b, 30))
			budget.FailAfter(fail)
			switch operation {
			case "unshare":
				e = m.Unshare(&b)
			case "unfold":
				e = m.Unfold(&b)
			case "reroot":
				e = m.Reroot(&b)
			}
			budget.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(operation, fail, e)
			}
			pArrayGraph(t, m, pool, budget, a, b)
			// Only successful PUSH reroot has the separately frozen source size defect.
			wantA := uint32(2)
			if e == nil && operation == "reroot" {
				wantA = 1
			}
			if pArraySize(t, m, a) != wantA || pArraySize(t, m, b) != 3 {
				t.Fatal(operation, fail, "sizeprefix", pArraySize(t, m, a), pArraySize(t, m, b))
			}
			pArrayOK(t, m.Del(&a))
			pArrayOK(t, m.Del(&b))
			pArrayOK(t, m.Close())
			pool.Close()
			if budget.Used() != 0 {
				t.Fatal("cleanup", operation, fail, budget.Used())
			}
		}
	}
}
func TestPArrayStrictThresholdsAndRetainedSlots(t *testing.T) {
	for _, count := range []int{8, 9} {
		m, _, _ := pArrayNew(t, false)
		var a, b Int32PArrayRef
		pArrayOK(t, m.PushBack(&a, 10))
		pArrayOK(t, m.PushBack(&a, 20))
		pArrayOK(t, m.Copy(&a, &b))
		for i := 0; i < count; i++ {
			pArrayOK(t, m.Set(&b, 1, int32(i)))
		}
		if pArrayGet(t, m, &b, 0) != 10 {
			t.Fatal("untouched value")
		}
		c, e := m.cell(b.cell)
		pArrayOK(t, e)
		want := uint32(PArraySet)
		if count == 9 {
			want = PArrayRoot
		}
		if c.kind != want {
			t.Fatal("strict trail >8", count, c)
		}
		pArrayOK(t, m.Del(&a))
		pArrayOK(t, m.Del(&b))
	}
	m, pool, budget := pArrayNew(t, true)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 10))
	pArrayOK(t, m.Copy(&a, &b))
	for i := 1; i <= 3; i++ {
		pArrayOK(t, m.Set(&b, 0, int32(i)))
		c, e := m.cell(b.cell)
		pArrayOK(t, e)
		if i <= 2 {
			if b.updates != uint32(i) || c.rc != 2 {
				t.Fatal("strict update >size", i, b, c)
			}
		} else if b.updates != 0 || c.rc != 1 {
			t.Fatal("unshare threshold", b, c)
		}
	}
	pArrayGraph(t, m, pool, budget, a, b)
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
	pArrayOK(t, m.PushBack(&a, 77))
	pArrayOK(t, m.PushBack(&a, 88))
	pArrayOK(t, m.PopBack(&a))
	c, e := m.cell(a.cell)
	pArrayOK(t, e)
	v, e := m.value(c.link, 1)
	pArrayOK(t, e)
	if v != 88 {
		t.Fatal("source retains popped physical slot", v)
	}
	pArrayOK(t, m.Del(&a))
}
func TestPArrayMkFailureClearsReleasedOutput(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	pool, e := arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	pArrayOK(t, e)
	m, e := NewInt32PArrayManager(Int32PArrayConfig{Budget: budget, Allocator: pool})
	pArrayOK(t, e)
	var a Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 4))
	budget.FailAfter(0)
	e = m.Mk(&a)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || a != (Int32PArrayRef{}) {
		t.Fatal("source dangling-output adaptation", a, e)
	}
	pArrayGraph(t, m, pool, budget, a)
	pArrayOK(t, m.Close())
	pool.Close()
	if budget.Used() != 0 {
		t.Fatal(budget.Used())
	}
}
func TestPArrayDisplayPrefixFaultAndBorrowedAllocationBounds(t *testing.T) {
	m, pool, budget := pArrayNew(t, false)
	var a, b Int32PArrayRef
	pArrayOK(t, m.PushBack(&a, 1))
	pArrayOK(t, m.Copy(&a, &b))
	for i := int32(0); i < 10; i++ {
		pArrayOK(t, m.Set(&b, 0, i))
	}
	full, e := m.DisplayInfo(b)
	pArrayOK(t, e)
	before := budget.Used()
	budget.FailAfter(1)
	prefix, e := m.DisplayInfo(b)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || prefix == "" || !strings.HasPrefix(full, prefix) {
		t.Fatal("defined display prefix", prefix, e)
	}
	if budget.Used() != before {
		t.Fatal("lexical format leak")
	}
	pArrayGraph(t, m, pool, budget, a, b)
	if _, e = m.allocateValues(^uint64(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	pArrayOK(t, m.Del(&a))
	pArrayOK(t, m.Del(&b))
}
func TestPArrayNilOutputChecks(t *testing.T) {
	m, _, _ := pArrayNew(t, false)
	var r Int32PArrayRef
	if e := m.SetFrom(nil, 0, 1, &r); !errors.Is(e, ErrPArrayRef) {
		t.Fatal(e)
	}
	if e := m.PopBackFrom(nil, &r); !errors.Is(e, ErrPArrayRef) {
		t.Fatal(e)
	}
}
func TestPArrayFormatHostBoundsWithoutAllocation(t *testing.T) {
	for _, limit := range []uint64{1<<31 - 1, 1<<63 - 1} {
		n, c, e := pArrayFormatCapacity(limit, limit-1, 1, limit)
		if e != nil || n != limit || c != limit {
			t.Fatal("exact host bound", limit, n, c, e)
		}
		for _, v := range [][3]uint64{{limit, limit, 1}, {limit/2 + 1, limit/2 + 1, 1}, {limit + 1, 0, 1}, {0, limit, 1}} {
			if _, _, e := pArrayFormatCapacity(v[0], v[1], v[2], limit); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("host overflow", limit, v, e)
			}
		}
	}
	n, c, e := pArrayFormatCapacity(0, 0, 64, 1<<31-1)
	if e != nil || n != 64 || c != 64 {
		t.Fatal(n, c, e)
	}
	n, c, e = pArrayFormatCapacity(64, 60, 5, 1<<31-1)
	if e != nil || n != 65 || c != 128 {
		t.Fatal(n, c, e)
	}
	if _, _, e = pArrayFormatCapacity(0, 0, 1, 63); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("minimum buffer bounds", e)
	}
}
