// Source: src/ast/bv_decl_plugin.h:387; src/util/rational.h:65.
package ast

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestASTBitVectorOriginalUint64Numeral(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)) }()
	for _, width := range []uint32{1, 8, 32, 64, 65, 257} {
		for _, value := range []uint64{0, 1, math.MaxInt32 - 1, math.MaxInt32, math.MaxUint32, math.MaxUint64} {
			got, e := m.BitVectorNumeralUint64(value, width)
			astTestMust(t, e)
			astTestPin(t, m, got)
			astTestMust(t, m.numerals.SetUint64(&n, value))
			want, e := m.BitVectorNumeral(&n, width)
			astTestMust(t, e)
			if got != want {
				t.Fatal("uint64 overload identity", value, width)
			}
			var bits uint32
			found, e := m.BitVectorNumeralValue(got, &n, &bits)
			astTestMust(t, e)
			text, e := m.numerals.String(&n)
			astTestMust(t, e)
			oracle := new(big.Int).Mod(new(big.Int).SetUint64(value), new(big.Int).Lsh(big.NewInt(1), uint(width)))
			if !found || bits != width || text != oracle.String() {
				t.Fatal(value, width, found, bits, text, oracle)
			}
		}
	}
}

func TestASTBitVectorUint64ConstructorStrictBoundaryBeforeWidthGuard(t *testing.T) {
	for _, value := range []uint64{math.MaxInt32 - 1, math.MaxInt32} {
		m := bvOffsetRuntimeTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		m.Budget().FailAfter(0)
		_, body := m.BitVectorNumeralUint64(value, 0)
		m.Budget().FailAfter(-1)
		f := m.bitVectorPlugin.numeralFrames
		if f == nil {
			t.Fatal("missing constructor owner")
		}
		constructor, _, _ := f.children[9].Causes()
		if value < math.MaxInt32 {
			if constructor != nil || body == nil || body.Error() != "bit-vector size must be greater than zero" {
				t.Fatal("inline constructor must precede inner width guard", constructor, body)
			}
		} else {
			var failure umemory.Failure
			if constructor == nil || !errors.As(body, &failure) || failure.Kind != umemory.FailureMemout || f.complete[5] {
				t.Fatal("uint64 INT_MAX constructor bypassed big allocation", constructor, body)
			}
		}
	}
}

func TestASTBitVectorUint64NumeralEveryNumericalReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		success := false
		func() {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			m.Budget().FailAfter(ordinal)
			h, body := m.BitVectorNumeralUint64(math.MaxUint64, 65)
			m.Budget().FailAfter(-1)
			if body == nil {
				if h.IsNull() || m.bitVectorPlugin.numeralFrames != nil {
					t.Fatal("constructor did not retire", ordinal, h)
				}
				success = true
			} else if !errors.Is(body, arena.ErrInjectedFailure) {
				var f umemory.Failure
				if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
					t.Fatal(ordinal, body)
				}
			}
		}()
		if success {
			if ordinal == 0 {
				t.Fatal("missing constructor allocations")
			}
			t.Log("first complete reservation", ordinal)
			return
		}
	}
	t.Fatal("uint64 constructor reservation horizon exhausted")
}
