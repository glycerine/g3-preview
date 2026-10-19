// Source action fixtures for algebraic_numbers.cpp:360–539. These are additional
// branch tests, not claims of registered src/test/algebraic.cpp ENSURE coverage.
package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func algebraicCellSourceFixture(t *testing.T) (*AlgebraicRootCore, *UPolynomialManager) {
	return algebraicCellSourceFixtureProfile(t, false)
}

func algebraicCellSourceFixtureProfile(t *testing.T, directTLS bool) (*AlgebraicRootCore, *UPolynomialManager) {
	t.Helper()
	budget := arena.NewBudget(1 << 24)
	config := serviceConfig(budget)
	var runtime *umemory.Runtime
	var e error
	profile := umemory.SmallAllocatorProfile{PointerBits: 64}
	if directTLS {
		runtime, e = umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true})
		serviceMust(t, e)
		config.Integers.Memory = runtime
		profile.Debug = true // actual non-Windows debug allocator direct profile
		config.Integers.Profile = profile
	}
	s, e := unumeral.NewRationalService(config)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, s.Close())
		if runtime != nil {
			serviceMust(t, runtime.Close())
		}
		if budget.Used() != 0 {
			t.Fatalf("cell fixture retained %d actual charged bytes", budget.Used())
		}
	})
	p, e := NewUPolynomialManagerFromRationalService(s, UPolynomialConfig{
		Checkpoint:  func() bool { t.Fatal("cell operation visited source checkpoint"); return false },
		Cooperation: func() error { t.Fatal("cell operation visited source cooperation"); return nil },
	})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(s.Manager.BudgetField)
	serviceMust(t, e)
	f, e := NewUPolynomialFactorizationService(p, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	b, e := unumeral.NewBinaryRationalManager(s.Manager.Z, unumeral.BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	bi, e := unumeral.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	a, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: s.Manager.Z.Memory, Profile: profile})
	serviceMust(t, e)
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: s.Manager, Polynomials: p, Factors: f, Binary: b, Intervals: bi, Allocator: a,
		Checkpoint:  func() bool { t.Fatal("cell core visited source checkpoint"); return false },
		Cooperation: func() error { t.Fatal("cell core visited source cooperation"); return nil },
	})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, c.Close())
		serviceMust(t, a.Close())
		bi.Close()
		serviceMust(t, b.CloseChecked())
		serviceMust(t, f.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, p.Close())
	})
	return c, p
}

func algebraicCellSourceRecord(t *testing.T, c *AlgebraicRootCore, a *Algebraic) algebraicRecord {
	t.Helper()
	l, e := c.location(a)
	serviceMust(t, e)
	if l == nil {
		t.Fatal("expected nonzero cell")
	}
	r, e := c.load(l)
	serviceMust(t, e)
	if _, e = c.config.Allocator.ObjectExtent(l.raw); e != nil {
		t.Fatal("named cell has no physical object", e)
	}
	return r
}

// Ownership is discovered from named caller/core fields, never from the live
// descriptor registry. Registries are inspected only afterward to reject an
// unmatched actual live object. Allocator free-pool chunks remain real roots.
func algebraicCellSourcePhysical(t *testing.T, c *AlgebraicRootCore, polys []UPolynomial, integers []*unumeral.Int, raw []umemory.Block, values ...*Algebraic) {
	t.Helper()
	z, p := c.config.Rationals.Integers(), c.config.Polynomials
	refs := map[unumeral.NumericCellRef]bool{}
	addRef := func(ref unumeral.NumericCellRef) {
		if ref == 0 || z.IsFailureRef(ref) {
			return
		}
		if !unumeral.ValidNumericRef(z.Cells, ref) {
			t.Fatal("invalid named numeral", ref)
		}
		refs[ref] = true
	}
	intType := reflect.TypeOf(unumeral.Int{})
	var fields func(reflect.Value)
	fields = func(v reflect.Value) {
		if v.Type() == intType {
			addRef(unumeral.NumericCellRef(testFieldBySourceName(v, "cell").Uint()))
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				fields(v.Field(i))
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				fields(v.Index(i))
			}
		}
	}
	// These real owners contain concrete cache/local numeral fields. Pointer
	// children are borrowed and are deliberately not recursively traversed.
	for _, owner := range []reflect.Value{reflect.ValueOf(c.config.Rationals).Elem(), reflect.ValueOf(p.modular).Elem(), reflect.ValueOf(c.config.Binary).Elem(), reflect.ValueOf(c.config.Intervals).Elem(), reflect.ValueOf(c.config.Binary.Diagnostics).Elem()} {
		fields(owner)
	}
	for _, ref := range z.Tmp {
		addRef(ref)
	}
	for _, ref := range z.Arg {
		addRef(ref)
	}
	addRef(z.IntMin.Cell)
	addRef(z.Two64.Cell)
	fields(reflect.ValueOf(z.RawDisplay))
	fields(reflect.ValueOf(z.Smt2Display))
	for _, n := range integers {
		if n != nil {
			addRef((n).Cell)
		}
	}
	if c.localActive {
		fields(reflect.ValueOf(c.localQ))
	}
	fields(reflect.ValueOf(c.inversion.values))
	fields(reflect.ValueOf(c.bounds.values))
	fields(reflect.ValueOf(c.arithmetic.q))
	fields(reflect.ValueOf(c.arithmetic.b))
	fields(reflect.ValueOf(c.predicate))
	fields(reflect.ValueOf(c.isRationalTmp))
	for _, vector := range []*BinaryRationalVector{c.roots, c.lowers, c.uppers} {
		if vector != nil {
			for i := uint32(0); i < uint32(len(vector.state.values)); i++ {
				fields(reflect.ValueOf(vector.state.values[i]))
			}
		}
	}
	backings := map[uint32]bool{}
	addState := func(s *upolyVectorState) {
		if s.backing == 0 {
			return
		}
		if s.origin != p.store.origin {
			t.Fatal("cell fixture unexpected foreign coefficient store")
		}
		b := &p.store.backings[s.backing-1]
		if !b.live || b.generation != s.generation {
			t.Fatal("named coefficient owner stale")
		}
		backings[s.backing] = true
		for i := uint32(0); uint64(i) < uint64(len(b.values)); i++ {
			addRef((b.values[i]).Cell)
		}
	}
	states := []*upolyVectorState{&p.basic, &p.div1, &p.div2, &p.exactDiv, &p.gcd1, &p.gcd2, &p.cra, &p.sqf1, &p.sqf2, &p.pw, &p.db, &p.dbab1, &p.dbab2, &p.tr, &p.push}
	for i := range p.mgcd {
		states = append(states, &p.mgcd[i])
	}
	for _, state := range states {
		addState(state)
	}
	if c.compareSequence != nil {
		addState(&c.compareSequence.state.coefficients)
	}
	for _, poly := range []UPolynomial{c.tmp2, c.tmp3, c.addTmp} {
		if poly.slot != 0 {
			polys = append(polys, poly)
		}
	}
	if c.factors != nil {
		addRef((c.factors.constant).Cell)
		for i := uint32(0); i < c.factors.count; i++ {
			addState(&c.factors.factors[i])
		}
	}
	objects := map[umemory.ObjectRef]bool{}
	cellHandles := map[Algebraic]bool{}
	values = append(append([]*Algebraic(nil), values...), &c.pending)
	if c.rootSelection != nil {
		for i := uint32(0); i < uint32(len(c.rootSelection.values)); i++ {
			values = append(values, &c.rootSelection.values[i])
		}
	}
	for _, a := range values {
		if a == nil || a.slot == 0 {
			continue
		}
		if cellHandles[*a] {
			continue
		}
		cellHandles[*a] = true
		l, e := c.locationCleanup(a)
		serviceMust(t, e)
		if l.raw != (umemory.ObjectRef{}) {
			if _, e = c.config.Allocator.ObjectExtent(l.raw); e != nil {
				t.Fatal(e)
			}
			objects[l.raw] = true
		}
		if !l.initialized {
			continue
		}
		r, e := c.load(l)
		serviceMust(t, e)
		fields(reflect.ValueOf(r.value))
		fields(reflect.ValueOf(r.interval))
		if r.poly.slot != 0 {
			polys = append(polys, r.poly)
		}
	}
	for i, l := range c.locations {
		if l.live && !cellHandles[Algebraic{c.origin, uint32(i + 1), l.generation}] {
			t.Fatal("unnamed live algebraic cell", i)
		}
	}
	for _, poly := range polys {
		if poly.slot != 0 {
			d, e := p.descriptor(poly)
			serviceMust(t, e)
			addState(&d.state)
		}
	}
	for i, b := range p.store.backings {
		if b.live && !backings[uint32(i+1)] {
			t.Fatal("unnamed live coefficient backing", i)
		}
	}
	for slot := uint64(1); slot <= uint64(z.Cells.Stats().Slots); slot++ {
		cell, e := z.Cells.Read(arena.Slot(slot))
		if e != nil {
			continue
		}
		if cell.Unpublished || cell.CleanupPending {
			addRef(unumeral.MakeNumericRef(arena.Slot(slot), z.Cells.State(arena.Slot(slot)).Generation))
		}
	}
	if uint32(len(refs)) != z.Cells.Stats().Live {
		t.Fatal("unnamed live numeric descriptors", len(refs), z.Cells.Stats().Live)
	}
	for ref := range refs {
		cell := z.Cell(ref)
		if cell.Payload != (arena.VectorRef{}) {
			t.Fatal("numeric cell retained phantom Words payload")
		}
		if cell.Released {
			continue
		}
		object, e := z.CellObject(ref)
		serviceMust(t, e)
		if _, e = z.Allocator.ObjectExtent(object); e != nil {
			t.Fatal("named integer lost physical object", e)
		}
	}
	if z.PendingCell.Object != (umemory.ObjectRef{}) {
		_, e := z.Allocator.ObjectExtent(z.PendingCell.Object)
		serviceMust(t, e)
	}
	rawRoots := map[umemory.Block]bool{}
	addRaw := func(block umemory.Block) {
		if block != (umemory.Block{}) {
			rawRoots[block] = true
			if _, e := z.RuntimeOwner().Read(block, 0); e != nil && !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatal("invalid named raw block", e)
			}
		}
	}
	for _, block := range raw {
		addRaw(block)
	}

	numeric, cells := z.Allocator.Stats(), c.config.Allocator.Stats()
	want := numeric.Chunks + numeric.DirectObjects + numeric.PendingBlocks + cells.Chunks + cells.DirectObjects + cells.PendingBlocks + uint32(len(rawRoots))
	if want != z.RuntimeOwner().Stats().LiveBlocks {
		t.Fatal("unnamed actual Runtime backing", want, z.RuntimeOwner().Stats())
	}
	if cells.Chunks == 0 && uint32(len(objects)) != cells.DirectObjects {
		t.Fatal("unnamed direct algebraic object", len(objects), cells)
	}
}

func TestAlgebraicCellSourceMutableNewAndExisting(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 7, 3))
	serviceMust(t, c.config.Rationals.Integers().Mul2k(&(*q.BorrowNumerator()), 70, &(*q.BorrowNumerator())))
	var a Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) })
	before := q
	serviceMust(t, c.SetTake(&a, &q))
	r := algebraicCellSourceRecord(t, c, &a)
	if r.value != before || q != unumeral.NewRational() {
		t.Fatal("new basic source swap did not transfer actual fields", r.value, q)
	}
	algebraicCellSourcePhysical(t, c, nil, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, nil, &a)
	serviceMust(t, c.config.Rationals.SetFraction(&q, 11, 5))
	serviceMust(t, c.config.Rationals.Integers().Mul2k(&(*q.BorrowNumerator()), 70, &(*q.BorrowNumerator())))
	before = q
	serviceMust(t, c.SetTake(&a, &q))
	r = algebraicCellSourceRecord(t, c, &a)
	if q != before {
		t.Fatal("existing basic source copy consumed mutable input")
	}
	equal, e := c.config.Rationals.Equal(&r.value, &q)
	serviceMust(t, e)
	if !equal {
		t.Fatal("existing basic copy value")
	}
	if (*r.value.BorrowNumerator()).Cell == (*q.BorrowNumerator()).Cell {
		t.Fatal("existing source Copy aliased physical numeral ownership")
	}
	algebraicCellSourcePhysical(t, c, nil, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, nil, &a)
}

func TestAlgebraicCellSourceConstInputPreserved(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 13, 7))
	serviceMust(t, c.config.Rationals.Integers().Mul2k(&(*q.BorrowNumerator()), 70, &(*q.BorrowNumerator())))
	before := q
	var a Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) })
	serviceMust(t, c.SetRational(&a, &q))
	r := algebraicCellSourceRecord(t, c, &a)
	if q != before {
		t.Fatal("const input changed")
	}
	equal, e := c.config.Rationals.Equal(&r.value, &q)
	serviceMust(t, e)
	if !equal {
		t.Fatal("const basic value")
	}
	if (*r.value.BorrowNumerator()).Cell == (*q.BorrowNumerator()).Cell {
		t.Fatal("const source Copy aliased actual big numerator")
	}
	algebraicCellSourcePhysical(t, c, nil, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, nil, &a)
}

func TestAlgebraicCellSourceLeadingPositiveAndCopyFlags(t *testing.T) {
	c, p := algebraicCellSourceFixture(t)
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(4), unumeral.NewInt(0), unumeral.NewInt(-2)}))
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(2), 0)
	var a, b Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) })
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	r := algebraicCellSourceRecord(t, c, &a)
	vv, e := p.wholeView(r.poly)
	serviceMust(t, e)
	for i, want := range []int32{-2, 0, 1} {
		x, e := p.read(vv, uint32(i))
		serviceMust(t, e)
		expected := unumeral.NewInt(want)
		eq, e := p.config.Integer.Equal(x, &expected)
		serviceMust(t, e)
		if !eq {
			t.Fatalf("normalized coefficient %d want %d", i, want)
		}
	}
	if r.flags&7 != 7 {
		t.Fatalf("minimal/not-rational/negative-lower flags %x", r.flags)
	}
	// m_i is independently copied, not recomputed by the copy operation.
	l, e := c.location(&a)
	serviceMust(t, e)
	r.flags |= 23 << 3
	serviceMust(t, c.store(l, &r))
	serviceMust(t, c.Copy(&b, &a))
	copy := algebraicCellSourceRecord(t, c, &b)
	if copy.flags != r.flags || copy.poly == r.poly {
		t.Fatal("root copy lost fields or aliased owning coefficient descriptor")
	}
	serviceMust(t, c.Copy(&b, &b))
	if after := algebraicCellSourceRecord(t, c, &b); after.poly != copy.poly || after.flags != copy.flags {
		t.Fatal("source self-copy changed cell")
	}
}

func TestAlgebraicCellSourceCloseRefusesCaller(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 5, 2))
	var a Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) })
	serviceMust(t, c.SetRational(&a, &q))
	before := algebraicCellSourceRecord(t, c, &a)
	used := c.config.Rationals.Budget().Used()
	raw := c.config.Rationals.Integers().RuntimeOwner().Stats()
	if e := c.Close(); !errors.Is(e, ErrAlgebraicLive) {
		t.Fatal("Close must refuse caller-owned cell", e)
	}
	if after := algebraicCellSourceRecord(t, c, &a); after != before {
		t.Fatal("Close consumed caller cell")
	}
	if used != c.config.Rationals.Budget().Used() || raw != c.config.Rationals.Integers().RuntimeOwner().Stats() {
		t.Fatal("refused Close changed actual ownership graph")
	}
}

func TestAlgebraicCellSourceCopyDoesNotNormalize(t *testing.T) {
	c, p := algebraicCellSourceFixture(t)
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(2), 0)
	var a, b Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) })
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	r := algebraicCellSourceRecord(t, c, &a)
	// Scaling the actual stored coefficient array keeps the same root and
	// interval. Source copy copies it verbatim; it never calls normalize_coeffs.
	serviceMust(t, p.SetInts(r.poly, []unumeral.Int{unumeral.NewInt(-4), unumeral.NewInt(0), unumeral.NewInt(2)}))
	serviceMust(t, c.Copy(&b, &a))
	copy := algebraicCellSourceRecord(t, c, &b)
	v, e = p.wholeView(copy.poly)
	serviceMust(t, e)
	for i, want := range []int32{-4, 0, 2} {
		x, e := p.read(v, uint32(i))
		serviceMust(t, e)
		y := unumeral.NewInt(want)
		equal, e := p.config.Integer.Equal(x, &y)
		serviceMust(t, e)
		if !equal {
			t.Fatalf("source copy normalized coefficient %d", i)
		}
	}
}

func TestAlgebraicCellSourceDebugSignBeforeFlagPublication(t *testing.T) {
	c, p := algebraicCellSourceFixture(t)
	polynomialAssertionActionsFixture(t, "TT")
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(2), 0)
	var existing, fresh Algebraic
	t.Cleanup(func() {
		serviceMust(t, c.Del(&fresh))
		serviceMust(t, c.Del(&existing))
		serviceMust(t, c.CleanupRetained())
	})
	serviceMust(t, c.SetRoot(&existing, v, &lower, &upper, false))
	old := algebraicCellSourceRecord(t, c, &existing)
	old.flags |= 23 << 3
	loc, e := c.location(&existing)
	serviceMust(t, e)
	serviceMust(t, c.store(loc, &old))
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), unumeral.NewInt(1)}))
	v, e = p.wholeView(poly)
	serviceMust(t, e)
	e = c.SetRoot(&existing, v, &lower, &upper, true)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal("zero lower must diagnose before sign publication", e)
		}
		r := algebraicCellSourceRecord(t, c, &existing)
		if r.flags&5 != 5 || r.flags>>3 != 0 {
			t.Fatal("existing minimal/not-rational fields must publish before lower sign assertion", r.flags)
		}
		e = c.SetRoot(&fresh, v, &lower, &upper, true)
		if !errors.Is(e, ErrUPolynomialAssertion) || fresh != (Algebraic{}) || c.pending == (Algebraic{}) {
			t.Fatal("new root caller publication crossed sign assertion", fresh, c.pending, e)
		}
		r = algebraicCellSourceRecord(t, c, &c.pending)
		if r.flags&5 != 0 {
			t.Fatal("new-cell minimal fields published before lower sign assertion", r.flags)
		}
	} else {
		// Source release omits both assertion evaluations. This fixture records
		// that branch, not a mathematically valid refinable interval campaign.
		serviceMust(t, e)
		serviceMust(t, c.SetRoot(&fresh, v, &lower, &upper, true))
	}
}

func TestAlgebraicCellSourceDebugUpperReadPrecedesSignPublication(t *testing.T) {
	c, p := algebraicCellSourceFixture(t)
	polynomialAssertionActionsFixture(t, "TT")
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(-2), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(-1), 0)
	var existing, fresh Algebraic
	t.Cleanup(func() {
		serviceMust(t, c.Del(&fresh))
		serviceMust(t, c.Del(&existing))
		serviceMust(t, c.CleanupRetained())
	})
	serviceMust(t, c.SetRoot(&existing, v, &lower, &upper, false))
	if algebraicCellSourceRecord(t, c, &existing).flags&2 != 0 {
		t.Fatal("fixture lower sign must be positive")
	}
	lower, upper = unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0)
	e = c.SetRoot(&existing, v, &lower, &upper, true)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal("equal negative signs must fail upper assertion", e)
		}
		if flags := algebraicCellSourceRecord(t, c, &existing).flags; flags&7 != 5 {
			t.Fatal("upper assertion must follow minimal and precede new negative sign", flags)
		}
		e = c.SetRoot(&fresh, v, &lower, &upper, true)
		if !errors.Is(e, ErrUPolynomialAssertion) || fresh != (Algebraic{}) || c.pending == (Algebraic{}) {
			t.Fatal("new upper guard publication", e, fresh, c.pending)
		}
		if flags := algebraicCellSourceRecord(t, c, &c.pending).flags; flags != 0 {
			t.Fatal("new upper assertion preceded both minimal and sign fields", flags)
		}
	} else {
		// Release omits the upper evaluation entirely at the source assertion.
		serviceMust(t, e)
		if flags := algebraicCellSourceRecord(t, c, &existing).flags; flags&7 != 7 {
			t.Fatal("release lower sign/minimal publication", flags)
		}
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, nil, nil, &existing, &fresh)
}

func TestAlgebraicCellSourceRootCopyDeletionOrder(t *testing.T) {
	c, p := algebraicCellSourceFixture(t)
	z := c.config.Rationals.Integers()
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
	one, two := unumeral.NewInt(1), unumeral.NewInt(2)
	lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.Int{}, 70), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.Int{}, 70)
	serviceMust(t, z.Mul2k(&one, 70, &(*lower.BorrowNumerator())))
	serviceMust(t, z.Mul2k(&two, 70, &(*upper.BorrowNumerator())))
	t.Cleanup(func() { serviceMust(t, c.config.Binary.Del(&lower)); serviceMust(t, c.config.Binary.Del(&upper)) })
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var src, dst Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&dst)); serviceMust(t, c.Del(&src)) })
	serviceMust(t, c.SetRoot(&src, v, &lower, &upper, true))
	serviceMust(t, c.SetRoot(&dst, v, &lower, &upper, true))
	negative := unumeral.NewInt(-2)
	var scaled unumeral.Int
	serviceMust(t, z.Mul2k(&negative, 70, &scaled))
	t.Cleanup(func() { serviceMust(t, z.Del(&scaled)) })
	for _, a := range []*Algebraic{&src, &dst} {
		r := algebraicCellSourceRecord(t, c, a)
		serviceMust(t, p.SetInts(r.poly, []unumeral.Int{scaled, unumeral.NewInt(0), (*lower.BorrowNumerator())}))
	}
	old := algebraicCellSourceRecord(t, c, &dst)
	if (*(*old.interval.BorrowLower()).BorrowNumerator()).Cell == 0 || (*(*old.interval.BorrowUpper()).BorrowNumerator()).Cell == 0 {
		t.Fatal("fixture must own real interval numerator cells")
	}
	last := (*(*old.interval.BorrowUpper()).BorrowNumerator()).Cell
	lastGeneration := z.Cells.State(arena.Slot(uint32(last))).Generation
	serviceMust(t, c.Copy(&dst, &src))
	after := algebraicCellSourceRecord(t, c, &dst)
	v, e = p.wholeView(after.poly)
	serviceMust(t, e)
	first, e := p.read(v, 0)
	serviceMust(t, e)
	// The actual numeric descriptor freelist is LIFO. Source ascending poly
	// destruction, then lower, then upper leaves old upper as first reuse.
	if uint32((first).Cell) != uint32(last) || z.Cells.State(arena.Slot(uint32((first).Cell))).Generation != lastGeneration+1 {
		t.Fatal("first coefficient did not reuse last destroyed upper descriptor", (first).Cell, last)
	}
	if reflect.TypeOf(unumeral.NumericCellRef(0)).Size() == 8 && (unumeral.ValidNumericRef(z.Cells, (*(*old.interval.BorrowLower()).BorrowNumerator()).Cell) || unumeral.ValidNumericRef(z.Cells, (*(*old.interval.BorrowUpper()).BorrowNumerator()).Cell)) {
		t.Fatal("root Copy retained old interval descriptors")
	}
}

// Prime the actual TLS lane to its strict 100000-byte boundary with one real
// caller-owned allocation. The next allocation publishes the source counter
// and then refuses; no synthetic event, restored counter, or mutable quota.
func algebraicCellSourceTLSBoundary(t *testing.T, c *AlgebraicRootCore) umemory.Block {
	t.Helper()
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	s := runtime.State()
	n := int64(100000) - s.LocalSize - 8
	if n < 0 {
		t.Fatal("unexpected fixture TLS lane extent", s)
	}
	block, e := runtime.Allocate(uint64(n))
	serviceMust(t, e)
	if runtime.State().LocalSize != 100000 {
		t.Fatal("real allocation did not reach exact TLS threshold", runtime.State())
	}
	serviceMust(t, runtime.SetMaxSize(1))
	return block
}

func TestAlgebraicCellSourceConstCopyPrecedesCellAllocation(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.Integers().SetString(&(*q.BorrowNumerator()), "1267650600228229401496703205376")) // 2^100
	before := q
	var a Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) })
	block := algebraicCellSourceTLSBoundary(t, c)
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	e := c.SetRational(&a, &q)
	var failure umemory.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !runtime.State().OutOfMemory {
		t.Fatal("expected real TLS refusal during const numerical copy", e)
	}
	if a != (Algebraic{}) || c.pending != (Algebraic{}) || len(c.locations) != 0 {
		t.Fatal("const source copy failed after cell allocation/publication", a, c.pending)
	}
	if q != before {
		t.Fatal("const copy refusal changed input")
	}
	if c.config.Allocator.Stats().DirectObjects != 0 || c.config.Allocator.Stats().PendingBlocks != 0 {
		t.Fatal("cell allocator visited before scoped rational copy", c.config.Allocator.Stats())
	}
	if c.config.Rationals.Integers().Allocator.Stats().PendingBlocks == 0 {
		t.Fatal("refused actual numerical allocation lost physical pending ownership")
	}
	algebraicCellSourcePhysical(t, c, nil, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, []umemory.Block{block}, &a)
	// Separate explicit caller teardown actions after immediate prefix checks.
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&block))
	serviceMust(t, c.CleanupRetained())
}

func TestAlgebraicCellSourceMutableRefusalDoesNotConsumeInput(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 5, 3))
	before := q
	var a Algebraic
	t.Cleanup(func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) })
	block := algebraicCellSourceTLSBoundary(t, c)
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	e := c.SetTake(&a, &q)
	var failure umemory.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !runtime.State().OutOfMemory {
		t.Fatal("expected actual cell allocation TLS refusal", e)
	}
	if a != (Algebraic{}) || q != before || c.callers != 0 {
		t.Fatal("allocation refusal consumed unpublished caller/input", a, q)
	}
	if c.config.Allocator.Stats().PendingBlocks != 1 {
		t.Fatal("refused cell raw block lacks actual allocator pending owner", c.config.Allocator.Stats())
	}
	algebraicCellSourcePhysical(t, c, nil, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, []umemory.Block{block}, &a)
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&block))
	serviceMust(t, c.config.Allocator.CleanupPending())
}

func TestAlgebraicCellSourceAllPublicationFailurePrefixes(t *testing.T) {
	for route := 0; route < 4; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			failures, success, unpublished, published := 0, 0, 0, 0
			for ordinal := int64(0); ordinal < 256 && success == 0; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					c, p := algebraicCellSourceFixtureProfile(t, true)
					z := c.config.Rationals.Integers()
					q := unumeral.NewRational()
					serviceMust(t, z.SetString(&(*q.BorrowNumerator()), "1267650600228229401496703205376"))
					serviceMust(t, z.SetString(&(*q.BorrowDenominator()), "1208925819614629174706177"))
					poly, e := p.NewPolynomial()
					serviceMust(t, e)
					serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
					lower, upper := unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(1), 0), unumeral.BinaryRationalFromBorrowedNumeral(unumeral.NewInt(2), 0)
					v, e := p.wholeView(poly)
					serviceMust(t, e)
					var src, dst Algebraic
					abortDestination := false
					t.Cleanup(func() {
						if abortDestination {
							serviceMust(t, c.CleanupValue(&dst))
						} else {
							serviceMust(t, c.Del(&dst))
						}
						serviceMust(t, c.Del(&src))
						serviceMust(t, c.CleanupRetained())
						serviceMust(t, p.DelPolynomial(&poly))
						serviceMust(t, c.config.Rationals.Del(&q))
					})
					if route >= 2 {
						serviceMust(t, c.SetRoot(&src, v, &lower, &upper, true))
						if route == 3 {
							serviceMust(t, c.SetRoot(&dst, v, &lower, &upper, false))
						}
						r := algebraicCellSourceRecord(t, c, &src)
						serviceMust(t, p.SetCoefficient(r.poly, 0, &(*q.BorrowNumerator())))
						serviceMust(t, p.SetCoefficient(r.poly, 2, &(*q.BorrowDenominator())))
					}
					budget := z.Budget()
					t.Cleanup(func() { budget.FailAfter(-1) })
					budget.FailAfter(ordinal)
					switch route {
					case 0:
						e = c.SetRational(&dst, &q)
					case 1:
						e = c.SetRoot(&dst, v, &lower, &upper, true)
					default:
						e = c.Copy(&dst, &src)
					}
					// At the exact source refusal: no fault disable, formatting,
					// caller deletion, sourceguard repair, or cleanup has run.
					algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, nil, &src, &dst)
					if e != nil {
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal("unexpected source failure category", e)
						}
						failures++
						if c.pending.slot != 0 {
							unpublished++
						}
						if dst.slot != 0 {
							published++
							l, readError := c.location(&dst)
							serviceMust(t, readError)
							if l.kind == 1 {
								r, readError := c.load(l)
								serviceMust(t, readError)
								if l.coefficientInitialized < r.size {
									before := dst
									rawCell := l.raw
									// Source Del consumes the placement-constructed prefix,
									// then encounters its first unconstructed mpz. This is
									// a guarded undefined continuation, not rollback.
									if delError := c.Del(&dst); !errors.Is(delError, arena.ErrBounds) {
										t.Fatal("incomplete source array Del guard", delError)
									}
									if dst != before || l.raw != rawCell || !l.sourceGuard || l.coefficientCursor != l.coefficientInitialized {
										t.Fatal("guard lost caller/root or ascending deletion cursor", dst, l)
									}
									after, readError := c.load(l)
									serviceMust(t, readError)
									if after.poly != r.poly || after.size != r.size {
										t.Fatal("guard repaired/freed source array metadata")
									}
									algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&(*q.BorrowNumerator()), &(*q.BorrowDenominator())}, nil, &src, &dst)
									// Explicit caller owning abort is confined to cleanup;
									// no source-mark/size repair or successful Del claim.
									abortDestination = true
								}
							}
						}
					} else {
						success++
						if dst.slot == 0 {
							t.Fatal("successful cell construction stayed unpublished")
						}
					}
					budget.FailAfter(-1)
				})
			}
			if failures == 0 || success != 1 {
				t.Fatal("sweep did not reach refusal and actual success", failures, success)
			}
			// Scalar root construction has no coefficient-container reservation.
			// BIG Copy routes below still exercise actual partial numeric copies.
			if route >= 2 && published == 0 {
				t.Fatal("Copy sweep did not cross caller publication before coefficient copy")
			}
			t.Logf("source route%d failures%d firstSuccess%d pendingPrefixes%d publishedPrefixes%d", route, failures, failures, unpublished, published)
		})
	}
}
