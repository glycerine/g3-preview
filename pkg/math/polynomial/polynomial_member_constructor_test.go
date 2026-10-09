// Source: src/math/polynomial/polynomial.cpp:193,507-544,1821-1850,2308-2344.
package polynomial

import (
	"testing"
)

func TestPolynomialMemberPreparationBeforeBody(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	defer func() { serviceMust(t, mm.DecManagerRef()) }()
	refs := mm.managerRefs
	c := MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }}
	m, e := preparePolynomialCore(c, true)
	serviceMust(t, e)
	if m.modular == nil || m.directoryInitialized || m.idsInitialized || m.monomialRetained || m.completed || m.zero != (Polynomial{}) || m.unit != (Polynomial{}) || mm.managerRefs != refs {
		t.Fatal("preparation executed source body")
	}
	serviceMust(t, m.constructPolynomialCoreDefaults())
	if !m.directoryInitialized || !m.idsInitialized || m.monomialRetained || mm.managerRefs != refs {
		t.Fatal("default members executed body")
	}
	serviceMust(t, m.initializePolynomialCoreBody())
	if !m.completed || !m.monomialRetained || mm.managerRefs != refs+1 {
		t.Fatal("source body retain/init missing")
	}
	serviceMust(t, m.CloseChecked())
	if mm.managerRefs != refs {
		t.Fatal("borrowed monomial retain not released")
	}
}
