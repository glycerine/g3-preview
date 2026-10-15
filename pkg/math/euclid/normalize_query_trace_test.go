package euclid

import (
	"bytes"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type refusedEuclideanTraceWriter struct{}

func (refusedEuclideanTraceWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestEuclideanNormalizeQueryTraceBeforeAndAfterSubstitution(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, err := NewSolver(m)
	euclidMust(t, err)
	for i := 0; i < 2; i++ {
		_, err = s.MkVar()
		euclidMust(t, err)
	}
	one, eqConstant := numeral.NewInt(1), numeral.NewInt(1)
	euclidMust(t, s.AssertEq([]numeral.Int{one}, []uint32{0}, &eqConstant, nullEquationIndex))
	consistent, err := s.Solve()
	euclidMust(t, err)
	if !consistent {
		t.Fatal("unit fixture is inconsistent")
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("euclidean_solver")
	euclidMust(t, s.BindTrace(tags))
	as := []numeral.Int{numeral.NewInt(3), numeral.NewInt(2)}
	constant, aPrime, cPrime := numeral.NewInt(4), numeral.NewInt(0), numeral.NewInt(0)
	var js []uint32
	euclidMust(t, s.Normalize(as, []uint32{1, 0}, &constant, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "3", "2")
	const footer = "------------------------------------------------\n"
	const before = "-------- [euclidean_solver] normalize src/math/euclid/euclidean_solver.cpp:734 ---------\n" +
		"before applying solution set\n3*x1 2*x0 \n" + footer
	const after = "-------- [euclidean_solver] normalize src/math/euclid/euclidean_solver.cpp:751 ---------\n" +
		"after applying solution set\n3*x1 \n" + footer
	if tags.ShouldTrace("euclidean_solver") {
		if got := output.String(); got != before+after {
			t.Fatalf("source normalize TRACE records: got %q, want %q", got, before+after)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased normalize TRACE emitted", output.String())
	}
	// A failed borrowed stream keeps its own bad state; source normalization
	// still runs and publishes the second query's arithmetic result.
	refusing := diagnostic.NewTraceTagsWithWriter(refusedEuclideanTraceWriter{})
	refusing.Enable("euclidean_solver")
	euclidMust(t, s.BindTrace(refusing))
	euclidMust(t, s.Normalize(as, []uint32{1, 0}, &constant, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "3", "2")
	euclidMust(t, s.BindTrace(tags))
	output.Reset()
	euclidMust(t, s.Reset())
	_, err = s.MkVar()
	euclidMust(t, err)
	euclidMust(t, s.Normalize(as[:1], []uint32{0}, &constant, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "3", "4")
	if tags.ShouldTrace("euclidean_solver") {
		want := "-------- [euclidean_solver] normalize src/math/euclid/euclidean_solver.cpp:734 ---------\n" +
			"before applying solution set\n3*x0 \n" + footer +
			"-------- [euclidean_solver] normalize src/math/euclid/euclidean_solver.cpp:751 ---------\n" +
			"after applying solution set\n3*x0 \n" + footer
		if output.String() != want {
			t.Fatalf("Reset lost borrowed trace binding: got %q, want %q", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased Reset trace emitted", output.String())
	}
	euclidMust(t, s.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	for _, value := range []*numeral.Int{&one, &eqConstant, &constant, &aPrime, &cPrime} {
		euclidMust(t, z.Del(value))
	}
}
