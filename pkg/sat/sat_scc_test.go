// Source: src/sat/sat_scc.cpp and src/sat/sat_elim_eqs.cpp.
package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func satSCCFixture(t *testing.T, n uint32) (*satSCC, *satSolverClauseLifecycle) {
	t.Helper()
	l, p, _, _ := satLifecycleFixture(t, n)
	local, e := p.config.context.NewValue()
	satOK(t, e)
	c, e := newSATSCC(l, &local)
	satOK(t, e)
	satOK(t, p.config.context.Del(&local))
	t.Cleanup(func() { satOK(t, c.Close()) })
	return c, l
}
func TestSATSCCDefaultEquivalenceAndActualConverter(t *testing.T) {
	c, l := satSCCFixture(t, 3)
	p := l.propagation
	satOK(t, l.mkBinClause(1, 2, false))
	satOK(t, l.mkBinClause(3, 0, false))
	n, e := c.run()
	satOK(t, e)
	if n != 1 || c.numElim != 1 {
		t.Fatal(n, c.numElim)
	}
	v, e := p.variables.wasEliminated(1)
	satOK(t, e)
	if !v {
		t.Fatal("actual eliminated column")
	}
	if p.converter.Size() != 1 {
		t.Fatal("actual converter entry", p.converter.Size())
	}
	r, e := p.converter.EntryAt(0)
	satOK(t, e)
	k, vv, e := p.converter.KindAndVar(r)
	satOK(t, e)
	if k != satModelElimVar || vv != 1 {
		t.Fatal(k, vv)
	}
	n, e = c.run()
	satOK(t, e)
	if n != 0 || p.converter.Size() != 1 {
		t.Fatal(n, p.converter.Size())
	}
}
func TestSATSCCExternalRootAndContradiction(t *testing.T) {
	for _, ext := range []bool{false, true} {
		c, l := satSCCFixture(t, 2)
		p := l.propagation
		if ext {
			satOK(t, p.variables.external.Write(1, 1))
		}
		satOK(t, l.mkBinClause(1, 2, false))
		satOK(t, l.mkBinClause(3, 0, false))
		n, e := c.run()
		satOK(t, e)
		if n != 1 {
			t.Fatal(n)
		}
		root := BoolVar(0)
		eliminated := BoolVar(1)
		if ext {
			root, eliminated = 1, 0
		}
		yes, e := p.variables.wasEliminated(eliminated)
		satOK(t, e)
		no, e := p.variables.wasEliminated(root)
		satOK(t, e)
		if !yes || no {
			t.Fatal(ext, yes, no)
		}
	}
	c, l := satSCCFixture(t, 1)
	satOK(t, l.mkBinClause(1, 1, false))
	satOK(t, l.mkBinClause(0, 0, false))
	n, e := c.run()
	satOK(t, e)
	if n != 0 || !l.propagation.inconsistent || l.propagation.converter.Size() != 0 {
		t.Fatal(n, l.propagation.inconsistent)
	}
}
func TestSATElimEqsAllClauseAritiesAndModels(t *testing.T) {
	for _, profile := range [3][2]bool{{false, false}, {true, false}, {true, true}} {
		learned, frozen := profile[0], profile[1]
		l, p, _, _ := satLifecycleFixture(t, 5)
		roots := []Literal{0, 0, 4, 6, 8}
		for _, ls := range [][]Literal{{0, 2, 4}, {0, 2, 4, 6}, {0, 2, 4, 6, 8}, {0, 3, 4}} {
			r, e := p.clauses.MkClause(ls, learned)
			satOK(t, e)
			if frozen {
				satOK(t, p.clauses.Freeze(r))
			} else {
				satOK(t, l.attachClause(r))
			}
			satOK(t, l.pushClause(learned, r))
		}
		e, er := newSATElimEqs(l)
		satOK(t, er)
		t.Cleanup(func() { satOK(t, e.Close()) })
		satOK(t, e.run(roots, []BoolVar{1}))
		vec := l.Original
		if learned {
			vec = l.learned
		}
		if len(vec) != 2 {
			t.Fatal("retain only arity3/4", len(vec))
		}
		for i, r := range vec {
			n, er := p.clauses.Size(r)
			satOK(t, er)
			if n != uint32(i+3) {
				t.Fatal(n)
			}
			for j := uint32(0); j < n; j++ {
				lit, er := p.clauses.Literal(r, j)
				satOK(t, er)
				want := Literal(0)
				if j > 0 {
					want = Literal(2 * (j + 1))
				}
				if lit != want {
					t.Fatal(lit, j)
				}
			}
		}
		if p.converter.Size() != 1 {
			t.Fatal(p.converter.Size())
		}
	}
}

func TestSATSCCIndependentBoundedImplicationGraphs(t *testing.T) {
	edges := [6][2]uint32{{0, 1}, {0, 2}, {1, 0}, {1, 2}, {2, 0}, {2, 1}}
	for bits := uint32(0); bits < 64; bits++ {
		c, l := satSCCFixture(t, 3)
		var reach [3][3]bool
		for i := 0; i < 3; i++ {
			reach[i][i] = true
		}
		for j, edge := range edges {
			if bits&(1<<uint(j)) != 0 {
				reach[edge[0]][edge[1]] = true
				satOK(t, l.mkBinClause(Literal(edge[0]*2+1), Literal(edge[1]*2), false))
			}
		}
		for k := 0; k < 3; k++ {
			for i := 0; i < 3; i++ {
				for j := 0; j < 3; j++ {
					reach[i][j] = reach[i][j] || (reach[i][k] && reach[k][j])
				}
			}
		}
		components := uint32(0)
		for i := 0; i < 3; i++ {
			first := true
			for j := 0; j < i; j++ {
				if reach[i][j] && reach[j][i] {
					first = false
				}
			}
			if first {
				components++
			}
		}
		n, e := c.run()
		satOK(t, e)
		if n != 3-components {
			t.Fatal(bits, n, components)
		}
		for mask := uint32(0); mask < 8; mask++ {
			model := Model{lbool.False, lbool.False, lbool.False}
			for i := uint32(0); i < 3; i++ {
				if mask&(1<<i) != 0 {
					model[i] = lbool.True
				}
			}
			for i := uint32(0); i < 3; i++ {
				el, er := l.propagation.variables.wasEliminated(BoolVar(i))
				satOK(t, er)
				if el {
					model[i] = lbool.Undef
				}
			}
			satOK(t, l.propagation.converter.Apply(model))
			for j, edge := range edges {
				if bits&(1<<uint(j)) != 0 && reach[edge[0]][edge[1]] && reach[edge[1]][edge[0]] && model[edge[0]] != model[edge[1]] {
					t.Fatal("converter failed equivalence", bits, mask, edge, model)
				}
			}
		}
	}
}
func TestSATElimEqsEmptyUnitUnaffectedAndExternal(t *testing.T) {
	for _, empty := range []bool{false, true} {
		l, p, _, _ := satLifecycleFixture(t, 3)
		e, err := newSATElimEqs(l)
		satOK(t, err)
		t.Cleanup(func() { satOK(t, e.Close()) })
		roots := []Literal{0, 0, 0}
		r, err := p.clauses.MkClause([]Literal{0, 2, 4}, true)
		satOK(t, err)
		satOK(t, p.clauses.Freeze(r))
		satOK(t, l.pushClause(true, r))
		if empty {
			satOK(t, p.assign(1, satJustification{}))
		}
		satOK(t, e.run(roots, nil))
		if empty {
			if !p.inconsistent || len(l.learned) != 1 || !p.clauses.IsLive(r) {
				t.Fatal("empty retains current owner and source oldsize")
			}
			n, err := p.clauses.Size(r)
			satOK(t, err)
			if n != 3 {
				t.Fatal(n)
			}
		} else {
			v, err := p.variables.valueLiteral(0)
			satOK(t, err)
			if v != lbool.True || len(l.learned) != 0 || p.clauses.IsLive(r) {
				t.Fatal("unit assign then delete", v, len(l.learned))
			}
		}
	}
	l, p, _, _ := satLifecycleFixture(t, 3)
	e, err := newSATElimEqs(l)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, e.Close()) })
	r, err := p.clauses.MkClause([]Literal{4, 0, 2}, false)
	satOK(t, err)
	satOK(t, l.pushClause(false, r))
	satOK(t, e.run([]Literal{0, 2, 4}, nil))
	for i, want := range []Literal{4, 0, 2} {
		got, err := p.clauses.Literal(r, uint32(i))
		satOK(t, err)
		if got != want {
			t.Fatal("unaffected skips sort", got, want)
		}
	}
	satOK(t, p.variables.external.Write(1, 1))
	satOK(t, e.run([]Literal{0, 0, 4}, []BoolVar{1}))
	elim, err := p.variables.wasEliminated(1)
	satOK(t, err)
	if elim || p.converter.Size() != 0 {
		t.Fatal("external adds real clauses instead of eliminated entry")
	}
}
func TestSATSCCParametersDefaultGlobalsAndFreshLocalStorage(t *testing.T) {
	c, l := satSCCFixture(t, 2)
	p := l.propagation
	ctx := p.config.context
	if !c.enabled {
		t.Fatal("source default true")
	}
	local, err := ctx.NewValue()
	satOK(t, err)
	t.Cleanup(func() { satOK(t, ctx.Del(&local)) })
	p.config.globals.Register = func(r util.GlobalParamsRegistrar) error {
		d, err := util.NewParamDescrs(p.config.symbols, util.ParamDescrsConfig{Budget: p.config.globals.Budget})
		if err != nil {
			return err
		}
		if err = collectSCCParamDescrs(d); err != nil {
			return err
		}
		return r.RegisterModule("sat", d)
	}
	satOK(t, p.config.globals.SetName("sat.scc", "false"))
	satOK(t, c.Update(&local))
	if c.enabled {
		t.Fatal("real module fallback")
	}
	key := "scc"
	satOK(t, ctx.SetBoolName(&local, &key, true))
	satOK(t, c.Update(&local))
	if !c.enabled {
		t.Fatal("local precedes module")
	}
	satOK(t, p.config.globals.SetName("sat.scc", "true"))
	before := satSCCAllocationCount(t, c)
	n, err := c.run()
	satOK(t, err)
	if n != 0 || satSCCAllocationCount(t, c) <= before {
		t.Fatal("fresh source vectors", n, before, p.memory.State())
	}
	after := p.memory.Stats()
	used := l.budget.Used()
	before = satSCCAllocationCount(t, c)
	n, err = c.run()
	satOK(t, err)
	if n != 0 || satSCCAllocationCount(t, c) <= before || p.memory.Stats() != after || l.budget.Used() != used {
		t.Fatal("fresh locals then exact retained plateau", n, p.memory.State(), after, p.memory.Stats(), used, l.budget.Used())
	}
	c.resetStatistics()
	if c.numElim != 0 {
		t.Fatal(c.numElim)
	}
}

func satSCCAllocationCount(t *testing.T, c *satSCC) uint64 {
	t.Helper()
	n, e := c.lifecycle.propagation.memory.AllocationCount()
	satOK(t, e)
	return n
}

func satSCCAudit(t *testing.T, c *satSCC, extra ...StoredClauseRef) {
	t.Helper()
	l := c.lifecycle
	p := l.propagation
	v := p.variables
	a := p.clauses
	refs := append([]StoredClauseRef(nil), extra...)
	addVector := func(x []StoredClauseRef, learned bool) {
		e := &c.eliminator
		if e.interrupted && e.learned == learned {
			refs = append(refs, x[e.drainPrefix:e.out]...)
			start := e.cursor
			if e.drainSuffix > start {
				start = e.drainSuffix
			}
			refs = append(refs, x[start:e.end]...)
		} else {
			refs = append(refs, x...)
		}
	}
	addVector(l.Original, false)
	addVector(l.learned, true)
	if l.pendingCandidate != (StoredClauseRef{}) {
		refs = append(refs, l.pendingCandidate)
	}
	for i, r := range refs {
		if !a.IsLive(r) {
			t.Fatal("named source owner stale", r)
		}
		for j := 0; j < i; j++ {
			if refs[j] == r {
				t.Fatal("duplicate scalar owner", r)
			}
		}
	}
	small := a.storage.Stats()
	if small.Live != uint32(len(refs)) {
		t.Fatal("unnamed physical clause", small.Live, len(refs))
	}
	want := c.header + c.eliminator.header + l.ownerBytes + uint64(cap(l.Original)+cap(l.learned))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes + uint64(cap(todo.inSet)) + 4*uint64(cap(todo.values))
	want += a.ownerBytes + 4*uint64(cap(a.ids.Free)) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes + mc.entryBytes + mc.pendingBytes + mc.store.ownerBytes + mc.store.directoryBytes + satWatchGraphBytes(mc.store.words.Stats())
	r := p.limit
	want += uint64(reflect.TypeOf(util.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*util.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
	if want != l.budget.Used() {
		t.Fatal("whole named once-only physical graph", want, l.budget.Used())
	}
	var raw []util.Block
	for _, b := range [7]*satSCCVector{&c.index, &c.lowlink, &c.stack, &c.inStack, &c.roots, &c.frames, &c.toElim} {
		if b.raw != (util.Block{}) {
			raw = append(raw, b.raw)
		}
		if b.pending != (util.Block{}) {
			raw = append(raw, b.pending)
		}
	}
	satScalarAuditRoots(t, v, raw...)
}

func TestSATSCCActualAllocationPrefixSweep(t *testing.T) {
	failures := 0
	success := false
	var successOrdinal int64
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		c, l := satSCCFixture(t, 5)
		p := l.propagation
		satOK(t, l.mkBinClause(1, 2, false))
		satOK(t, l.mkBinClause(3, 0, false))
		satOK(t, l.mkBinClause(3, 4, false))
		satOK(t, l.mkBinClause(5, 2, false))
		for _, ls := range [][]Literal{{0, 2, 6}, {0, 2, 6, 8}, {1, 3, 6, 8}} {
			_, err := l.MkClauseCore(uint32(len(ls)), ls, false)
			satOK(t, err)
		}
		l.budget.FailAfter(ordinal)
		n, err := c.run()
		satSCCAudit(t, c)
		if err != nil {
			if !errors.Is(err, arena.ErrInjectedFailure) {
				var mf util.Failure
				if !errors.As(err, &mf) || mf.Kind != util.FailureMemout || !p.memory.State().OutOfMemory {
					t.Fatal(ordinal, err)
				}
			}
			failures++
		} else {
			if n != 2 {
				t.Fatal(ordinal, n)
			}
			success = true
			successOrdinal = ordinal
		}
		l.budget.FailAfter(-1)
		satOK(t, c.Close())
		satLifecycleAssertOwned(t, l)
		satOK(t, l.Close())
		if success {
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal(failures, success)
	}
	t.Logf("actual refusal ordinals0..%d, first mathematical success %d", failures-1, successOrdinal)
}

func TestSATSCCCancelBeforeNodeAndCapturedPhysicalFrame(t *testing.T) {
	c, l := satSCCFixture(t, 2)
	p := l.propagation
	satOK(t, l.mkBinClause(1, 2, false))
	satOK(t, l.mkBinClause(3, 0, false))
	p.limit.SetCancel(1)
	before := p.limit.Count()
	n, err := c.run()
	if !errors.Is(err, ErrSATSolverCanceled) || n != 0 || c.nextIndex != 0 || p.converter.Size() != 0 || p.limit.Count() != before+1 {
		t.Fatal(n, err, c.nextIndex, p.limit.Count())
	}
	satSCCAudit(t, c)
	p.limit.ResetCancel()
	// A source frame holds raw watch pointers, not a movable watch-list reference.
	c.active = true
	satOK(t, c.index.resize(p.memory, 4, ^uint32(0)))
	satOK(t, c.lowlink.resize(p.memory, 4, ^uint32(0)))
	satOK(t, c.inStack.resize(p.memory, 4, 0))
	satOK(t, c.newNode(0))
	_, _, it, err := c.frame(0)
	satOK(t, err)
	list, err := p.variables.watches.Get(0)
	satOK(t, err)
	satOK(t, list.Finalize())
	_, err = satPropagationCursorRead(&it, 0)
	if !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("captured physical backing cannot follow replaced list", err)
	}
	satSCCAudit(t, c)
	satOK(t, c.closeVectors())
	if c.report.Running() {
		t.Fatal("report completed only after locals")
	}
}
func TestSATSCCColdConstructionAndSourceStatWrap(t *testing.T) {
	l, p, _, b := satLifecycleFixture(t, 2)
	local, err := p.config.context.NewValue()
	satOK(t, err)
	t.Cleanup(func() { satOK(t, p.config.context.Del(&local)) })
	used := b.Used()
	b.FailAfter(0)
	c, err := newSATSCC(l, &local)
	if c != nil || !errors.Is(err, arena.ErrInjectedFailure) || b.Used() != used {
		t.Fatal(c, err, b.Used(), used)
	}
	b.FailAfter(-1)
	satLifecycleAssertOwned(t, l)
	c, err = newSATSCC(l, &local)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, l.mkBinClause(1, 2, false))
	satOK(t, l.mkBinClause(3, 0, false))
	c.numElim = ^uint32(0)
	n, err := c.run()
	satOK(t, err)
	if n != 1 || c.numElim != 0 {
		t.Fatal(n, c.numElim)
	}
	copy := *c
	if _, err = copy.run(); !errors.Is(err, ErrSATOwner) {
		t.Fatal(err)
	}
	satOK(t, c.Close())
	if _, err = c.run(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
}
func TestSATElimEqsBinaryUnitConflictKeepsSourcePrefix(t *testing.T) {
	l, p, _, _ := satLifecycleFixture(t, 2)
	e, err := newSATElimEqs(l)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, e.Close()) })
	satOK(t, l.mkBinClause(0, 2, false))
	satOK(t, p.assign(1, satJustification{}))
	// Cleanup reaches conflicting assign before source wlist.set_end. No invented
	// early return in operator(): original clauses are still visited as source.
	list, err := p.variables.watches.Get(1)
	satOK(t, err)
	before, err := list.Size()
	satOK(t, err)
	satOK(t, e.run([]Literal{0, 0}, nil))
	after, err := list.Size()
	satOK(t, err)
	if !p.inconsistent || before != after || p.converter.Size() != 0 {
		t.Fatal("unit conflict before set_end", before, after, p.inconsistent)
	}
}

func TestSATSCCVectorSourceGeometryAndConstructorPublication(t *testing.T) {
	c, l := satSCCFixture(t, 1)
	r := l.propagation.memory
	c.active = true
	for _, width := range []uint32{1, 4, 36} {
		v := satSCCVector{width: width}
		satOK(t, v.grow(r))
		cap, err := v.cap(r)
		satOK(t, err)
		extent, err := r.Size(v.raw)
		satOK(t, err)
		if cap != 2 || extent != uint64(8+2*width) {
			t.Fatal(width, cap, extent)
		}
		satOK(t, r.WriteUint32(v.raw, 0, ^uint32(0)))
		before := satSCCAllocationCount(t, c)
		err = v.grow(r)
		var ex z3exception.DefaultException
		if !errors.As(err, &ex) || ex.Message() != "Overflow encountered when expanding vector" || satSCCAllocationCount(t, c) != before {
			t.Fatal(width, err)
		}
		satOK(t, r.WriteUint32(v.raw, 0, 2))
		satOK(t, v.close(r))
	}
	// Explicit malformed retained-frame backing: a constructor is not a source
	// published element until all captured iterator fields have been initialized.
	satOK(t, c.frames.grow(r))
	satOK(t, r.Reallocate(&c.frames.raw, 16))
	satOK(t, c.index.resize(r, 2, ^uint32(0)))
	satOK(t, c.lowlink.resize(r, 2, ^uint32(0)))
	satOK(t, c.inStack.resize(r, 2, 0))
	err := c.newNode(0)
	if !errors.Is(err, arena.ErrBounds) {
		t.Fatal(err)
	}
	n, err := c.frames.size(r)
	satOK(t, err)
	if n != 0 {
		t.Fatal("incomplete frame constructor published source size", n)
	}
	satSCCAudit(t, c)
	satOK(t, c.closeVectors())
}

func TestSATSCCVectorDebugCapacityAssertionBeforeOverflow(t *testing.T) {
	c, l := satSCCFixture(t, 1)
	r := l.propagation.memory
	c.active = true
	satOK(t, c.index.grow(r))
	satOK(t, r.WriteUint32(c.index.raw, 0, 0))
	before := satSCCAllocationCount(t, c)
	err := c.index.grow(r)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatal("source capacity SASSERT before growth", err)
		}
	} else {
		var ex z3exception.DefaultException
		if !errors.As(err, &ex) || ex.Message() != "Overflow encountered when expanding vector" {
			t.Fatal(err)
		}
	}
	if satSCCAllocationCount(t, c) != before {
		t.Fatal("no allocation before source assertion/overflow")
	}
	satSCCAudit(t, c)
	satOK(t, r.WriteUint32(c.index.raw, 0, 2))
	satOK(t, c.closeVectors())
}

func TestSATSCCResizeWholeReservationBeforeSizeAndShrink(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		c, l := satSCCFixture(t, 1)
		c.active = true
		r := l.propagation.memory
		l.budget.FailAfter(ordinal)
		err := c.index.resize(r, 9, 77)
		n, e := c.index.size(r)
		satOK(t, e)
		capacity, e := c.index.cap(r)
		satOK(t, e)
		satSCCAudit(t, c)
		if err != nil && capacity >= 2 {
			if n != 0 {
				t.Fatalf("later growth refusal published size %d, source retains 0 (capacity %d)", n, capacity)
			}
			reached = true
		}
		l.budget.FailAfter(-1)
		satOK(t, c.closeVectors())
		if reached {
			break
		}
	}
	if !reached {
		t.Fatal("no actual later growth refusal")
	}
	c, l := satSCCFixture(t, 1)
	c.active = true
	r := l.propagation.memory
	satOK(t, c.index.resize(r, 5, 17))
	satOK(t, c.index.resize(r, 2, 99))
	n, e := c.index.size(r)
	satOK(t, e)
	if n != 2 {
		t.Fatal("source shrink", n)
	}
	satOK(t, c.index.resize(r, 2, 99))
	x, e := c.index.get(r, 1, 0)
	satOK(t, e)
	if x != 17 {
		t.Fatal("equal resize must not construct", x)
	}
	satOK(t, c.index.resize(r, 4, 31))
	for i := uint32(0); i < 4; i++ {
		x, e := c.index.get(r, i, 0)
		satOK(t, e)
		want := uint32(17)
		if i >= 2 {
			want = 31
		}
		if x != want {
			t.Fatal(i, x, want)
		}
	}
	satSCCAudit(t, c)
	satOK(t, c.closeVectors())
}

func TestSATElimEqsOwningIdentityGuard(t *testing.T) {
	_, l := satSCCFixture(t, 1)
	e, err := newSATElimEqs(l)
	satOK(t, err)
	copied := *e
	before := l.budget.Used()
	if err = copied.Close(); !errors.Is(err, ErrSATOwner) {
		t.Fatal("copied owning helper", err)
	}
	if l.budget.Used() != before {
		t.Fatal("copied Close refunded owner")
	}
	var zero satElimEqs
	if err = zero.Close(); !errors.Is(err, ErrSATOwner) {
		t.Fatal("zero owning helper", err)
	}
	satOK(t, e.Close())
	satOK(t, e.Close())
}

func TestSATSCCResizePublishesBeforeReachedConstructorGuard(t *testing.T) {
	c, l := satSCCFixture(t, 1)
	c.active = true
	r := l.propagation.memory
	satOK(t, c.index.grow(r))
	// Explicit malformed capacity extent: the existing physical block has two
	// scalar slots. Source size publication precedes the endangered third write.
	satOK(t, r.WriteUint32(c.index.raw, 0, 3))
	err := c.index.resize(r, 3, 43)
	if !errors.Is(err, arena.ErrBounds) {
		t.Fatal(err)
	}
	n, e := c.index.size(r)
	satOK(t, e)
	if n != 3 {
		t.Fatal("source size before ascending constructors", n)
	}
	for i := uint32(0); i < 2; i++ {
		x, e := c.index.get(r, i, 0)
		satOK(t, e)
		if x != 43 {
			t.Fatal(i, x)
		}
	}
	satSCCAudit(t, c)
	// No pointer/counter recovery; source scalar destructor frees actual backing.
	satOK(t, c.closeVectors())
}

func TestSATElimEqsSortActualLargeBacking(t *testing.T) {
	c, l := satSCCFixture(t, 16)
	a := l.propagation.clauses
	// Test-control input is deliberately raw, not a pre-simplified solver clause.
	// Descending repeated literals includes adjacent complementary polarities.
	input := make([]Literal, 512)
	for i := range input {
		input[i] = Literal(31 - i/16)
	}
	clause := satMk(t, a, input, true)
	satOK(t, c.eliminator.sortClause(clause, 512))
	for i := uint32(0); i < 512; i++ {
		x, e := a.Literal(clause, i)
		satOK(t, e)
		if x != Literal(i/16) {
			t.Fatal(i, x)
		}
	}
	satSCCAudit(t, c, clause)
	satOK(t, a.Delete(clause))
}
