package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanIntAddmulMergedRowsOccurrencesAndQueue(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 6; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	as1 := []numeral.Int{numeral.NewInt(3), numeral.NewInt(5), numeral.NewInt(7)}
	as2 := []numeral.Int{numeral.NewInt(4), numeral.NewInt(-5), numeral.NewInt(6)}
	xs1, xs2 := []uint32{1, 3, 5}, []uint32{0, 3, 4}
	k := numeral.NewInt(1)
	buffer, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	var out []uint32
	s.occurrences[3] = []uint32{7}
	euclidMust(t, s.addmulInt(as1, xs1, &k, as2, xs2, buffer, &out, true, false, 7, nullEquationIndex))
	if !equalUint32Slices(out, []uint32{0, 1, 4, 5}) || !equalUint32Slices(s.occurrences[0], []uint32{7}) ||
		!equalUint32Slices(s.occurrences[4], []uint32{7}) || len(s.occurrences[3]) != 0 {
		t.Fatalf("merged ids or occurrences: %v, %v", out, s.occurrences)
	}
	checkEuclideanIntBuffer(t, z, buffer, []string{"4", "3", "6", "7"})
	// The excluded variable keeps its occurrence while its coefficient cancels.
	s.occurrences[3] = []uint32{7}
	euclidMust(t, s.addmulInt(as1, xs1, &k, as2, xs2, buffer, &out, true, false, 7, 3))
	if !equalUint32Slices(s.occurrences[3], []uint32{7}) {
		t.Fatal("excluded occurrence was removed")
	}
	// The queue variant inserts solved new terms and erases a canceled term.
	s.solved[0], s.solved[3], s.solved[4] = 2, 1, 3
	euclidMust(t, s.queue.Insert(3))
	euclidMust(t, s.addmulInt(as1, xs1, &k, as2, xs2, buffer, &out, false, true, nullEquationIndex, nullEquationIndex))
	if s.queue.Contains(3) || !s.queue.Contains(0) || !s.queue.Contains(4) || !s.queue.CheckInvariant() {
		t.Fatal("source queue updates", s.queue.Contains(0), s.queue.Contains(3), s.queue.Contains(4))
	}
	for _, want := range []int32{0, 4} {
		got, e := s.queue.EraseMin()
		euclidMust(t, e)
		if got != want {
			t.Fatalf("queue minimum = %d, want %d", got, want)
		}
	}
	euclidMust(t, buffer.Close())
	euclidMust(t, s.Close())
	for i := range as1 {
		euclidMust(t, z.Del(&as1[i]))
		euclidMust(t, z.Del(&as2[i]))
	}
	euclidMust(t, z.Del(&k))
}

func checkEuclideanIntBuffer(t *testing.T, m *numeral.IntegerManager, b *numeral.ScopedIntVector, want []string) {
	t.Helper()
	values, e := b.BorrowedValues()
	euclidMust(t, e)
	if len(values) != len(want) {
		t.Fatalf("integer result length = %d, want %d", len(values), len(want))
	}
	for i := range values {
		got, e := m.String(&values[i])
		euclidMust(t, e)
		if got != want[i] {
			t.Fatalf("integer result[%d] = %s, want %s", i, got, want[i])
		}
	}
}

func TestEuclideanRationalAddmulMergedJustifications(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	bs1 := []numeral.Rational{numeral.NewRational(), numeral.NewRational()}
	bs2 := []numeral.Rational{numeral.NewRational(), numeral.NewRational(), numeral.NewRational()}
	for i, value := range []string{"1/2", "2/3"} {
		euclidMust(t, m.SetString(&bs1[i], value))
	}
	for i, value := range []string{"1/5", "-2/3", "3/7"} {
		euclidMust(t, m.SetString(&bs2[i], value))
	}
	buffer, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	var out []uint32
	k := numeral.NewInt(1)
	euclidMust(t, s.addmulRational(bs1, []uint32{1, 3}, &k, bs2, []uint32{0, 3, 4}, buffer, &out))
	if !equalUint32Slices(out, []uint32{0, 1, 4}) {
		t.Fatalf("merged justification IDs = %v", out)
	}
	values, e := buffer.BorrowedValues()
	euclidMust(t, e)
	for i, want := range []string{"1/5", "1/2", "3/7"} {
		got, e := m.String(&values[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("rational result[%d] = %s, want %s", i, got, want)
		}
	}
	euclidMust(t, buffer.Close())
	euclidMust(t, s.Close())
	for i := range bs1 {
		euclidMust(t, m.Del(&bs1[i]))
	}
	for i := range bs2 {
		euclidMust(t, m.Del(&bs2[i]))
	}
	euclidMust(t, m.Integers().Del(&k))
}
