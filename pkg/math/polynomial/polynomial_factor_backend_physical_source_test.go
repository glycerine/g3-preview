// Source: src/math/polynomial/polynomial.cpp:6683-6725;
// src/math/polynomial/upolynomial_factorization.cpp:1019-1299.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Enumerate actual caller/callee members. Backing directories supply only
// physical allocated extents; they never discover numerical owner authority.
func polynomialFactorBackendNamedRoots(t *testing.T, full *MultivariatePolynomialManager) (roots []*util.Int, bytes uint64) {
	t.Helper()
	seenManager := map[*UPolynomialManager]bool{}
	seenStore := map[*UPolynomialCoefficientStore]bool{}
	seenFactors := map[*UPolynomialFactors]bool{}
	var addManager func(*UPolynomialManager)
	vector := func(m *UPolynomialManager, v *upolyVectorState) {
		if m == nil || v == nil || v.backing == 0 {
			return
		}
		values := m.values(v)
		if v.initialized > uint32(len(values)) {
			t.Fatal("actual vector initialized extent", v)
		}
		for i := uint32(0); i < v.initialized; i++ {
			roots = append(roots, &values[i])
		}
	}
	addStore := func(store *UPolynomialCoefficientStore) {
		if store == nil || seenStore[store] {
			return
		}
		seenStore[store] = true
		if !store.closed {
			bytes += store.header
		}
		bytes += uint64(cap(store.backings)) * uint64(reflect.TypeOf(upolyBacking{}).Size())
		for _, backing := range store.backings {
			if backing.live {
				bytes += uint64(cap(backing.values)) * uint64(reflect.TypeOf(util.Int{}).Size())
			}
		}
	}
	addManager = func(m *UPolynomialManager) {
		if m == nil || seenManager[m] {
			return
		}
		seenManager[m] = true
		if m.live != 0 {
			t.Fatal("actual UP descriptor owner must be named", m.live)
		}
		if !m.closed {
			bytes += m.header
		}
		if !m.modular.Closed() {
			bytes += m.modular.Header
		}
		roots = append(roots, &(*m.modular.BorrowModulus()), &m.modular.Lower, &m.modular.Upper, &m.modular.InvTmp1, &m.modular.InvTmp2, &m.modular.InvTmp3, &m.modular.DivTmp)
		bytes += uint64(cap(m.pages))*uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()) + uint64(len(m.pages))*uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()) + uint64(cap(m.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size())
		addStore(m.store)
		for _, b := range m.bindings {
			addStore(b.store)
		}
		for _, v := range []*upolyVectorState{&m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push, &m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw} {
			vector(m, v)
		}
	}
	factors := func(fs *UPolynomialFactors) {
		if fs == nil || seenFactors[fs] {
			return
		}
		seenFactors[fs] = true
		addManager(fs.manager)
		if !fs.closed {
			bytes += fs.header
		}
		bytes += uint64(cap(fs.factors))*uint64(reflect.TypeOf(upolyVectorState{}).Size()) + uint64(cap(fs.degrees.values))*4
		roots = append(roots, &fs.constant)
		for i := range fs.factors {
			vector(fs.manager, &fs.factors[i])
		}
	}
	addManager(full.upm)
	vector(full.upm, full.factorNumeralVector.output)
	vector(full.upm, &full.factorUnivariate.p1)
	factors(full.factorUnivariate.fs)
	service := full.factorService
	if service == nil {
		return
	}
	if !service.closed {
		bytes += service.header
	}
	roots = append(roots, &service.diagnostic.numeral, &service.preparation.numeral)
	for history := service.squareFreeDiagnosticHistory; history != nil; history = history.next {
		bytes += history.header
	}
	scopes := []*upolyFactorSquareFreeScope{}
	if service.scope != nil {
		scopes = append(scopes, service.scope)
	}
	for c := service.squareFreeHistory; c != nil; c = c.historyNext {
		scopes = append(scopes, c)
	}
	for _, c := range scopes {
		bytes += c.header
		addManager(c.manager)
		addManager(c.zp)
		addManager(c.zpe)
		roots = append(roots, &c.p, &c.bestP, &c.gcdTmp, &c.leading, &c.trialContent, &c.mignotte.norm, &c.mignotte.bound, &c.mignotte.tmp, &c.primeTrial.transport, &c.trialHelper.tmp)
		for _, v := range []*upolyVectorState{&c.fpp, &c.trial, &c.quotient} {
			vector(c.manager, v)
		}
		vector(c.primeTrial.zp, &c.primeTrial.fzp)
		factors(c.best)
		factors(c.lifted)
		factors(c.primeTrial.current)
		bytes += uint64(cap(c.ds.words)+cap(c.primeTrial.degrees.words))*4 + uint64(cap(c.it.enabled)) + uint64(cap(c.it.current))*4
		h := &c.hensel
		addManager(h.related)
		addManager(h.prime)
		addManager(h.lift)
		roots = append(roots, &h.am, &h.initialLC, &h.finalLCInv, &h.transport, &h.checkPower, &h.extendedScoped[0], &h.extendedScoped[1], &h.monicRaw[0], &h.monicRaw[1])
		for _, rows := range [][]upolyVectorState{h.outer[:], h.quadratic[:], h.iteration[:], h.individual[:], h.checkVectors[:], h.quadraticCheck[:], h.extended[:], h.extendedDebug[:]} {
			for i := range rows {
				vector(c.manager, &rows[i])
			}
		}
	}
	for h := service.henselPairs; h != nil; h = h.next {
		bytes += h.header
		addManager(h.rm)
		addManager(h.check.brCore)
		addManager(h.check.bCore)
		roots = append(roots, &h.am, &h.check.br, &h.check.transport)
		for i := uint8(2); i <= 7; i++ {
			m, v := h.vector(i)
			vector(m, v)
		}
		for _, v := range []*upolyVectorState{&h.check.test1, &h.check.test2a, &h.check.test2b, &h.check.test3a, &h.check.test3b} {
			vector(service.manager, v)
		}
	}
	return
}
