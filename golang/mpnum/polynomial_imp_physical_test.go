// Source: src/math/polynomial/polynomial.cpp:1821-1850,2324-2370. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func polynomialImpPhysicalAudit(t *testing.T, full *MultivariatePolynomialManager, supplied ...*memory.SmallAllocator) {
	t.Helper()
	polynomialImpPhysicalAuditValues(t, full, nil, supplied...)
}

func polynomialImpPhysicalAuditValues(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, supplied ...*memory.SmallAllocator) {
	t.Helper()
	polynomialImpPhysicalAuditNamed(t, full, polys, nil, 0, supplied...)
}
func polynomialImpPhysicalAuditNamed(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*Int, externalHeaders uint64, supplied ...*memory.SmallAllocator) {
	polynomialImpPhysicalAuditNamedExtra(t, full, polys, input, externalHeaders, nil, supplied...)
}
func polynomialImpPhysicalAuditNamedExtra(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*Int, externalHeaders uint64, extraRaw []memory.Block, supplied ...*memory.SmallAllocator) {
	t.Helper()
	m := full.core
	owners := []*PolynomialNewtonInterpolator{}
	references := []*PolynomialRef{}
	refvectors := []*PolynomialRefVector{}
	lazy := []*PolynomialNewtonVector{&full.newton}
	handlers := []*PolynomialDeletionHandler{}
	external := []memory.Block{m.tmp1.raw, m.tmp1.pending, full.degree2pos.raw, full.degree2pos.pending, full.lexPermutation.raw, full.lexPermutation.pending, full.varDegrees.raw, full.varDegrees.pending, m.substitution.found.raw, m.substitution.found.pending, m.substitution.max.maxDegree.raw, m.substitution.max.maxDegree.pending, m.substitution.max.variables.raw, m.substitution.max.variables.pending, m.substitution.positions.positions.raw, m.substitution.positions.positions.pending}
	external = append(external, extraRaw...)
	external = append(external, m.queryPositions.raw, m.queryPositions.pending)
	external = append(external, full.lexBuckets.data.raw, full.lexBuckets.data.pending, full.lexBuckets.data.old)
	for i := range full.lexBuckets.children {
		v := &full.lexBuckets.children[i]
		external = append(external, v.raw, v.pending, v.old)
	}
	adapterBytes := uint64(cap(m.quasiHistory))*uint64(reflect.TypeOf(polynomialQuasiResultantHistory{}).Size()) + full.header + uint64(cap(full.lexBuckets.children))*uint64(reflect.TypeOf(polynomialVector{}).Size())
	public := []*Int{&full.derivative.n, &full.derivative.a, &full.composition.a}
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		public = append(public, &b.cohortMovingCoefficient, &b.appendScope.movingCoefficient, &b.arithmeticScope.transport)
	}
	if m.transportBound {
		public = append(public, &m.transport)
	}
	if m.scalarDivision.transportBound {
		public = append(public, &m.scalarDivision.transport)
	}
	for _, raw := range []*integerScalarRawScope{&m.scalarDivision.division, &m.scalarDivision.extended} {
		for i := uint8(0); i < raw.count; i++ {
			public = append(public, &raw.values[i])
		}
	}

	external = append(external, full.conversion.ms.raw, full.conversion.ms.pending)
	adapterBytes += uint64(cap(full.conversion.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	if b := &full.conversion.as; b.constructed {
		external = append(external, b.raw, b.pending)
		for i := uint32(0); i < b.count; i++ {
			a, e := b.physical(i)
			serviceMust(t, e)
			public = append(public, &a)
		}
	}
	if b := &full.toPolynomial.buffer; b.constructed {
		external = append(external, b.raw, b.pending)
		for i := uint32(0); i < b.count; i++ {
			a, e := b.physical(i)
			serviceMust(t, e)
			public = append(public, &a)
		}
	}
	if m.substitution.scope.active {
		if m.substitution.scope.newAConstructed {
			public = append(public, &m.substitution.scope.newA)
		}
		if m.substitution.scope.tmpConstructed {
			public = append(public, &m.substitution.scope.tmp)
		}
	}
	if full.cra.active {
		for i := uint8(0); i < full.cra.constructed; i++ {
			public = append(public, &full.cra.nums[i])
		}
	}
	if full.normalization.active {
		if full.normalization.gConstructed {
			public = append(public, &full.normalization.g)
		}
		if full.normalization.aConstructed {
			public = append(public, &full.normalization.a)
		}
	}
	if m.contentLocal.active {
		public = append(public, &m.contentLocal.ai)
	}
	if full.monic.active {
		public = append(public, &full.monic.inv, &full.monic.newA)
	}
	for i := range full.composition.raw {
		public = append(public, &full.composition.raw[i])
	}
	for i := range full.squareRoot.values {
		public = append(public, &full.squareRoot.values[i])
	}
	public = append(public, &full.squareRoot.two)
	public = append(public, input...)
	adapterBytes += uint64(cap(full.squareRoot.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.squareRoot.orphanInvocations)) * 8

	adapterBytes += uint64(cap(full.gcdFrames)) * uint64(reflect.TypeOf((*polynomialGCDFunctionScope)(nil)).Size())
	for _, f := range full.gcdFrames {
		adapterBytes += f.header
		if !f.active {
			continue
		}
		for i := range f.nums {
			public = append(public, &f.nums[i])
		}
		if f.zp.active {
			public = append(public, &f.zp.p)
		}
		for i := range f.refs {
			references = append(references, &f.refs[i])
		}
		for i := range f.powers {
			external = append(external, f.powers[i].raw, f.powers[i].pending)
		}
		external = append(external, f.vars.raw, f.vars.pending, f.monomialBuffer.raw, f.monomialBuffer.pending)
	}
	adapterBytes += uint64(cap(full.iccpFrames)) * uint64(reflect.TypeOf((*polynomialICCPScope)(nil)).Size())
	for _, f := range full.iccpFrames {
		adapterBytes += f.header
	}
	adapterBytes += uint64(cap(m.substitution.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += externalHeaders
	adapterBytes += uint64(cap(m.display.frames))*uint64(reflect.TypeOf((*polynomialDisplayFrame)(nil)).Size()) + uint64(len(m.display.frames))*uint64(reflect.TypeOf(polynomialDisplayFrame{}).Size())
	for _, f := range m.display.frames {
		public = append(public, &f.numeral)
		external = append(external, f.characters.raw, f.characters.pending)
		for _, b := range f.limbs.locals {
			external = append(external, b.raw, b.pending)
		}
	}
	es := &full.evaluation
	adapterBytes += uint64(cap(es.rationalFrames))*uint64(reflect.TypeOf((*polynomialEvalRationalFrame)(nil)).Size()) + uint64(len(es.rationalFrames))*uint64(reflect.TypeOf(polynomialEvalRationalFrame{}).Size())
	adapterBytes += uint64(cap(es.integerFrames))*uint64(reflect.TypeOf((*polynomialEvalIntegerFrame)(nil)).Size()) + uint64(len(es.integerFrames))*uint64(reflect.TypeOf(polynomialEvalIntegerFrame{}).Size())
	adapterBytes += uint64(cap(es.intervalFrames))*uint64(reflect.TypeOf((*polynomialEvalIntervalFrame)(nil)).Size()) + uint64(len(es.intervalFrames))*uint64(reflect.TypeOf(polynomialEvalIntervalFrame{}).Size())
	for _, f := range es.rationalFrames {
		public = append(public, &f.aux.num, &f.aux.den)
	}
	for _, f := range es.integerFrames {
		public = append(public, &f.aux)
	}
	for _, f := range es.intervalFrames {
		public = append(public, &f.aux.lower.num, &f.aux.upper.num)
	}
	adapterBytes += uint64(cap(full.composition.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.derivative.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.derivative.orphanInvocations)) * 8
	if full.upm != nil {
		u := full.upm
		if len(u.store.backings) != 0 || len(u.pages) != 0 {
			t.Fatal("constructor census requires fresh actual UP member")
		}
		if !u.closed {
			adapterBytes += u.header
		}
		if !u.modular.closed {
			adapterBytes += u.modular.header
		}
		if !u.store.closed {
			adapterBytes += u.store.header
		}
	}
	for i := range full.retirements {
		r := &full.retirements[i]
		adapterBytes += uint64(cap(r.fields)) * uint64(reflect.TypeOf(Int{}).Size())
		if r.consumed {
			for j := r.drain; j < uint32(len(r.fields)); j++ {
				public = append(public, &r.fields[j])
			}
		} else {
			v := [6]*polynomialVector{&m.cheap2.as, &m.cheap.as, &m.som2.as, &m.som.as, &m.linearTerms, &m.rationalTerms}[i]
			for j := uint32(0); j < r.captured; j++ {
				a, e := v.coefficient(m, j)
				serviceMust(t, e)
				if a != r.fields[j] {
					t.Fatal("tentative retirement does not match actual still-owning source field")
				}
			}
		}
	}
	z := m.integer
	refs := map[numericCellRef]bool{}
	add := func(a Int) {
		if a.cell == 0 || z.isFailureRef(a.cell) {
			return
		}
		if !validNumericRef(z.cells, a.cell) {
			t.Fatal("invalid named numeral", a)
		}
		refs[a.cell] = true
	}
	for _, r := range z.tmp {
		add(Int{cell: r})
	}
	for _, r := range z.arg {
		add(Int{cell: r})
	}
	for i := range z.rootRaw.values {
		add(z.rootRaw.values[i])
	}
	if z.matchesDividesNumeralInvocation(&z.dividesNumeral.receipt) {
		add(z.dividesNumeral.numeral)
	}
	for _, s := range []*integerRawScope{&z.divisionRaw, &z.extendedRaw, &z.gcdRaw} {
		for i := uint8(0); i < s.count; i++ {
			add(s.values[i])
		}
	}
	add(z.powerRaw.value)
	for i := range z.perfectSquare.values {
		add(z.perfectSquare.values[i])
	}
	for i := range z.rawDisplay.roots {
		add(z.rawDisplay.roots[i])
	}
	add(z.smt2Display.value)
	if z.matchesAddMulInvocation(&z.gcdRaw.addMul) || z.matchesAddMulInvocation(&z.addMulInvocation.receipt) {
		add(z.addMulInvocation.tmp)
	}
	add(z.intMin)
	add(z.two64)
	for _, a := range public {
		if a != nil {
			add(*a)
		}
	}
	if full.upm != nil && full.upm.modular != nil {
		u := full.upm.modular
		for _, a := range []Int{u.p, u.lower, u.upper, u.invTmp1, u.invTmp2, u.invTmp3, u.divTmp} {
			add(a)
		}
	}
	q := m.config.Rationals
	if q != nil {
		r := &q.rawArithmetic
		public = append(public, &r.div, &r.power.num, &r.power.den, &r.compareOne.num, &r.compareOne.den, &r.compareTwo.num, &r.compareTwo.den, &r.addOne, &r.addTwo, &r.norm)
		for _, a := range public {
			if a != nil {
				add(*a)
			}
		}
		for _, a := range []Int{q.nTmp, q.addTmp1, q.addTmp2, q.addMulTmp.num, q.addMulTmp.den, q.ltTmp1.num, q.ltTmp1.den, q.ltTmp2.num, q.ltTmp2.den} {
			add(a)
		}
	}
	if m.modular != nil {
		for _, a := range []Int{m.modular.p, m.modular.lower, m.modular.upper, m.modular.invTmp1, m.modular.invTmp2, m.modular.invTmp3, m.modular.divTmp, m.zeroNumeral, m.minusOne, m.negativeCoefficient, m.constantCopy, m.linearConstant, m.constantScope.numeral, m.constantScope.rational.num, m.constantScope.rational.den, m.exactDivision.numeral, m.pseudoDivision.numeral, m.dividesLocal.numeral} {
			add(a)
		}
	}
	cell := func(p Polynomial, n uint32) {
		for i := uint32(0); i < n; i++ {
			a, e := m.coefficientCore(p, i)
			serviceMust(t, e)
			add(a)
		}
	}
	for _, p := range []Polynomial{m.zero, m.unit, m.constantScope.result, m.overloadResult, m.negativeResult} {
		if p != (Polynomial{}) && !(p == m.zero && full.bodyStage > 0) && !(p == m.unit && full.bodyStage > 1) {
			l, e := m.polynomialLocation(p)
			serviceMust(t, e)
			if l.released {
				continue
			}
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

	vectors := []*polynomialVector{&m.rationalTerms, &m.linearTerms, &m.linearMonomials, &m.polynomials, &full.skeletons.data}
	adapterBytes += full.skeletons.header + uint64(cap(full.skeletons.children))*uint64(reflect.TypeOf((*PolynomialSkeleton)(nil)).Size())
	seenSkeleton := map[*PolynomialSkeleton]bool{}
	addSkeleton := func(sk *PolynomialSkeleton) {
		if sk == nil || seenSkeleton[sk] {
			return
		}
		seenSkeleton[sk] = true
		adapterBytes += sk.header + uint64(cap(sk.retiredEntries)+cap(sk.retiredOriginals))*uint64(reflect.TypeOf(Monomial{}).Size())
		vectors = append(vectors, &sk.entries, &sk.powers, &sk.originals)
		external = append(external, sk.ms.raw, sk.ms.pending)
	}
	for _, sk := range full.skeletons.children {
		addSkeleton(sk)
	}
	for _, f := range full.gcdFrames {
		if !f.active {
			continue
		}
		addSkeleton(f.newSkeleton)
		if f.sparse.self != nil && !f.sparse.closed {
			vectors = append(vectors, &f.sparse.inputs, &f.sparse.outputs)
			adapterBytes += f.sparse.header
		}
	}

	somRaw, _, somNumerals, somBytes := polynomialSOMNamedRoots(t, m, []*PolynomialSOMVector{&full.iccp})
	external = append(external, somRaw...)
	for _, a := range somNumerals {
		add(a)
	}
	extraBytes := adapterBytes + somBytes
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
	for _, f := range full.gcdFrames {
		if f.active {
			addPolynomial(f.returned.value)
			addPolynomial(f.returned.incoming)
		}
	}
	for _, f := range full.iccpFrames {
		if !f.active {
			continue
		}
		external = append(external, f.filter.raw, f.filter.pending, f.powers.raw, f.powers.pending)
		addPolynomial(f.ci.value)
		addPolynomial(f.ci.incoming)
	}
	addPolynomial(full.squareRoot.freshResult)
	addPolynomial(full.composition.local.value)
	addPolynomial(full.composition.local.incoming)
	addPolynomial(full.composition.freshResult)
	if full.conversion.ownsResult {
		addPolynomial(full.conversion.result)
	}
	if full.toPolynomial.ownsResult {
		addPolynomial(full.toPolynomial.result)
	}
	addPolynomial(m.exactDivision.returned)
	addPolynomial(m.translation.freshResult)
	for i := range m.pseudoDivision.refs {
		addPolynomial(m.pseudoDivision.refs[i].value)
		addPolynomial(m.pseudoDivision.refs[i].incoming)
	}
	if m.powerLocal.active {
		addPolynomial(m.powerLocal.result.value)
		addPolynomial(m.powerLocal.result.incoming)
	}
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
		for _, a := range []Int{n.scope.product, n.scope.aux, n.scope.c} {
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
		vectors = append(vectors, &b.as, &b.ms)
	}
	for _, v := range vectors {
		if v.width != 12 {
			continue
		}
		if v.raw != (memory.Block{}) {
			for i := uint32(0); i < v.initialized; i++ {
				a, e := v.coefficient(m, i)
				serviceMust(t, e)
				add(a)
			}
		}
		if v.old != (memory.Block{}) {
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
	if uint32(len(refs)) != z.cells.Stats().Live {
		t.Fatal("unnamed actual numeric descriptor", len(refs), z.cells.Stats().Live)
	}
	for ref := range refs {
		c := z.cell(ref)
		if c.payload != (arena.VectorRef{}) {
			t.Fatal("fake Words cell")
		}
		if !c.released {
			o, e := z.cellObject(ref)
			serviceMust(t, e)
			_, e = z.allocator.ObjectExtent(o)
			serviceMust(t, e)
		}
	}
	if z.pendingCell.object != (memory.ObjectRef{}) {
		_, e := z.allocator.ObjectExtent(z.pendingCell.object)
		serviceMust(t, e)
	}
	raw := map[memory.Block]bool{}
	put := func(b memory.Block) {
		if b == (memory.Block{}) {
			return
		}
		if raw[b] {
			t.Fatal("duplicate named physical root")
		}
		raw[b] = true
	}
	for _, b := range external {
		put(b)
	}
	mm := m.monomials
	if mm != nil {
		for _, r := range []memory.Block{mm.table.raw, mm.table.pending, mm.renameTable.raw, mm.renameTable.pending, mm.renameFound, mm.powers.raw, mm.powers.pending} {
			put(r)
		}
		for _, tmp := range mm.temps {
			put(tmp.raw)
			put(tmp.pending)
		}
		for _, r := range mm.ids.RawRoots() {
			put(r)
		}
	}
	for _, r := range m.ids.RawRoots() {
		put(r)
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
	for _, s := range z.natural.division {
		put(s.raw)
		put(s.pending)
	}
	if z.natural.verificationActive {
		put(z.natural.verification.raw)
		put(z.natural.verification.pending)
	}
	if z.natural.formatting.active || z.natural.formatting.pending {
		for _, s := range z.natural.formatting.locals {
			put(s.raw)
			put(s.pending)
		}
	}
	put(z.natural.charFormatting.buffer.raw)
	put(z.natural.charFormatting.buffer.pending)
	za := z.allocator.Stats()
	ma := memory.AllocatorStats{}
	mmBytes := uint64(0)
	if mm != nil {
		ma = mm.allocator.Stats()
		mmBytes = mm.header + uint64(cap(mm.locations))*uint64(reflect.TypeOf(monomialLocation{}).Size()) + ma.TotalBytes
	}
	for _, a := range supplied {
		if a == nil || (mm != nil && mm.allocator == a) {
			continue
		}
		st := a.Stats()
		ma.Chunks += st.Chunks
		ma.DirectObjects += st.DirectObjects
		ma.PendingBlocks += st.PendingBlocks
		mmBytes += st.TotalBytes
	}
	if uint32(len(raw))+za.Chunks+za.DirectObjects+za.PendingBlocks+ma.Chunks+ma.DirectObjects+ma.PendingBlocks != m.memory.Stats().LiveBlocks {
		t.Fatal("unnamed Runtime root", len(raw), za, ma, m.memory.Stats())
	}
	bytes := uint64(cap(m.linearGenerated)) + integerActualOwnedBytes(z) + m.header + mmBytes + uint64(cap(m.locations))*uint64(reflect.TypeOf(polynomialLocation{}).Size())
	if m.modular != nil && !m.modular.closed {
		bytes += m.modular.header
	}
	if q != nil {
		bytes += q.headerBytes
	}
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		bytes += uint64(cap(b.coefficientOwners)+cap(b.monomialOwners)+cap(b.coefficientTransferred)+cap(b.monomialTransferred)) * 8
	}
	for _, v := range vectors {
		bytes += uint64(cap(v.retired)) * uint64(reflect.TypeOf(Int{}).Size())
	}
	for _, h := range handlers {
		if h == nil || h.self != h || h.budget != z.budget {
			t.Fatal("invalid named handler")
		}
		bytes += h.header
	}
	bytes += extraBytes
	if bytes != z.budget.Used() {
		t.Fatalf("actual charged headers and backings %d/%d full%d core%d mm%d up%d/%d/%d extra%d mmbytes%d runtime%d qa%d monoalloc%d", bytes, z.budget.Used(), full.header, m.header, mm.header, full.upm.header, full.upm.modular.header, full.upm.store.header, extraBytes, mmBytes, m.memory.Stats().TotalBytes, q.headerBytes, ma.TotalBytes)
	}
}

func polynomialImpPhysicalAuditNamedRaw(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*Int, externalHeaders uint64, extraRaw []memory.Block, supplied ...*memory.SmallAllocator) {
	polynomialImpPhysicalAuditNamedExtra(t, full, polys, input, externalHeaders, extraRaw, supplied...)
}
