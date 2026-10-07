// Source: src/ast/fpa_decl_plugin.cpp (utility MPF formatting integration).
package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTCheckpointMPFFormatTextActualNumericalInvocationResult(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, c := range []struct {
		sig  int32
		exp  int64
		sign bool
		want string
	}{
		{0, 0, false, "1.0 0"}, {1, 0, true, "-1.00000011920928955078125 0"},
		{1, -127, false, "0.00000011920928955078125 -126"},
	} {
		n := util.NewInt(c.sig)
		var a util.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&a, 8, 24, c.sign, c.exp, &n))
		text, e := util.NewArbitraryFloatFormatText(m.BudgetField)
		astTestMust(t, e)
		result, e := p.numeric.ToStringInto(&a, text)
		astTestMust(t, e)
		length, e := result.Length()
		astTestMust(t, e)
		got := make([]byte, int(length))
		for i := range got {
			got[i], e = result.Byte(uint64(i))
			astTestMust(t, e)
		}
		if string(got) != c.want {
			t.Fatal("literal finite source body", string(got), c.want)
		}
		if text.Numeric.SourceReady() != nil {
			t.Fatal("source epilogue incomplete")
		}
		astTestMust(t, text.SourceFinalize())
		if _, e = result.Byte(0); e == nil {
			t.Fatal("return c_str retained after literal DT")
		}
		astTestMust(t, text.Close())
		astTestMust(t, p.numeric.Del(&a))
	}
	astTestMust(t, m.Close())
}

func TestASTCheckpointMPFFormatTextActualNumericalEveryReservationPrefix(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterFloatingPointPlugin()
		astTestMust(t, e)
		p := m.floatingPointPlugin
		n := util.NewInt(1)
		var a util.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&a, 8, 260, false, 0, &n))
		baseline := m.BudgetField.Used() - m.Stats().AllocatedBytes - m.numerals.RuntimeOwner().Stats().TotalBytes
		text, e := util.NewArbitraryFloatFormatText(m.BudgetField)
		astTestMust(t, e)
		m.BudgetField.FailAfter(ordinal)
		_, call := p.numeric.ToStringInto(&a, text)
		t.Logf("actual prefix %d body=%v", ordinal, call)
		astFloatingPointFormatRawGraph(t, m.numerals, p, text)
		owned := text.Header + uint64(len(text.Result.External)+len(text.Result.Pending)+len(text.Temporary.External)+len(text.Temporary.Pending))
		if text.Stream.NumericTextStream != nil {
			stats := text.Stream.NumericTextStream.Stats()
			if !text.Stream.ClosedField {
				owned += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
			}
		}
		want := baseline + m.Stats().AllocatedBytes + m.numerals.RuntimeOwner().Stats().TotalBytes + owned
		if m.BudgetField.Used() != want {
			t.Fatal("before-lift actual NUM/text cold graph", ordinal, call, m.BudgetField.Used(), want)
		}
		if call == nil {
			complete = true
		}
		m.BudgetField.FailAfter(-1)
		if closeErr := text.Close(); closeErr != nil {
			t.Fatalf("ordinal=%d body=%v close=%v", ordinal, call, closeErr)
		}
		astTestMust(t, p.numeric.Del(&a))
		if sourceErr := m.SourceFinalize(); sourceErr != nil {
			if !errors.Is(sourceErr, arena.ErrInvalidSlot) {
				t.Fatal("actual enclosing source stop", ordinal, sourceErr)
			}
			z := testFieldBySourceName(reflect.ValueOf(p.numeric).Elem(), "integer").Elem()
			stage := testFieldBySourceName(z, "sourceFinalizeStage").Uint()
			if !testFieldBySourceName(z, "sourceGuard").Bool() || stage < 2 || stage > 5 {
				t.Fatal("unproved source scratch-hole domain", ordinal, stage)
			}
			field := "tmp"
			if stage%2 == 1 {
				field = "arg"
			}
			if testFieldBySourceName(z, field).Index(int((stage-2)/2)).Uint() != 0 {
				t.Fatal("source did not reach actual NULL scratch", ordinal, stage)
			}
		}
		if closeErr := m.Close(); closeErr != nil {
			t.Fatalf("parent ordinal=%d body=%v close=%v numericSource=%v", ordinal, call, closeErr, text.Numeric.SourceCause())
		}
		if complete {
			t.Log("first complete real numerical/text prefix", ordinal)
			break
		}
	}
	if !complete {
		t.Fatal("no actual composite numerical/text success")
	}
}
