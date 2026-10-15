package euclid

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanAddMulTraceConcreteIntegerAndRationalCancellation(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	_, err = s.mkVar(false)
	euclidMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver_add_mul")
	s.trace = tags
	first, second, k := numeral.NewInt(5), numeral.NewInt(-5), numeral.NewInt(1)
	ints, err := numeral.NewScopedIntVector(z)
	euclidMust(t, err)
	var xs []uint32
	euclidMust(t, s.addmulInt([]numeral.Int{first}, []uint32{0}, &k,
		[]numeral.Int{second}, []uint32{0}, ints, &xs, false, false, nullEquationIndex, nullEquationIndex))
	if len(xs) != 0 || ints.Size() != 0 {
		t.Fatal("integer cancellation changed", xs, ints.Size())
	}
	q1, q2 := numeral.NewRational(), numeral.NewRational()
	euclidMust(t, m.SetString(&q1, "1/2"))
	euclidMust(t, m.SetString(&q2, "-1/2"))
	rats, err := numeral.NewScopedRationalVector(m)
	euclidMust(t, err)
	var js []uint32
	euclidMust(t, s.addmulRational([]numeral.Rational{q1}, []uint32{0}, &k,
		[]numeral.Rational{q2}, []uint32{0}, rats, &js))
	if len(js) != 0 || rats.Size() != 0 {
		t.Fatal("rational cancellation changed", js, rats.Size())
	}
	if tags.ShouldTrace("euclidean_solver_add_mul") {
		want := "-------- [euclidean_solver_add_mul] addmul src/math/euclid/euclidean_solver.cpp:404 ---------\n" +
			"i1: 0, i2: 0 new_a: 0\nas1: 5, k: 1, as2: -5\n------------------------------------------------\n" +
			"-------- [euclidean_solver_add_mul] addmul src/math/euclid/euclidean_solver.cpp:404 ---------\n" +
			"i1: 0, i2: 0 new_a: 0\nas1: 1/2, k: 1, as2: -1/2\n------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("typed addmul source TRACE records: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased addmul TRACE emitted", output.String())
	}
	refusing := diagnostic.NewTraceTagsWithWriter(refusedEuclideanTraceWriter{})
	refusing.Enable("euclidean_solver_add_mul")
	s.trace = refusing
	euclidMust(t, s.addmulInt([]numeral.Int{first}, []uint32{0}, &k,
		[]numeral.Int{second}, []uint32{0}, ints, &xs, false, false, nullEquationIndex, nullEquationIndex))
	if len(xs) != 0 {
		t.Fatal("refused trace changed integer cancellation")
	}
	euclidMust(t, ints.Close())
	euclidMust(t, rats.Close())
	euclidMust(t, s.Close())
	for _, value := range []*numeral.Int{&first, &second, &k} {
		euclidMust(t, z.Del(value))
	}
	euclidMust(t, m.Del(&q1))
	euclidMust(t, m.Del(&q2))
}
