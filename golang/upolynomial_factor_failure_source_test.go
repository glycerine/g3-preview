// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestUPolynomialFactorFullAlgorithmImmediateFailureRoots(t *testing.T) {
	failures, successes := 0, 0
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		m, _, b := upolySourceOwner(t)
		primes, e := mpnum.NewPrimeGenerator(b)
		if e != nil {
			t.Fatal(e)
		}
		service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
		if e != nil {
			t.Fatal(e)
		}
		fs, e := m.NewFactors()
		if e != nil {
			t.Fatal(e)
		}
		p := upolySourceNew(t, m, nil)
		input := make([]mpnum.Int, 4)
		// (x-12345678901234567890)(x+1)(x+2), frozen expansion.
		for i, text := range []string{"-24691357802469135780", "-37037036703703703668", "-12345678901234567887", "1"} {
			if e = m.IntegerManager().SetString(&input[i], text); e != nil {
				t.Fatal(e)
			}
		}
		if e = m.SetInts(p, input); e != nil {
			t.Fatal(e)
		}
		for i := range input {
			if e = m.IntegerManager().Del(&input[i]); e != nil {
				t.Fatal(e)
			}
		}
		before := m.Stats()
		external := before.IntegerLiveCells - before.OwnedNumeralCells - before.IntegerCachedCells - fs.Stats().OwnedNumeralCells
		b.FailAfter(ordinal)
		_, err := service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams())
		immediate := m.Stats()
		owned := fs.Stats()
		// Inspect the actual live numeral/Words graph BEFORE disabling faults,
		// allocating formatting buffers, deleting results or closing any owner.
		if immediate.IntegerLiveCells != external+immediate.OwnedNumeralCells+immediate.IntegerCachedCells+owned.OwnedNumeralCells || immediate.IntegerWordsRefs != immediate.IntegerLiveCells+immediate.IntegerNaturalWordsRefs {
			t.Fatal("immediate whole graph", ordinal, err, immediate, owned)
		}
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal(ordinal, err)
		}
		b.FailAfter(-1)
		if err == nil {
			successes++
			if n, e := fs.DistinctFactors(); e != nil || n != 3 {
				t.Fatal(n, e)
			}
			out := upolySourceNew(t, m, nil)
			if e = fs.Multiply(out); e != nil {
				t.Fatal(e)
			}
			if same, e := m.Equal(p, out); e != nil || !same {
				t.Fatal(same, e)
			}
		} else {
			failures++
		}
		if e = fs.Close(); e != nil {
			t.Fatal(e)
		}
		if e = service.Close(); e != nil {
			t.Fatal(e)
		}
		if e = primes.Close(); e != nil {
			t.Fatal(e)
		}
		if err == nil {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal("bounded sweep must include first successful endpoint", failures, successes)
	}
	t.Log("actual refusal ordinals and successful endpoint", failures, successes)
}
func TestUPolynomialFactorCancellationAndConstantPrefix(t *testing.T) {
	m, _, b, limit, _ := upolySourceOwnerServices(t)
	primes, e := mpnum.NewPrimeGenerator(b)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := primes.Close(); e != nil {
			t.Error(e)
		}
	}()
	service, e := mpnum.NewUPolynomialFactorizationService(m, mpnum.UPolynomialFactorizationConfig{Primes: primes})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := service.Close(); e != nil {
			t.Error(e)
		}
	}()
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	old := upolySourceNew(t, m, []int32{1, 1})
	if e = fs.PushCopy(mustUPolyView(t, m, old), 2); e != nil {
		t.Fatal(e)
	}
	zero := upolySourceNew(t, m, nil)
	if _, e = service.Factor(mustUPolyView(t, m, zero), fs, mpnum.DefaultUPolynomialFactorParams()); e != nil {
		t.Fatal(e)
	}
	if n, e := fs.DistinctFactors(); e != nil || n != 1 {
		t.Fatal("zero factor path does not clear old factors", n, e)
	}
	// Actual source resource owner cancellation, not a successful callback stub.
	p := upolySourceNew(t, m, []int32{12, 0, -8, 0, 1})
	limit.cancel()
	before := m.Stats()
	external := before.IntegerLiveCells - before.OwnedNumeralCells - before.IntegerCachedCells - fs.Stats().OwnedNumeralCells
	_, err := service.Factor(mustUPolyView(t, m, p), fs, mpnum.DefaultUPolynomialFactorParams())
	after := m.Stats()
	if !errors.Is(err, mpnum.ErrUPolynomialCanceled) {
		t.Fatal(err)
	}
	if after.IntegerLiveCells != external+after.OwnedNumeralCells+after.IntegerCachedCells+fs.Stats().OwnedNumeralCells {
		t.Fatal("cancellation roots", after, fs.Stats())
	}
}
