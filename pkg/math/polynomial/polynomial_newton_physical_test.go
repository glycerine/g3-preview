// Source: src/math/polynomial/polynomial.cpp:1885-2338,2481-2633.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Independent named field census for this selected owner and its ordinary
// unsynchronized borrowed rational cohort. This does not infer caller roots
// from the locator directory, nor cover unrelated rational arithmetic scopes.
func polynomialNewtonPhysicalAudit(t *testing.T, m *MultivariatePolynomialCore, owners []*PolynomialNewtonInterpolator, references []*PolynomialRef, refvectors []*PolynomialRefVector, lazy []*PolynomialNewtonVector, polys []Polynomial, handlers []*PolynomialDeletionHandler, public ...*util.Int) {
	polynomialNewtonPhysicalWithRaw(t, m, owners, references, refvectors, lazy, polys, handlers, nil, public...)
}

func polynomialNewtonPhysicalWithRaw(t *testing.T, m *MultivariatePolynomialCore, owners []*PolynomialNewtonInterpolator, references []*PolynomialRef, refvectors []*PolynomialRefVector, lazy []*PolynomialNewtonVector, polys []Polynomial, handlers []*PolynomialDeletionHandler, external []util.Block, public ...*util.Int) {
	polynomialNewtonPhysicalWithExtra(t, m, owners, references, refvectors, lazy, polys, handlers, external, 0, public...)
}
func polynomialNewtonPhysicalWithExtra(t *testing.T, m *MultivariatePolynomialCore, owners []*PolynomialNewtonInterpolator, references []*PolynomialRef, refvectors []*PolynomialRefVector, lazy []*PolynomialNewtonVector, polys []Polynomial, handlers []*PolynomialDeletionHandler, external []util.Block, adapterBytes uint64, public ...*util.Int) {
	t.Helper()
	z := m.integer
	refs := map[util.NumericCellRef]bool{}
	add := func(a util.Int) {
		if (a).Cell == 0 || z.IsFailureRef((a).Cell) {
			return
		}
		if !z.IsLiveCellToken(uint64((a).Cell)) {
			t.Fatal("invalid named numeral", a)
		}
		refs[(a).Cell] = true
	}
	for _, r := range z.Tmp {
		add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(r), uint32(uint64(r) >> 32)}))
	}
	for _, r := range z.Arg {
		add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(r), uint32(uint64(r) >> 32)}))
	}
	if z.MatchesDividesNumeralInvocation(&z.DividesNumeral.Receipt) {
		add(z.DividesNumeral.Numeral)
	}
	for _, s := range []*util.IntegerRawScope{&z.DivisionRaw, &z.ExtendedRaw, &z.GcdRaw} {
		for i := uint8(0); i < s.Count; i++ {
			add(s.Values[i])
		}
	}
	if z.MatchesAddMulInvocation(&z.GcdRaw.AddMul) || z.MatchesAddMulInvocation(&z.AddMulInvocation.Receipt) {
		add(z.AddMulInvocation.Tmp)
	}
	add(z.IntMin)
	add(z.Two64)
	public = append(public, &m.substitution.scope.newA, &m.substitution.scope.tmp)
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		public = append(public, &b.cohortMovingCoefficient, &b.appendScope.movingCoefficient)
	}
	if m.transportBound {
		public = append(public, &m.transport)
	}
	for _, a := range public {
		if a != nil {
			add(*a)
		}
	}
	q := m.config.Rationals
	if q != nil {
		for _, a := range []util.Int{q.NTmp, q.AddTmp1, q.AddTmp2, q.AddMulTmp.Num, q.AddMulTmp.Den, q.LtTmp1.Num, q.LtTmp1.Den, q.LtTmp2.Num, q.LtTmp2.Den} {
			add(a)
		}
	}
	for _, a := range []util.Int{(*m.modular.BorrowModulus()), m.modular.Lower, m.modular.Upper, m.modular.InvTmp1, m.modular.InvTmp2, m.modular.InvTmp3, m.modular.DivTmp, m.zeroNumeral, m.minusOne, m.negativeCoefficient, m.constantCopy, m.linearConstant, m.constantScope.numeral, (*m.constantScope.rational.BorrowNumerator()), (*m.constantScope.rational.BorrowDenominator()), m.exactDivision.numeral, m.pseudoDivision.numeral, m.dividesLocal.numeral} {
		add(a)
	}
	cell := func(p Polynomial, n uint32) {
		for i := uint32(0); i < n; i++ {
			a, e := m.coefficientCore(p, i)
			serviceMust(t, e)
			add(a)
		}
	}
	for _, p := range []Polynomial{m.zero, m.unit, m.constantScope.result, m.overloadResult, m.negativeResult} {
		if p != (Polynomial{}) {
			n, e := m.readWord(p, 8)
			serviceMust(t, e)
			cell(p, n)
		}
	}
	if m.pending.value != (Polynomial{}) {
		l, e := m.polynomialLocation(m.pending.value)
		serviceMust(t, e)
		cell(m.pending.value, l.initializedCoefficients)
	}
	vectors := []*polynomialVector{&m.rationalTerms, &m.linearTerms, &m.linearMonomials, &m.polynomials}
	extraBytes := adapterBytes
	addPolynomial := func(p Polynomial) {
		if p == (Polynomial{}) {
			return
		}
		l, e := m.polynomialLocation(p)
		serviceMust(t, e)
		if l.released {
			return
		}
		cell(p, l.initializedCoefficients)
	}
	addPolynomial(m.exactDivision.returned)
	for i := range m.pseudoDivision.refs {
		addPolynomial(m.pseudoDivision.refs[i].value)
		addPolynomial(m.pseudoDivision.refs[i].incoming)
	}
	if m.powerLocal.active {
		addPolynomial(m.powerLocal.result.value)
		addPolynomial(m.powerLocal.result.incoming)
	}
	polys = append(polys, polynomialSubstitutionNamedPolynomials(m)...)
	polys = append(polys, m.translation.freshResult)
	for _, p := range polys {
		addPolynomial(p)
	}
	for _, r := range references {
		extraBytes += r.header
		addPolynomial(r.root.value)
		addPolynomial(r.root.incoming)
	}
	for _, v := range refvectors {
		extraBytes += v.header
		vectors = append(vectors, &v.state.values)
		count, e := v.state.values.size(m)
		serviceMust(t, e)
		for i := v.state.drain; i < count; i++ {
			p, e := v.state.get(m, i)
			serviceMust(t, e)
			addPolynomial(p)
		}
		addPolynomial(v.state.incoming)
	}
	for _, v := range lazy {
		extraBytes += v.header + uint64(cap(v.children))*uint64(reflect.TypeOf((*PolynomialNewtonInterpolator)(nil)).Size())
		vectors = append(vectors, &v.data)
		for _, n := range v.children {
			if n != nil {
				owners = append(owners, n)
			}
		}
		if v.pending != nil {
			found := false
			for _, n := range owners {
				if n == v.pending {
					found = true
				}
			}
			if !found {
				owners = append(owners, v.pending)
			}
		}
	}
	seenNewton := map[*PolynomialNewtonInterpolator]bool{}
	for _, n := range owners {
		if n == nil || seenNewton[n] {
			continue
		}
		seenNewton[n] = true
		extraBytes += n.header
		vectors = append(vectors, &n.inputs, &n.invs, &n.vs.values)
		for _, a := range []util.Int{n.scope.product, n.scope.aux, n.scope.c} {
			add(a)
		}
		for _, r := range []*polynomialReference{&n.scope.temp, &n.scope.auxPoly, &n.scope.u} {
			addPolynomial(r.value)
			addPolynomial(r.incoming)
		}
		count, e := n.vs.values.size(m)
		serviceMust(t, e)
		for i := n.vs.drain; i < count; i++ {
			p, e := n.vs.get(m, i)
			serviceMust(t, e)
			addPolynomial(p)
		}
		addPolynomial(n.vs.incoming)
	}

	buffers := [4]*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2}
	for _, b := range buffers {
		add(b.gcd)
		add(b.cohortMovingCoefficient)
		add(b.appendScope.movingCoefficient)
		add(b.arithmeticScope.transport)
		extraBytes += uint64(cap(b.coefficientOwners)+cap(b.monomialOwners)+cap(b.coefficientTransferred)+cap(b.monomialTransferred)) * 8
		vectors = append(vectors, &b.as, &b.ms)
	}
	for _, v := range vectors {
		if v.width != 12 {
			continue
		}
		if v.raw != (util.Block{}) {
			for i := uint32(0); i < v.initialized; i++ {
				a, e := v.coefficient(m, i)
				serviceMust(t, e)
				add(a)
			}
		}
		if v.old != (util.Block{}) {
			old := polynomialVector{raw: v.old, width: 12}
			for i := v.moved; i < v.oldInitialized; i++ {
				a, e := old.coefficient(m, i)
				serviceMust(t, e)
				add(a)
			}
		}
		for i := uint32(0); i < v.retiredActive; i++ {
			add(v.retired[i])
		}
	}
	for slot := uint64(1); slot <= uint64(z.Cells.Stats().Slots); slot++ {
		c, e := z.Cells.Read(arena.Slot(slot))
		if e == nil && (c.Unpublished || c.CleanupPending) {
			add(util.IntFromNativeWords([3]uint32{uint32(0), uint32(util.MakeNumericRef(arena.Slot(slot), z.Cells.State(arena.Slot(slot)).Generation)), uint32(uint64(util.MakeNumericRef(arena.Slot(slot), z.Cells.State(arena.Slot(slot)).Generation)) >> 32)}))
		}
	}
	if uint32(len(refs)) != z.Cells.Stats().Live {
		t.Fatal("unnamed actual numeric descriptor", len(refs), z.Cells.Stats().Live)
	}
	for ref := range refs {
		c := z.Cell(ref)
		if c.Payload != (arena.VectorRef{}) {
			t.Fatal("fake Words cell")
		}
		if !c.Released {
			o, e := z.CellObject(ref)
			serviceMust(t, e)
			_, e = z.SourceAllocator().ObjectExtent(o)
			serviceMust(t, e)
		}
	}
	if z.PendingAllocationObject() != (util.ObjectRef{}) {
		_, e := z.SourceAllocator().ObjectExtent(z.PendingAllocationObject())
		serviceMust(t, e)
	}
	raw := map[util.Block]bool{}
	put := func(b util.Block) {
		if b == (util.Block{}) {
			return
		}
		if raw[b] {
			t.Fatal("duplicate named physical root")
		}
		raw[b] = true
	}
	external = append(external, polynomialSubstitutionNamedRaw(m)...)
	for _, b := range external {
		put(b)
	}
	mm := m.monomials
	for _, r := range []util.Block{mm.table.raw, mm.table.pending, mm.renameTable.raw, mm.renameTable.pending, mm.renameFound, mm.powers.raw, mm.powers.pending} {
		put(r)
	}
	for _, tmp := range mm.temps {
		put(tmp.raw)
		put(tmp.pending)
	}
	for _, v := range vectors {
		put(v.raw)
		put(v.pending)
		put(v.old)
	}
	for _, b := range buffers {
		put(b.positions.raw)
		put(b.positions.pending)
	}
	for _, s := range z.Naturals().Division {
		put(s.Raw)
		put(s.Pending)
	}
	if z.Naturals().VerificationActive {
		put(z.Naturals().Verification.Raw)
		put(z.Naturals().Verification.Pending)
	}
	if z.Naturals().Formatting.Active || z.Naturals().Formatting.PendingField {
		for _, s := range z.Naturals().Formatting.Locals {
			put(s.Raw)
			put(s.Pending)
		}
	}
	put(z.Naturals().CharFormatting.Buffer.Raw)
	put(z.Naturals().CharFormatting.Buffer.PendingField)
	za, ma := z.SourceAllocator().Stats(), mm.allocator.Stats()
	if uint32(len(raw))+za.Chunks+za.DirectObjects+za.PendingBlocks+ma.Chunks+ma.DirectObjects+ma.PendingBlocks != m.memory.Stats().LiveBlocks {
		t.Fatal("unnamed Runtime root", len(raw), za, ma, m.memory.Stats())
	}
	bytes := uint64(cap(m.substitution.orphans))*16 + uint64(cap(m.linearGenerated)) + uint64(m.ids.FreeCapacity()+mm.ids.FreeCapacity())*4 + integerActualOwnedBytes(z) + m.header + mm.header + uint64(cap(mm.locations))*uint64(reflect.TypeOf(monomialLocation{}).Size()) + ma.TotalBytes + uint64(cap(m.locations))*uint64(reflect.TypeOf(polynomialLocation{}).Size())
	if !m.modular.Closed() {
		bytes += m.modular.Header
	}
	if q != nil {
		bytes += q.HeaderBytes
	}
	for _, v := range vectors {
		bytes += uint64(cap(v.retired)) * uint64(reflect.TypeOf(util.Int{}).Size())
	}
	for _, h := range handlers {
		if h == nil || h.self != h || h.budget != z.Budget() {
			t.Fatal("invalid named handler")
		}
		bytes += h.header
	}
	bytes += extraBytes
	if bytes != z.Budget().Used() {
		t.Fatal("actual charged headers and backings", bytes, z.Budget().Used())
	}
}
