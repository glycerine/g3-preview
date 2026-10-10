package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestUPolynomialFactorEmptyServiceOwnHeaderCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	z, e := unumeral.NewIntegerManager(unumeral.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z, Checkpoint: func() bool { panic("unexpected source checkpoint") }, Cooperation: func() error { panic("unexpected source cooperation") }})
	if e != nil {
		t.Fatal(e)
	}
	primes, e := primegen.NewGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	s, e := NewUPolynomialFactorizationService(m, UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	header := s.Stats().OwnerBytes
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if e = primes.Close(); e != nil {
		t.Fatal(e)
	}
	before := b.Used()
	if e = s.Close(); e != nil {
		t.Fatal("empty service cleanup unnecessarily read closed borrowed owners", e)
	}
	if b.Used() != before-header {
		t.Fatal("actual service header not refunded", before, header, b.Used())
	}
	z.Close()
	if b.Used() != 0 {
		t.Fatal("whole actual owner graph", b.Used())
	}
}
