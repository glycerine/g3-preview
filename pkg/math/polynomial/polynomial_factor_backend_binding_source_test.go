// Source: src/math/polynomial/polynomial.cpp:6683-6725;
// src/util/prime_generator.cpp:99-107 (explicit borrowed Go binding).
package polynomial

import (
	"testing"

	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestPolynomialFactorBackendActualBorrowedBinding(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	if full.factorService != nil {
		t.Fatal("backend must be lazy")
	}
	defaultPrimes, e := primegen.BorrowProcessGenerator()
	serviceMust(t, e)
	service, e := full.factorizationService()
	serviceMust(t, e)
	if service.config.Primes != defaultPrimes || service != full.factorService {
		t.Fatal("default backend must borrow the process generator")
	}
	serviceMust(t, service.Close())
	full.factorService = nil
	primes, e := primegen.NewGenerator(full.core.integer.Budget())
	serviceMust(t, e)
	full.factorConfig = UPolynomialFactorizationConfig{Primes: primes}
	service, e = full.factorizationService()
	serviceMust(t, e)
	if service.config.Primes != primes || service.manager != full.upm || service != full.factorService {
		t.Fatal("actual borrowed pointer and actual UP receiver")
	}
	used := full.core.integer.Budget().Used()
	again, e := full.factorizationService()
	serviceMust(t, e)
	if again != service || used != full.core.integer.Budget().Used() {
		t.Fatal("must reuse actual lazy service")
	}
	serviceMust(t, service.Close())
	full.factorService = nil
	serviceMust(t, primes.Close())
}

func TestPolynomialFactorBackendConfigBorrowsUntilActualOwnerEnd(t *testing.T) {
	q, b := rationalManager(t, false)
	primes, e := primegen.NewGenerator(b)
	serviceMust(t, e)
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}, Factorization: UPolynomialFactorizationConfig{Primes: primes}})
	serviceMust(t, e)
	if full.factorService != nil {
		t.Fatal("construction must not allocate lazy backend")
	}
	service, e := full.factorizationService()
	serviceMust(t, e)
	if service.config.Primes != primes {
		t.Fatal("copied or recreated borrowed generator")
	}
	serviceMust(t, full.CloseChecked())
	if !service.closed || primes.Closed() {
		t.Fatal("actual enclosing owner closes service, not borrowed generator")
	}
	serviceMust(t, primes.Close())
}
