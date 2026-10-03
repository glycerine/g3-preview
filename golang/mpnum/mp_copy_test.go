package mpnum

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestIntegerCopyFromSourceStorageContract(t *testing.T) {
	source, _ := integerManager(t)
	dest, budget := integerManager(t)
	var a, b Int
	defer source.Del(&a)
	defer dest.Del(&b)
	// Same-owner delegation preserves the existing source set/copy behavior.
	if e := source.SetUint64(&a, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	if e := source.CopyFrom(&a, source, &a); e != nil {
		t.Fatal(e)
	}
	if e := source.ensureOwnedCapacity(&a, 19); e != nil {
		t.Fatal(e)
	}
	if e := dest.CopyFrom(&b, source, &a); e != nil {
		t.Fatal(e)
	}
	if dest.cell(b.cell).capacity != source.cell(a.cell).capacity || dest.cell(b.cell).size != source.cell(a.cell).size || b.val != a.val {
		t.Fatal("source allocation shape not copied")
	}
	old := b.cell
	if e := source.Neg(&a); e != nil {
		t.Fatal(e)
	}
	budget.FailAfter(0)
	if e := dest.CopyFrom(&b, source, &a); e != nil || b.cell != old {
		t.Fatal("reuse requires no reservation", e)
	}
	budget.FailAfter(-1)
	if integerString(t, dest, &b) != "-18446744073709551615" {
		t.Fatal("signed copy")
	}
	// Unsigned source set(INT_MAX) selects a big one-limb representation.
	if e := source.SetUint64(&a, math.MaxInt32); e != nil {
		t.Fatal(e)
	}
	if e := dest.CopyFrom(&b, source, &a); e != nil || b.IsSmall() {
		t.Fatal("big representation normalized", e)
	}
	for _, v := range []int32{math.MinInt32, -1, 0, 1, math.MaxInt32} {
		x := NewInt(v)
		if e := dest.CopyFrom(&b, source, &x); e != nil || !b.IsSmall() || b.val != v {
			t.Fatal(v, b, e)
		}
	}
}

func TestIntegerCopyFromIndependentOwnersAndSourceClose(t *testing.T) {
	source, sourceBudget := integerManager(t)
	dest, _ := integerManager(t)
	rng := rand.New(rand.NewSource(20071))
	values := make([]Int, 80)
	want := make([]string, len(values))
	for i := range values {
		z := new(big.Int).SetUint64(rng.Uint64())
		z.Lsh(z, uint(i%131))
		z.Add(z, new(big.Int).SetUint64(rng.Uint64()))
		if i%2 != 0 {
			z.Neg(z)
		}
		want[i] = z.String()
		var a Int
		setIntegerOracle(t, source, &a, z)
		before := sourceBudget.Used()
		if e := dest.CopyFrom(&values[i], source, &a); e != nil {
			t.Fatal(e)
		}
		if sourceBudget.Used() != before {
			t.Fatal("source storage changed")
		}
		if e := source.Del(&a); e != nil {
			t.Fatal(e)
		}
	}
	source.Close()
	if sourceBudget.Used() != 0 {
		t.Fatal(sourceBudget.Used())
	}
	for i := range values {
		if got := integerString(t, dest, &values[i]); got != want[i] {
			t.Fatal(i, got, want[i])
		}
		if e := dest.Del(&values[i]); e != nil {
			t.Fatal(e)
		}
	}
}

func TestRationalCopyFromRawStatesAndIndependentLifetime(t *testing.T) {
	for _, synch := range []bool{false, true} {
		source, sourceBudget := rationalManager(t, synch)
		dest, _ := rationalManager(t, !synch)
		var values []Rational
		var expected []string
		for _, parts := range [][2]string{{"2", "4"}, {"2", "-3"}, {"-1", "-1"}, {"0", "5"}, {"-18446744073709551615", "-79228162514264337593543950337"}} {
			a, b := NewRational(), NewRational()
			if e := source.z.SetString(&a.num, parts[0]); e != nil {
				t.Fatal(e)
			}
			if e := source.z.SetString(&a.den, parts[1]); e != nil {
				t.Fatal(e)
			}
			before := a
			if e := dest.CopyFrom(&b, source, &a); e != nil {
				t.Fatal(e)
			}
			if a != before || a.num.val != b.num.val || a.den.val != b.den.val || a.IsSmall() != b.IsSmall() {
				t.Fatal("copy changed raw representation")
			}
			values = append(values, b)
			expected = append(expected, parts[0]+"/"+parts[1])
			if e := source.Del(&a); e != nil {
				t.Fatal(e)
			}
		}
		source.Close()
		if sourceBudget.Used() != 0 {
			t.Fatal(sourceBudget.Used())
		}
		for i := range values {
			if got := rationalString(t, dest, &values[i]); got != expected[i] {
				t.Fatal(i, got, expected[i])
			}
			if e := dest.Del(&values[i]); e != nil {
				t.Fatal(e)
			}
		}
	}
}

func TestNumeralCopyFromFailurePrefixesAndImmediateOwnership(t *testing.T) {
	for _, rational := range []bool{false, true} {
		for _, bigOutput := range []bool{false, true} {
			failed, passed := false, false
			for prefix := int64(0); prefix < 80; prefix++ {
				source, sourceBudget := rationalManager(t, true)
				dest, budget := rationalManager(t, false)
				a, b := NewRational(), NewRational()
				n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 1279), big.NewInt(29))
				d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 959), big.NewInt(11))
				setIntegerOracle(t, source.z, &a.num, n)
				setIntegerOracle(t, source.z, &a.den, d)
				if bigOutput {
					if e := dest.SetFraction(&b, math.MaxInt64, math.MaxUint64); e != nil {
						t.Fatal(e)
					}
				} else {
					if e := dest.SetInt64(&b, 17); e != nil {
						t.Fatal(e)
					}
				}
				// Occupy the actual free cell pool so every new allocation is observable.
				var fillers []Int
				for dest.z.cells.Stats().Live+dest.z.cells.Stats().Retired < dest.z.cells.Stats().Slots {
					var x Int
					if e := dest.z.SetUint64(&x, math.MaxUint64); e != nil {
						t.Fatal(e)
					}
					fillers = append(fillers, x)
				}
				beforeA, beforeB := a, b
				cells, words := dest.z.cells.Stats().Live, dest.z.words.Stats().Live
				sourceCells, sourceWords, sourceUsed := source.z.cells.Stats().Live, source.z.words.Stats().Live, sourceBudget.Used()
				budget.FailAfter(prefix)
				var e error
				if rational {
					e = dest.CopyFrom(&b, source, &a)
				} else {
					e = dest.z.CopyFrom(&b.num, source.z, &a.num)
				}
				if a != beforeA || source.z.cells.Stats().Live != sourceCells || source.z.words.Stats().Live != sourceWords || sourceBudget.Used() != sourceUsed {
					t.Fatal("source mutation", rational, bigOutput, prefix)
				}
				if e != nil {
					failed = true
					if !errors.Is(e, arena.ErrInjectedFailure) || b != beforeB || dest.z.cells.Stats().Live != cells || dest.z.words.Stats().Live != words {
						t.Fatal("copy failure ownership/publication", rational, bigOutput, prefix, e)
					}
				} else {
					passed = true
				}
				budget.FailAfter(-1)
				if passed {
					if integerString(t, dest.z, &b.num) != n.String() {
						t.Fatal("copied numerator")
					}
					if rational && integerString(t, dest.z, &b.den) != d.String() {
						t.Fatal("copied denominator")
					}
				}
				for i := range fillers {
					dest.z.Del(&fillers[i])
				}
				dest.Del(&b)
				source.Del(&a)
				dest.Close()
				source.Close()
				if budget.Used() != 0 || sourceBudget.Used() != 0 {
					t.Fatal("copy graph leaked")
				}
				if passed {
					break
				}
			}
			if !failed || !passed {
				t.Fatal("missing failure sweep endpoint", rational, bigOutput)
			}
		}
	}
}

func TestNumeralCopyFromWarmGuardsAndSameOwner(t *testing.T) {
	source, _ := integerManager(t)
	dest, budget := integerManager(t)
	var a, b Int
	source.SetUint64(&a, math.MaxUint64)
	dest.CopyFrom(&b, source, &a)
	budget.FailAfter(0)
	if allocations := testing.AllocsPerRun(100, func() {
		if e := dest.CopyFrom(&b, source, &a); e != nil {
			panic(e)
		}
	}); allocations != 0 {
		t.Fatal(allocations)
	}
	budget.FailAfter(-1)
	if e := dest.CopyFrom(&b, nil, &a); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
	if e := dest.CopyFrom(&a, source, &a); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("same raw address across owners", e)
	}
	if e := dest.CopyFrom(nil, source, &a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	source.Del(&a)
	dest.Del(&b)
	source.Close()
	if e := dest.CopyFrom(&b, source, &a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	q, _ := rationalManager(t, true)
	x := NewRational()
	q.SetFraction(&x, 2, 3)
	if e := q.CopyFrom(&x, q, &x); e != nil || rationalString(t, q, &x) != "2/3" {
		t.Fatal(e)
	}
	if e := q.CopyFrom(&x, nil, &x); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
	q.Del(&x)
	qs, _ := rationalManager(t, true)
	qd, qb := rationalManager(t, false)
	qa, qout := NewRational(), NewRational()
	if e := qs.SetFraction(&qa, math.MaxInt64, math.MaxUint64); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		if e := qd.CopyFrom(&qout, qs, &qa); e != nil {
			t.Fatal(e)
		}
	}
	before := qb.Used()
	qb.FailAfter(0)
	if allocations := testing.AllocsPerRun(100, func() {
		if e := qd.CopyFrom(&qout, qs, &qa); e != nil {
			panic(e)
		}
	}); allocations != 0 {
		t.Fatal("rational copy allocations", allocations)
	}
	if qb.Used() != before {
		t.Fatal("warm rational copy budget")
	}
	qb.FailAfter(-1)
	qs.Del(&qa)
	qd.Del(&qout)
	qs.Close()
	if e := qd.CopyFrom(&qout, qs, &qa); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
