package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// A reservation can fail after earlier coefficients have already been built.
// Every producer must return without losing those owned temporary values.
func TestRealClosureRationalPolynomialProducerReservationPrefixes(t *testing.T) {
	for _, producer := range []struct {
		name string
		run  func(*realValueStore, []*realValue) ([]*realValue, error)
	}{
		{"derivative", func(s *realValueStore, p []*realValue) ([]*realValue, error) {
			return s.derivativeRationalPolynomial(p)
		}},
		{"product", func(s *realValueStore, p []*realValue) ([]*realValue, error) {
			return s.multiplyRationalPolynomials(p, p)
		}},
		{"remainder", func(s *realValueStore, p []*realValue) ([]*realValue, error) {
			return s.signedRemainderRational(p, p[:2])
		}},
		{"pseudo-remainder", func(s *realValueStore, p []*realValue) ([]*realValue, error) {
			return s.signedPseudoRemainderRational(p, p[:2])
		}},
	} {
		t.Run(producer.name, func(t *testing.T) {
			failures, successes := 0, 0
			var lastErr error
			for prefix := int64(0); prefix < 320; prefix++ {
				budget := arena.NewBudget(1 << 24)
				service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
					Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
				}})
				if err != nil {
					t.Fatal(err)
				}
				store := &realValueStore{numbers: service.Manager}
				p := make([]*realValue, 3)
				for i := range p {
					p[i], err = store.newRational(int64(i+2), 1)
					if err != nil {
						t.Fatal(err)
					}
					values := []string{
						"184467440737095516170000000000000000000000000000000000000000000000000000000000000000000000001",
						"184467440737095516190000000000000000000000000000000000000000000000000000000000000000000000003",
						"184467440737095516210000000000000000000000000000000000000000000000000000000000000000000000007",
					}
					if producer.name == "remainder" {
						values = []string{"18446744073709551617", "18446744073709551619", "18446744073709551621"}
					}
					if err := store.numbers.SetString(&p[i].rational, values[i]); err != nil {
						t.Fatal(err)
					}
				}
				budget.FailAfter(prefix)
				out, opErr := producer.run(store, p)
				budget.FailAfter(-1)
				if opErr != nil {
					failures++
					lastErr = opErr
				} else {
					successes++
				}
				if opErr != nil && len(out) != 0 {
					t.Fatalf("prefix %d: failed producer returned coefficients", prefix)
				}
				if err := store.releaseSturmCoefficients(out); err != nil {
					t.Fatalf("prefix %d: release result: %v", prefix, err)
				}
				if err := store.cleanupPendingPolynomialSequence(); err != nil {
					t.Fatalf("prefix %d: cleanup pending: %v", prefix, err)
				}
				if err := store.releaseCoefficients(p); err != nil {
					t.Fatalf("prefix %d: release inputs: %v", prefix, err)
				}
				if err := service.Close(); err != nil {
					t.Fatalf("prefix %d: close service: %v", prefix, err)
				}
				if used := budget.Used(); used != 0 {
					t.Fatalf("prefix %d: retained %d physical bytes after %v", prefix, used, opErr)
				}
				if opErr == nil {
					break
				}
			}
			if failures == 0 || successes == 0 {
				t.Fatalf("reservation sweep did not cover both failed and complete producers: %d failures, %d successes; last error: %v", failures, successes, lastErr)
			}
		})
	}
}
