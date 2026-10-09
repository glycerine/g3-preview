// Source: src/math/polynomial/polynomial.cpp:1821-1850,2324-2370.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func polynomialImpPhysicalAudit(t *testing.T, full *MultivariatePolynomialManager, supplied ...*util.SmallAllocator) {
	t.Helper()
	polynomialImpPhysicalAuditValues(t, full, nil, supplied...)
}

func polynomialImpPhysicalAuditValues(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, supplied ...*util.SmallAllocator) {
	t.Helper()
	polynomialImpPhysicalAuditNamed(t, full, polys, nil, 0, supplied...)
}
func polynomialImpPhysicalAuditNamed(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*util.Int, externalHeaders uint64, supplied ...*util.SmallAllocator) {
	polynomialImpPhysicalAuditNamedExtra(t, full, polys, input, externalHeaders, nil, supplied...)
}
func polynomialImpPhysicalAuditNamedExtra(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*util.Int, externalHeaders uint64, extraRaw []util.Block, supplied ...*util.SmallAllocator) {
	t.Helper()
	m := full.core
	owners := []*PolynomialNewtonInterpolator{}
	references := []*PolynomialRef{}
	refvectors := []*PolynomialRefVector{}
	lazy := []*PolynomialNewtonVector{&full.newton}
	handlers := []*PolynomialDeletionHandler{}
	external := []util.Block{m.tmp1.raw, m.tmp1.pending, m.substitution.found.raw, m.substitution.found.pending}
	external = append(external, extraRaw...)
	adapterBytes := uint64(cap(m.variableNumeralHistory))*uint64(reflect.TypeOf(polynomialVariableNumeralHistory{}).Size()) + uint64(cap(m.quasiHistory))*uint64(reflect.TypeOf(polynomialQuasiResultantHistory{}).Size()) + full.header
	public := []*util.Int{&full.derivative.n, &full.derivative.a, &full.composition.a}
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		public = append(public, &b.cohortMovingCoefficient, &b.appendScope.movingCoefficient, &b.arithmeticScope.transport)
	}
	if m.transportBound {
		public = append(public, &m.transport)
	}
	if m.scalarDivision.transportBound {
		public = append(public, &m.scalarDivision.transport)
	}
	for _, raw := range []*util.IntegerScalarRawScope{&m.scalarDivision.division, &m.scalarDivision.extended} {
		for i := uint8(0); i < raw.Count; i++ {
			public = append(public, &raw.Values[i])
		}
	}

	if b := &full.conversion.as; b.Constructed {
		for i := uint32(0); i < b.Count(); i++ {
			a, e := b.Physical(i)
			serviceMust(t, e)
			public = append(public, &a)
		}
	}
	adapterBytes += uint64(cap(full.factorUnivariateHistory)) * uint64(reflect.TypeOf(polynomialFactorUnivariateHistory{}).Size())
	adapterBytes += uint64(cap(full.factorNumeralVectorHistory)) * uint64(reflect.TypeOf(polynomialFactorNumeralVectorHistory{}).Size())
	adapterBytes += uint64(cap(full.toPolynomialHistory)) * uint64(reflect.TypeOf(polynomialToPolynomialHistory{}).Size())
	if full.factorNumeralVector.constructed {
		polys = append(polys, full.factorNumeralVector.np.value, full.factorNumeralVector.np.incoming)
	}
	if full.toPolynomial.transportBound {
		public = append(public, &full.toPolynomial.transport)
	}
	if b := &full.toPolynomial.buffer; b.Constructed {
		for i := uint32(0); i < b.Count(); i++ {
			a, e := b.Physical(i)
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
	}
	adapterBytes += uint64(cap(full.iccpFrames)) * uint64(reflect.TypeOf((*polynomialICCPScope)(nil)).Size())
	for _, f := range full.iccpFrames {
		adapterBytes += f.header
	}
	adapterBytes += uint64(cap(m.substitution.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.factorFrames)) * uint64(reflect.TypeOf((*polynomialFactorCoreFrame)(nil)).Size())
	for _, frame := range full.factorFrames {
		adapterBytes += frame.header
		if frame.minimum != nil {
			adapterBytes += frame.minimum.header
		}
		if frame.numeralConstructed {
			public = append(public, &frame.numeral)
		}
		for i := uint8(0); i < frame.references; i++ {
			polys = append(polys, frame.refs[i].value, frame.refs[i].incoming)
		}
	}
	adapterBytes += externalHeaders
	adapterBytes += uint64(cap(m.display.frames))*uint64(reflect.TypeOf((*polynomialDisplayFrame)(nil)).Size()) + uint64(len(m.display.frames))*uint64(reflect.TypeOf(polynomialDisplayFrame{}).Size())
	for _, f := range m.display.frames {
		public = append(public, &f.numeral)
	}
	es := &full.evaluation
	adapterBytes += uint64(cap(es.rationalFrames))*uint64(reflect.TypeOf((*polynomialEvalRationalFrame)(nil)).Size()) + uint64(len(es.rationalFrames))*uint64(reflect.TypeOf(polynomialEvalRationalFrame{}).Size())
	adapterBytes += uint64(cap(es.integerFrames))*uint64(reflect.TypeOf((*polynomialEvalIntegerFrame)(nil)).Size()) + uint64(len(es.integerFrames))*uint64(reflect.TypeOf(polynomialEvalIntegerFrame{}).Size())
	adapterBytes += uint64(cap(es.intervalFrames))*uint64(reflect.TypeOf((*polynomialEvalIntervalFrame)(nil)).Size()) + uint64(len(es.intervalFrames))*uint64(reflect.TypeOf(polynomialEvalIntervalFrame{}).Size())
	for _, f := range es.rationalFrames {
		public = append(public, &(*f.aux.BorrowNumerator()), &(*f.aux.BorrowDenominator()))
	}
	for _, f := range es.integerFrames {
		public = append(public, &f.aux)
	}
	for _, f := range es.intervalFrames {
		public = append(public, &(*(*f.aux.BorrowLower()).BorrowNumerator()), &(*(*f.aux.BorrowUpper()).BorrowNumerator()))
	}
	adapterBytes += uint64(cap(full.composition.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.derivative.orphans)) * uint64(reflect.TypeOf(Monomial{}).Size())
	adapterBytes += uint64(cap(full.derivative.orphanInvocations)) * 8
	backendNumerals, backendBytes := polynomialFactorBackendNamedRoots(t, full)
	public = append(public, backendNumerals...)
	adapterBytes += backendBytes
	if full.factorUnivariate.constructed >= 2 {
		polys = append(polys, full.factorUnivariate.p.value, full.factorUnivariate.p.incoming)
	}
	if full.factorUnivariate.conditionalReference {
		polys = append(polys, full.factorUnivariate.f.value, full.factorUnivariate.f.incoming)
	}

	for i := range full.retirements {
		r := &full.retirements[i]
		adapterBytes += uint64(cap(r.fields)) * uint64(reflect.TypeOf(util.Int{}).Size())
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
	for i := range z.RootRaw.Values {
		add(z.RootRaw.Values[i])
	}
	if z.MatchesDividesNumeralInvocation(&z.DividesNumeral.Receipt) {
		add(z.DividesNumeral.Numeral)
	}
	for _, s := range []*util.IntegerRawScope{&z.DivisionRaw, &z.ExtendedRaw, &z.GcdRaw} {
		for i := uint8(0); i < s.Count; i++ {
			add(s.Values[i])
		}
	}
	add(z.PowerRaw.Value)
	for i := range z.PerfectSquare.Values {
		add(z.PerfectSquare.Values[i])
	}
	for i := range z.RawDisplay.Roots {
		add(z.RawDisplay.Roots[i])
	}
	add(z.Smt2Display.Value)
	if z.MatchesAddMulInvocation(&z.GcdRaw.AddMul) || z.MatchesAddMulInvocation(&z.AddMulInvocation.Receipt) {
		add(z.AddMulInvocation.Tmp)
	}
	add(z.IntMin)
	add(z.Two64)
	for _, a := range public {
		if a != nil {
			add(*a)
		}
	}
	if full.upm != nil && full.upm.modular != nil {
		u := full.upm.modular
		for _, a := range []util.Int{(*u.BorrowModulus()), u.Lower, u.Upper, u.InvTmp1, u.InvTmp2, u.InvTmp3, u.DivTmp} {
			add(a)
		}
	}
	q := m.config.Rationals
	if q != nil {
		r := &q.RawArithmetic
		public = append(public, &r.Div, &r.Power.Num, &r.Power.Den, &r.CompareOne.Num, &r.CompareOne.Den, &r.CompareTwo.Num, &r.CompareTwo.Den, &r.AddOne, &r.AddTwo, &r.Norm)
		for _, a := range public {
			if a != nil {
				add(*a)
			}
		}
		for _, a := range []util.Int{q.NTmp, q.AddTmp1, q.AddTmp2, q.AddMulTmp.Num, q.AddMulTmp.Den, q.LtTmp1.Num, q.LtTmp1.Den, q.LtTmp2.Num, q.LtTmp2.Den} {
			add(a)
		}
	}
	if m.modular != nil {
		for _, a := range []util.Int{(*m.modular.BorrowModulus()), m.modular.Lower, m.modular.Upper, m.modular.InvTmp1, m.modular.InvTmp2, m.modular.InvTmp3, m.modular.DivTmp, m.zeroNumeral, m.minusOne, m.negativeCoefficient, m.constantCopy, m.linearConstant, m.constantScope.numeral, (*m.constantScope.rational.BorrowNumerator()), (*m.constantScope.rational.BorrowDenominator()), m.exactDivision.numeral, m.pseudoDivision.numeral, m.dividesLocal.numeral} {
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

	vectors := []*polynomialVector{&m.rationalTerms, &m.linearTerms}
	adapterBytes += full.skeletons.header
	seenSkeleton := map[*PolynomialSkeleton]bool{}
	addSkeleton := func(sk *PolynomialSkeleton) {
		if sk == nil || seenSkeleton[sk] {
			return
		}
		seenSkeleton[sk] = true
		adapterBytes += sk.header
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
			for _, values := range [][]util.Int{f.sparse.inputs, f.sparse.outputs} {
				for _, a := range values {
					add(a)
				}
			}
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
		count := uint32(len(v.state.values))
		for i := v.state.drain; i < count; i++ {
			p, e := v.state.get(m, i)
			serviceMust(t, e)
			addPolynomial(p)
		}
		addPolynomial(v.state.incoming)
	}
	for _, v := range lazy {
		extraBytes += v.header
		for _, n := range append(append([]*PolynomialNewtonInterpolator(nil), v.children...), v.orphans...) {
			if n != nil {
				owners = append(owners, n)
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
		for _, values := range [][]util.Int{n.inputs, n.invs} {
			for _, a := range values {
				add(a)
			}
		}
		for _, a := range []util.Int{n.scope.product, n.scope.aux, n.scope.c} {
			add(a)
		}
		for _, r := range []*polynomialReference{&n.scope.temp, &n.scope.auxPoly, &n.scope.u} {
			addPolynomial(r.value)
			addPolynomial(r.incoming)
		}
		count := uint32(len(n.vs.values))
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
	for _, b := range external {
		put(b)
	}
	mm := m.monomials
	if mm != nil {
		for _, r := range []util.Block{mm.table.raw, mm.table.pending, mm.renameTable.raw, mm.renameTable.pending, mm.renameFound, mm.powers.raw, mm.powers.pending} {
			put(r)
		}
		for _, tmp := range mm.temps {
			put(tmp.raw)
			put(tmp.pending)
		}
	}
	for _, v := range vectors {
		put(v.raw)
		put(v.pending)
		put(v.old)
	}

	za := z.SourceAllocator().Stats()
	ma := util.AllocatorStats{}
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
	bytes := uint64(m.ids.FreeCapacity())*4 + integerActualOwnedBytes(z) + m.header + mmBytes + uint64(cap(m.locations))*uint64(reflect.TypeOf(polynomialLocation{}).Size())
	if mm != nil {
		bytes += uint64(mm.ids.FreeCapacity()) * 4
	}
	if m.modular != nil && !m.modular.Closed() {
		bytes += m.modular.Header
	}
	if q != nil {
		bytes += q.HeaderBytes
	}
	for _, b := range []*polynomialSOM{&m.som, &m.som2, &m.cheap, &m.cheap2} {
		bytes += uint64(cap(b.coefficientOwners)+cap(b.monomialOwners)+cap(b.coefficientTransferred)+cap(b.monomialTransferred)) * 8
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
		t.Fatalf("actual charged headers and backings %d/%d full%d core%d mm%d up%d/%d/%d extra%d mmbytes%d runtime%d qa%d monoalloc%d", bytes, z.Budget().Used(), full.header, m.header, mm.header, full.upm.header, full.upm.modular.Header, full.upm.store.header, extraBytes, mmBytes, m.memory.Stats().TotalBytes, q.HeaderBytes, ma.TotalBytes)
	}
}

func polynomialImpPhysicalAuditNamedRaw(t *testing.T, full *MultivariatePolynomialManager, polys []Polynomial, input []*util.Int, externalHeaders uint64, extraRaw []util.Block, supplied ...*util.SmallAllocator) {
	polynomialImpPhysicalAuditNamedExtra(t, full, polys, input, externalHeaders, extraRaw, supplied...)
}
