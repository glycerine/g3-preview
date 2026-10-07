// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.
package polynomial

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func monomialFixture(t *testing.T) *MonomialManager {
	t.Helper()
	budget := arena.NewBudget(1 << 25)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: budget})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, r.Close())
		if budget.Used() != 0 {
			t.Fatal("real owner refund", budget.Used())
		}
	})
	return m
}
func monomialRead(t *testing.T, m *MonomialManager, a Monomial) []MonomialPower {
	t.Helper()
	info, e := m.Info(a)
	serviceMust(t, e)
	out := make([]MonomialPower, info.Size)
	for i := range out {
		out[i], e = m.ReadPower(a, uint32(i))
		serviceMust(t, e)
	}
	return out
}
func monomialEqualPowers(t *testing.T, m *MonomialManager, a Monomial, want []MonomialPower) {
	t.Helper()
	got := monomialRead(t, m, a)
	if len(got) != len(want) {
		t.Fatal(got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatal(got, want)
		}
	}
}
func monomialOwn(t *testing.T, m *MonomialManager, p []MonomialPower) Monomial {
	t.Helper()
	a, e := m.MkMonomial(p)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	t.Cleanup(func() { serviceMust(t, m.DecRef(a)) })
	return a
}
func TestMonomialSourceConstructorAndInterning(t *testing.T) {
	m := monomialFixture(t)
	unit, e := m.Unit()
	serviceMust(t, e)
	i, e := m.Info(unit)
	serviceMust(t, e)
	if i.ID != 0 || i.Size != 0 || i.TotalDegree != 0 || i.RefCount != 1 {
		t.Fatal(i)
	}
	if m.RuntimeOwner().Stats().LiveBlocks != 6 {
		t.Fatal("table, four actual temp blocks, unit pool", m.RuntimeOwner().Stats())
	}
	for v := uint32(0); v < 10; v++ {
		x, e := m.MkVar()
		serviceMust(t, e)
		if x != v {
			t.Fatal(x, v)
		}
	}
	a := monomialOwn(t, m, []MonomialPower{{0, 2}, {3, 1}, {9, 4}})
	b, e := m.MkMonomial([]MonomialPower{{0, 2}, {3, 1}, {9, 4}})
	serviceMust(t, e)
	if a != b {
		t.Fatal("actual canonical identity", a, b)
	}
	i, e = m.Info(a)
	serviceMust(t, e)
	if i.TotalDegree != 7 || i.RefCount != 1 {
		t.Fatal(i)
	}
	for v := uint32(0); v < 10; v++ {
		d, e := m.DegreeOf(a, v)
		serviceMust(t, e)
		want := uint32(0)
		if v == 0 {
			want = 2
		}
		if v == 3 {
			want = 1
		}
		if v == 9 {
			want = 4
		}
		if d != want {
			t.Fatal(v, d, want)
		}
	}
}
func TestMonomialIndependentProductDivisionGCD(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 6; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	// Independent finite exponent model; no production polynomial algorithm.
	for code := uint32(0); code < 64; code++ {
		left, right := []MonomialPower{}, []MonomialPower{}
		var product, gcd, q1, q2 []MonomialPower
		for v := uint32(0); v < 3; v++ {
			a := (code >> (2 * v)) & 3
			b := (code >> (2 * ((v + 1) % 3))) & 3
			if a != 0 {
				left = append(left, MonomialPower{v, a})
			}
			if b != 0 {
				right = append(right, MonomialPower{v, b})
			}
			if a+b != 0 {
				product = append(product, MonomialPower{v, a + b})
			}
			d := a
			if b < d {
				d = b
			}
			if d != 0 {
				gcd = append(gcd, MonomialPower{v, d})
			}
			if a > d {
				q1 = append(q1, MonomialPower{v, a - d})
			}
			if b > d {
				q2 = append(q2, MonomialPower{v, b - d})
			}
		}
		a, e := m.MkMonomial(left)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(a))
		b, e := m.MkMonomial(right)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(b))
		p, e := m.Mul(a, b)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(p))
		monomialEqualPowers(t, m, p, product)
		d, ok, e := m.Divide(p, a)
		serviceMust(t, e)
		if !ok || d != b {
			t.Fatal("source quotient", d, b, ok)
		}
		g, x, y, e := m.GCD(a, b)
		serviceMust(t, e)
		monomialEqualPowers(t, m, g, gcd)
		if len(gcd) == 0 {
			if x != b || y != a {
				t.Fatal("source no-common GCD swaps remainder outputs")
			}
		} else {
			monomialEqualPowers(t, m, x, q1)
			monomialEqualPowers(t, m, y, q2)
		}
		// Returned intern values start at refcount zero; protect aliases before
		// destroying inputs, then release each genuine owning reference once.
		serviceMust(t, m.IncRef(g))
		serviceMust(t, m.IncRef(x))
		serviceMust(t, m.IncRef(y))
		serviceMust(t, m.DecRef(g))
		serviceMust(t, m.DecRef(x))
		serviceMust(t, m.DecRef(y))
		serviceMust(t, m.DecRef(p))
		serviceMust(t, m.DecRef(b))
		serviceMust(t, m.DecRef(a))
	}
}
func TestMonomialSourceVariableSortPowerAndDerivative(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 5; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	xs := []uint32{3, 1, 3, 0, 1}
	a, e := m.MkFromVariables(xs)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	t.Cleanup(func() { serviceMust(t, m.DecRef(a)) })
	for i, v := range []uint32{0, 1, 1, 3, 3} {
		if xs[i] != v {
			t.Fatal("source sorts caller array", xs)
		}
	}
	monomialEqualPowers(t, m, a, []MonomialPower{{0, 1}, {1, 2}, {3, 2}})
	squared, e := m.Power(a, 2)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(squared))
	t.Cleanup(func() { serviceMust(t, m.DecRef(squared)) })
	root, ok, e := m.Sqrt(squared)
	serviceMust(t, e)
	if !ok || root != a {
		t.Fatal(root, ok)
	}
	_, ok, e = m.Sqrt(a)
	serviceMust(t, e)
	if ok {
		t.Fatal("odd degree square root")
	}
	d, e := m.Derivative(a, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(d))
	t.Cleanup(func() { serviceMust(t, m.DecRef(d)) })
	monomialEqualPowers(t, m, d, []MonomialPower{{0, 1}, {1, 1}, {3, 2}})
}

// Dormant consumer semantics from polynomial.cpp tst_mm1329/1333: the
// registered driver currently returns after PSC before this fixture. This is
// selected monomial sharing/convert evidence, not a completed polynomial test.
func TestMonomialDormantSourceManagerIdentityAndConversion(t *testing.T) {
	a := monomialFixture(t)
	b := monomialFixture(t)
	for i := 0; i < 3; i++ {
		_, e := a.MkVar()
		serviceMust(t, e)
	}
	x := monomialOwn(t, a, []MonomialPower{{0, 2}, {1, 1}, {2, 1}})
	y, e := b.Convert(a, x)
	serviceMust(t, e)
	serviceMust(t, b.IncRef(y))
	t.Cleanup(func() { serviceMust(t, b.DecRef(y)) })
	if x == y {
		t.Fatal("independent owners have distinct scalar identity")
	}
	monomialEqualPowers(t, b, y, []MonomialPower{{0, 2}, {1, 1}, {2, 1}})
	n, e := b.NumVars()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("convert grows actual variables", n)
	}
}

func TestMonomialActualConstructorFailureUnwind(t *testing.T) {
	success := false
	refusals := 0
	for ordinal := int64(0); ordinal < 100 && !success; ordinal++ {
		budget := arena.NewBudget(1 << 25)
		runtime, e := util.NewRuntime(util.RuntimeConfig{Budget: budget})
		serviceMust(t, e)
		budget.FailAfter(ordinal)
		m, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: runtime, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
		if e == nil {
			success = true
			if m.RuntimeOwner() != runtime {
				t.Fatal("borrowed actual runtime")
			}
		} else {
			refusals++
			var failed *MonomialManager
			if errors.As(e, &failed) {
				if failed != m || failed.OriginalCause() == nil {
					t.Fatal("already charged owner itself", m, e)
				}
			} else if !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
		}
		// Inspect roots while the actual refusal remains armed, never infer raw
		// malloc ownership from Budget.Reserve or from a live locator sweep.
		if m != nil {
			monomialPhysicalAudit(t, m)
		}
		budget.FailAfter(-1)
		if m != nil {
			serviceMust(t, m.CloseChecked())
		}
		if runtime.Stats().LiveBlocks != 0 {
			t.Fatal("constructor unwind/owning cleanup real roots", ordinal, runtime.Stats())
		}
		if budget.Used() != runtime.Stats().TotalBytes {
			t.Fatal("retained cold Runtime metadata", ordinal, budget.Used(), runtime.Stats())
		}
		serviceMust(t, runtime.Close())
		if budget.Used() != 0 {
			t.Fatal(ordinal, budget.Used())
		}
	}
	if !success || refusals < 5 {
		t.Fatal("every real construction prefix through success", success, refusals)
	}
}
func TestMonomialTmpSourceFreeBeforeGrowFailure(t *testing.T) {
	m := monomialFixture(t)
	old := m.temps[0].raw
	oldCount, _ := m.RuntimeOwner().AllocationCount()
	m.config.Budget.FailAfter(0)
	powers := make([]MonomialPower, 129)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), 1}
	}
	_, e := m.MkMonomial(powers)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual raw allocation refusal", e)
	}
	if _, e = m.RuntimeOwner().Size(old); e == nil {
		t.Fatal("source old temporary was not physically freed")
	}
	if m.temps[0].raw != (util.Block{}) {
		t.Fatal("source missing temp root must remain a hole")
	}
	count, _ := m.RuntimeOwner().AllocationCount()
	if count != oldCount+1 || !m.RuntimeOwner().State().OutOfMemory {
		t.Fatal("actual source count/OOM publishes before refused physical preparation", count, oldCount, m.RuntimeOwner().State())
	}
	monomialPhysicalAudit(t, m)
	m.config.Budget.FailAfter(-1)
}
func TestMonomialTmpExactCapacitySourceAssertion(t *testing.T) {
	m := monomialFixture(t)
	powers := make([]MonomialPower, 128)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), 1}
	}
	old := m.temps[0].raw
	a, e := m.MkMonomial(powers)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal("source sz<capacity at128", e)
		}
	} else {
		serviceMust(t, e)
		serviceMust(t, m.IncRef(a))
		serviceMust(t, m.DecRef(a))
	}
	if m.temps[0].raw != old {
		t.Fatal("strict assertion does not cause source growth")
	}
	monomialPhysicalAudit(t, m)
}
func TestMonomialSourceRefcountAndExponentWidth(t *testing.T) {
	m := monomialFixture(t)
	_, e := m.MkVar()
	serviceMust(t, e)
	a := monomialOwn(t, m, []MonomialPower{{0, math.MaxUint32}})
	// Scalar width evidence does not require billions of source references.
	if monomialRefIncrement(math.MaxUint32) != 0 || monomialExponentProduct(math.MaxUint32, 2) != math.MaxUint32-1 {
		t.Fatal("source unsigned arithmetic")
	}
	b, e := m.Power(a, 2)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(b))
	t.Cleanup(func() { serviceMust(t, m.DecRef(b)) })
	monomialEqualPowers(t, m, b, []MonomialPower{{0, math.MaxUint32 - 1}})
}

type monomialTestSink struct{ text strings.Builder }

func (s *monomialTestSink) WriteLiteral(v string) error { s.text.WriteString(v); return nil }
func (s *monomialTestSink) WriteUnsigned(v uint32) error {
	var digits [10]byte
	b := strconv.AppendUint(digits[:0], uint64(v), 10)
	s.text.Write(b)
	return nil
}
func (s *monomialTestSink) WriteVariable(v uint32) error {
	s.text.WriteByte('x')
	return s.WriteUnsigned(v)
}
func TestMonomialSourceDisplayParenthesisQuirks(t *testing.T) {
	m := monomialFixture(t)
	_, e := m.MkVar()
	serviceMust(t, e)
	unit, e := m.Unit()
	serviceMust(t, e)
	a := monomialOwn(t, m, []MonomialPower{{0, 1}})
	b := monomialOwn(t, m, []MonomialPower{{0, 3}})
	for _, tc := range []struct {
		a    Monomial
		text string
	}{{unit, "1)"}, {a, "x0)"}, {b, "(* x0 x0 x0)"}} {
		sink := &monomialTestSink{}
		serviceMust(t, m.DisplaySMT2Into(tc.a, sink))
		if sink.text.String() != tc.text {
			t.Fatal(sink.text.String(), tc.text)
		}
	}
	sink := &monomialTestSink{}
	serviceMust(t, m.DisplayInto(b, sink, false))
	if sink.text.String() != "x0^3" {
		t.Fatal(sink.text.String())
	}
}
func TestMonomialSourceRenameAndSearch(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 12; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	powers := make([]MonomialPower, 10)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), uint32(i + 1)}
	}
	a := monomialOwn(t, m, powers)
	for x := uint32(0); x < 12; x++ {
		index, e := m.IndexOf(a, x)
		serviceMust(t, e)
		want := uint32(math.MaxUint32)
		if x < 10 {
			want = x
		}
		if index != want {
			t.Fatal(x, index, want)
		}
		prev, e := m.MaxSmallerThan(a, x)
		serviceMust(t, e)
		want = math.MaxUint32
		if x > 0 {
			want = x - 1
			if want > 9 {
				want = 9
			}
		}
		if prev != want {
			t.Fatal(x, prev, want)
		}
	}
	rename := []uint32{11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	serviceMust(t, m.Rename(rename))
	want := make([]MonomialPower, 10)
	for i := range want {
		want[i] = MonomialPower{uint32(i + 2), uint32(10 - i)}
	}
	monomialEqualPowers(t, m, a, want)
	same, e := m.MkMonomial(want)
	serviceMust(t, e)
	if same != a {
		t.Fatal("rename rebuilds real intern buckets")
	}
}

func TestMonomialSourceInternTemporaryPublicationBeforeStorage(t *testing.T) {
	m := monomialFixture(t)
	_, e := m.MkVar()
	serviceMust(t, e)
	before, _ := m.ids.Range()
	m.config.Budget.FailAfter(0)
	_, e = m.MkVariable(0, 1)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual source object allocation refusal", e)
	}
	if m.creation.stage == 0 || m.table.size != 2 {
		t.Fatal("unit plus actually published temporary table key", m.creation, m.table.size)
	}
	key, e := m.table.key(m, m.creation.tableIndex)
	serviceMust(t, e)
	if key.kind != monomialTemporaryKey || key.slot != 0 {
		t.Fatal("source temporary key must precede permanent object", key)
	}
	after, _ := m.ids.Range()
	if after != before {
		t.Fatal("ID consumed before refused source storage", after, before)
	}
	monomialPhysicalAudit(t, m)
	m.config.Budget.FailAfter(-1)
	serviceMust(t, m.CleanupCreation())
	if m.creation.stage != 0 || m.table.size != 1 {
		t.Fatal("owning abort retains unit and removes only actual pending binding")
	}
}

func TestMonomialOwningCloseRejectsBorrowedUnitReference(t *testing.T) {
	m := monomialFixture(t)
	u, e := m.Unit()
	serviceMust(t, e)
	serviceMust(t, m.IncRef(u))
	before := m.config.Budget.Used()
	e = m.CloseChecked()
	if !errors.Is(e, ErrMonomialLive) {
		t.Fatalf("owning Close consumed caller unit reference: %v", e)
	}
	if m.closing || m.config.Budget.Used() != before {
		t.Fatal("caller refusal must precede owning stop")
	}
	serviceMust(t, m.DecRef(u))
}
func TestMonomialCompletedSourceDestructorReverseMembers(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: budget})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	var temps [4]util.Block
	for i := range temps {
		temps[i] = m.temps[i].raw
	}
	table := m.table.raw
	serviceMust(t, m.SourceFinalize())
	if m.sourceStage != 10 || r.Stats().LiveBlocks != 0 {
		t.Fatal("literal member destruction", m.sourceStage, r.Stats())
	}
	for _, b := range temps {
		if _, e = r.Read(b, 0); e == nil {
			t.Fatal("destroyed temp remains live")
		}
	}
	if _, e = r.Read(table, 0); e == nil {
		t.Fatal("destroyed table remains live")
	}
	if _, e = m.Unit(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary owner guard", e)
	}
	monomialPhysicalAudit(t, m)
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if budget.Used() != 0 {
		t.Fatal(budget.Used())
	}
}

func TestMonomialSourceFinalizeCallerLifetime(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(strconv.FormatBool(direct), func(t *testing.T) {
			b := arena.NewBudget(1 << 20)
			r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
			serviceMust(t, e)
			profile := util.SmallAllocatorProfile{PointerBits: 64, Debug: direct}
			m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: profile})
			serviceMust(t, e)
			_, e = m.MkVar()
			serviceMust(t, e)
			a, e := m.MkVariable(0, 2)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(a))
			l, e := m.location(a)
			serviceMust(t, e)
			raw, _, e := m.allocator.RawLocation(l.raw)
			serviceMust(t, e)
			e = m.SourceFinalize()
			if polynomialSourceDebug {
				if !errors.Is(e, ErrMonomialAssertion) || m.sourceStage != 1 || m.unit != (Monomial{}) {
					t.Fatal("source debug unit deletion before table assertion", e, m.sourceStage)
				}
				if _, e = r.Read(raw, 0); e != nil {
					t.Fatal("debug stop freed caller backing", e)
				}
				serviceMust(t, m.CleanupValue(&a))
				serviceMust(t, m.SourceFinalize())
			} else {
				serviceMust(t, e)
				if direct {
					if _, e = r.Read(raw, 0); e != nil {
						t.Fatal("direct caller raw must outlive source allocator", e)
					}
				} else if _, e = r.Read(raw, 0); e == nil {
					t.Fatal("pooled backing not consumed")
				}
				if direct {
					if e = r.Close(); !errors.Is(e, util.ErrRuntimeLive) {
						t.Fatal("live runtime root guard", e)
					}
				}
				serviceMust(t, m.CleanupValue(&a))
			}
			serviceMust(t, m.CloseChecked())
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
	}
}

func TestMonomialSourceInternFailurePrefixesAndRetry(t *testing.T) {
	refusals, success := 0, false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		b := arena.NewBudget(1 << 22)
		r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
		serviceMust(t, e)
		m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r})
		serviceMust(t, e)
		var owned [20]Monomial
		for i := range owned {
			_, e = m.MkVar()
			serviceMust(t, e)
			owned[i], e = m.MkVariable(uint32(i), 1)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(owned[i]))
		}
		_, e = m.MkVar()
		serviceMust(t, e)
		b.FailAfter(ordinal)
		var newPowers [21]MonomialPower
		for i := range newPowers {
			newPowers[i] = MonomialPower{uint32(i), 2}
		}
		a, e := m.MkMonomial(newPowers[:])
		monomialPhysicalAudit(t, m)
		if e != nil {
			refusals++
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(ordinal, e)
			}
		} else {
			success = true
			serviceMust(t, m.IncRef(a))
		}
		b.FailAfter(-1)
		if e != nil {
			serviceMust(t, m.CleanupCreation())
			if m.table.pending != (util.Block{}) {
				serviceMust(t, m.memory.Deallocate(&m.table.pending))
			}
		} else {
			serviceMust(t, m.DecRef(a))
		}
		for _, v := range owned {
			serviceMust(t, m.DecRef(v))
		}
		serviceMust(t, m.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(ordinal, b.Used())
		}
		if success {
			break
		}
	}
	if !success || refusals == 0 {
		t.Fatal("full bounded refusal-to-success campaign", refusals, success)
	}
}
func TestMonomialDefaultSourceProfileAndWarmPhysicalGraph(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r})
	serviceMust(t, e)
	_, e = m.MkVar()
	serviceMust(t, e)
	a, e := m.MkVariable(0, 3)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	before := r.Stats()
	budget := b.Used()
	count, e := r.AllocationCount()
	serviceMust(t, e)
	var problem error
	wrong := false
	allocs := testing.AllocsPerRun(20, func() {
		got, e := m.MkVariable(0, 3)
		if e != nil {
			problem = e
		}
		if got != a {
			wrong = true
		}
	})
	if problem != nil || wrong || allocs != 0 {
		t.Fatal(problem, wrong, allocs)
	}
	afterCount, e := r.AllocationCount()
	serviceMust(t, e)
	if afterCount != count || r.Stats() != before || b.Used() != budget {
		t.Fatal("intern lookup invented source allocation", count, afterCount, budget, b.Used())
	}
	monomialPhysicalAudit(t, m)
	serviceMust(t, m.DecRef(a))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMonomialSourceRetainedPhysicalPowerAccess(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 2; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	a := monomialOwn(t, m, []MonomialPower{{0, 1}, {1, 2}})
	l, e := m.location(a)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 1))
	p, e := m.ReadPower(a, 1)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else if e != nil || p != (MonomialPower{1, 2}) {
		t.Fatal("release reads initialized physical power", p, e)
	}
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 2))
}

// This bounded public Runtime recipe reaches guarded source signed overflow;
// it is not valid allocation exhaustion or an allocation-limit free refusal.
func TestMonomialSourceTempPostphysicalReleaseAcknowledgment(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	p := make([]MonomialPower, 20000)
	for i := range p {
		p[i] = MonomialPower{uint32(i), 1}
	}
	a, e := m.MkMonomial(p)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	serviceMust(t, m.DecRef(a))
	old := m.temps[0].raw
	contextTLSNearMinimum(t, r)
	before := r.Stats()
	e = m.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || m.sourceStage != 7 || m.temps[0].raw != (util.Block{}) {
		t.Fatal("actual source free then TLS guard", e, m.sourceStage, m.temps[0].raw)
	}
	if _, e = r.Read(old, 0); e == nil {
		t.Fatal("old temp remains physical")
	}
	if r.Stats().LiveBlocks >= before.LiveBlocks {
		t.Fatal("no actual completed physical frees")
	}
	e = m.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || m.sourceStage != 8 {
		t.Fatal("next actual ID-vector free retains separate TLS error", e, m.sourceStage)
	}
	e = m.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || m.sourceStage != 9 || m.table.raw != (util.Block{}) {
		t.Fatal("next actual table free retains separate TLS error", e, m.sourceStage)
	}
	serviceMust(t, m.SourceFinalize())
	if m.sourceStage != 10 {
		t.Fatal(m.sourceStage)
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMonomialSourceRawPowerArrayProduct(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 4; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	left := []MonomialPower{{0, 2}, {2, 3}}
	right := []MonomialPower{{1, 5}, {2, 4}, {3, 1}}
	r, e := m.MulPowers(left, right)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(r))
	monomialEqualPowers(t, m, r, []MonomialPower{{0, 2}, {1, 5}, {2, 7}, {3, 1}})
	if m.publishedObjects != 2 {
		t.Fatal("raw overload invented operand monomial owners", m.publishedObjects)
	}
	serviceMust(t, m.DecRef(r))
}

func TestMonomialBorrowedAllocatorClosedGuardBeforeHeader(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	a, e := util.NewSmallAllocator(util.SmallAllocatorConfig{Runtime: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	serviceMust(t, a.Close())
	before := r.Stats()
	used := b.Used()
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Allocator: a})
	if m != nil || !errors.Is(e, arena.ErrClosed) || b.Used() != used || r.Stats() != before {
		t.Fatal("closed borrowed owner before header", m, e, used, b.Used(), before, r.Stats())
	}
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMonomialSourceRepeatedRenameFreshLexicalTable(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 3; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	a := monomialOwn(t, m, []MonomialPower{{0, 1}})
	b := monomialOwn(t, m, []MonomialPower{{2, 2}})
	serviceMust(t, m.Rename([]uint32{2, 1, 0}))
	serviceMust(t, m.Rename([]uint32{2, 1, 0}))
	if m.table.size != 3 {
		t.Fatal("each source lexical new_table begins empty", m.table.size)
	}
	monomialEqualPowers(t, m, a, []MonomialPower{{0, 1}})
	monomialEqualPowers(t, m, b, []MonomialPower{{2, 2}})
}

func TestMonomialScalarQueriesStopAtOwnerLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r})
	serviceMust(t, e)
	_, e = m.MkVar()
	serviceMust(t, e)
	n, e := m.NumVars()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	valid, e := m.IsValidVar(0)
	serviceMust(t, e)
	if !valid {
		t.Fatal("real variable")
	}
	serviceMust(t, m.SourceFinalize())
	if _, e = m.NumVars(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if _, e = m.IsValidVar(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
}

func TestMonomialOwningCloseAfterUnitIDRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r})
	serviceMust(t, e)
	b.FailAfter(0)
	e = m.CloseChecked()
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual unit ID vector reservation", e)
	}
	if !m.closing || m.unit == (Monomial{}) {
		t.Fatal("retained real private unit", m.closing, m.unit)
	}
	monomialPhysicalAudit(t, m)
	if _, e = m.Unit(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMonomialOwningCloseRetainsPostphysicalIDLocal(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	s := r.State()
	if s.LocalSize <= 0 || s.LocalSize >= 100000 {
		t.Fatal(s)
	}
	padding, e := r.Allocate(uint64(100000-s.LocalSize) - 8)
	serviceMust(t, e)
	if r.State().LocalSize != 100000 {
		t.Fatal("strict source TLS threshold", r.State())
	}
	serviceMust(t, r.SetMaxCount(1))
	before := r.Stats()
	e = m.CloseChecked()
	var failure util.Failure
	if !errors.As(e, &failure) || failure.Kind != util.FailureAllocExceeded || failure.ErrorCode() != 113 {
		t.Fatal("postphysical ID allocation-count refusal", e)
	}
	roots := m.ids.RawRoots()
	if roots[1] == (util.Block{}) || roots[0] != (util.Block{}) || r.Stats().LiveBlocks != before.LiveBlocks+1 {
		t.Fatal("unpublished actual ID vector local", roots, before, r.Stats())
	}
	monomialPhysicalAudit(t, m, padding)
	if !m.closing || m.unit == (Monomial{}) {
		t.Fatal("actual stopped private unit", m.closing, m.unit)
	}
	serviceMust(t, r.SetMaxCount(0))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Deallocate(&padding))
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMonomialSourceMaxSmallerNullAssertion(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 2; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	a := monomialOwn(t, m, []MonomialPower{{0, 1}, {1, 2}})
	v, e := m.MaxSmallerThan(a, math.MaxUint32)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal("actual source null_var assertion", v, e)
		}
	} else if e != nil || v != 1 {
		t.Fatal(v, e)
	}
}

func TestMonomialSourceMissingTempHoleKeepsStoppedHistory(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	var p [129]MonomialPower
	for i := range p {
		p[i] = MonomialPower{uint32(i), 1}
	}
	b.FailAfter(0)
	_, e = m.MkMonomial(p[:])
	if !numericPhysicalReservationRefusal(e) || m.temps[0].raw != (util.Block{}) {
		t.Fatal("actual free-before-allocation hole", e)
	}
	monomialPhysicalAudit(t, m)
	b.FailAfter(-1)
	e = m.SourceFinalize()
	if !errors.Is(e, ErrMonomialPending) || m.sourceStage != 7 || m.temps[0].destroyConsumed {
		t.Fatal("missing source pointer is not completed destructor", e, m.sourceStage, m.temps[0])
	}
	monomialPhysicalAudit(t, m)
	serviceMust(t, m.CloseChecked())
	if m.sourceStage != 7 {
		t.Fatal("owning cleanup invented source completion", m.sourceStage)
	}
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// These are named incomplete physical objects, not completed source cells.
// Their owning abort releases the real allocation without inventing fields.
func monomialReadProfileObject(t *testing.T, m *MonomialManager, n uint32) Monomial {
	t.Helper()
	slot, e := m.prepareLocation()
	serviceMust(t, e)
	obj, e := m.allocator.Allocate(uint64(20 + 8*n))
	serviceMust(t, e)
	l := &m.locations[slot-1]
	l.raw = obj
	l.requested = 20 + 8*n
	l.live = true
	l.sourceRaw, l.sourceOffset, e = m.allocator.RawLocation(obj)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(obj, 12, n))
	a := Monomial{m.origin, slot, l.generation}
	t.Cleanup(func() { serviceMust(t, m.CleanupValue(&a)) })
	return a
}
func TestMonomialSourceSingletonValidityDoesNotReadPower(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	ok, e := m.validPowerProductCore(a, 1)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	ok, e = (monomialInput{manager: m, value: a, count: 1}).valid()
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	l, e := m.location(a)
	serviceMust(t, e)
	_, e = m.allocator.ReadUint32(l.raw, 20)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
}
func TestMonomialSourceDegreeSumDoesNotReadVariable(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 1)
	l, e := m.location(a)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 24, 7))
	n, e := m.degreeSumCore(a, 1)
	serviceMust(t, e)
	if n != 7 {
		t.Fatal(n)
	}
	n, e = (monomialInput{manager: m, value: a, count: 1}).degreeSum()
	serviceMust(t, e)
	if n != 7 {
		t.Fatal(n)
	}
	_, e = m.allocator.ReadUint32(l.raw, 20)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
}
func TestMonomialSourceValidityFieldShortCircuit(t *testing.T) {
	m := monomialFixture(t)
	a := monomialReadProfileObject(t, m, 2)
	l, e := m.location(a)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 20, 2))
	serviceMust(t, m.allocator.WriteUint32(l.raw, 28, 1))
	ok, e := m.IsValid(a)
	serviceMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	// Neither uninitialized degrees nor unrelated ID/refcount/hash is reached.
	serviceMust(t, m.allocator.WriteUint32(l.raw, 28, 3))
	_, e = m.IsValid(a)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	serviceMust(t, m.allocator.WriteUint32(l.raw, 24, 4))
	serviceMust(t, m.allocator.WriteUint32(l.raw, 32, 5))
	serviceMust(t, m.allocator.WriteUint32(l.raw, 8, 9))
	ok, e = m.IsValid(a)
	serviceMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
}
