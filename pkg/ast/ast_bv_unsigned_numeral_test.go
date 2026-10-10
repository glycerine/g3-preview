// Source: src/ast/rewriter/bv_rewriter.h:177; src/util/rational.h:48.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math"
	"testing"
)

func TestASTBitVectorUnsignedNumeralConstructorBoundary(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, n := range []uint32{0, 1, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		got, e := m.BitVectorNumeralUnsigned(n, 65)
		astTestMust(t, e)
		astTestPin(t, m, got)
		expected, e := m.BitVectorNumeralUint64(uint64(n), 65)
		astTestMust(t, e)
		if got != expected {
			t.Fatal("unsigned numeral value", n)
		}
	}
}

func TestASTBitVectorUnsignedConstructorInclusiveBoundaryBeforeWidthGuard(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	m.Budget().FailAfter(0)
	_, body := m.BitVectorNumeralUnsigned(math.MaxInt32, 0)
	m.Budget().FailAfter(-1)
	var source ASTException
	if !errors.As(body, &source) {
		t.Fatal("width guard", body)
	}
	f := m.bitVectorPlugin.numeralFrames
	if f == nil || !f.complete[5] {
		t.Fatal("incomplete inline unsigned constructor")
	}
	constructor, _, _ := f.children[9].Causes()
	if constructor != nil {
		t.Fatal("unsigned INT_MAX allocated", constructor)
	}
}

func TestASTBitVectorUnsignedNumeralEveryNumericalReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			m.Budget().FailAfter(ordinal)
			h, body := m.BitVectorNumeralUnsigned(math.MaxUint32, 65)
			m.Budget().FailAfter(-1)
			if body == nil {
				success = true
				if h.IsNull() || m.bitVectorPlugin.numeralFrames != nil {
					t.Fatal("constructor did not retire")
				}
			} else {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f umemory.Failure
					if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
						t.Fatal("unexpected numerical refusal", body)
					}
				}
			}
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("constructor reservation horizon", success, failures)
	}
}
