// Source: src/smt/smt_literal.h:35-103; src/smt/smt_literal.cpp:49-59,86-98;
// src/smt/smt_types.h:33-37; source literal fixtures src/test/diff_logic.cpp:44,61-66.
package smtcore

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSMTLiteralSignedSentinelsAndSourceOperations(t *testing.T) {
	n := DefaultSMTLiteral()
	if n.Index() != -2 || n.Var() != SMTNullBoolVar || n.Sign() || n.Hash() != 0xfffffffe || reflect.TypeOf(n).Size() != 4 {
		t.Fatal("default source literal", n)
	}
	if SMTTrueLiteral.Index() != 0 || SMTFalseLiteral.Index() != 1 || SMTFirstBoolVar != 1 {
		t.Fatal("source constants")
	}
	for _, v := range []int32{0, 1, 2, 3, 4, 5, 6, 63, 64, 1073741823} {
		for _, sign := range []bool{false, true} {
			l := NewSMTLiteral(v, sign)
			want := int64(v) * 2
			if sign {
				want++
			}
			if l.Index() != int32(want) || l.Var() != v || l.Sign() != sign || l.Hash() != uint32(want) {
				t.Fatal(v, sign, l)
			}
			neg := l.Negated()
			if neg.Var() != v || neg.Sign() == sign || neg.Negated() != l {
				t.Fatal("negation", l, neg)
			}
			copy := l
			copy.Negate()
			if copy != neg || l.Index() != int32(want) {
				t.Fatal("in-place versus value negation")
			}
		}
	}
	for _, raw := range []int32{-2147483648, -3, -2, -1, 0, 1, 2147483647} {
		l := SMTLiteralFromIndex(raw)
		if l.Index() != raw || l.Var() != raw>>1 || l.Sign() != (raw&1 != 0) || l.Hash() != uint32(raw) {
			t.Fatal("signed decode", raw, l)
		}
	}
	if !n.Less(SMTTrueLiteral) || !SMTLiteralFromIndex(-1).Less(SMTFalseLiteral) || SMTFalseLiteral.Less(SMTTrueLiteral) {
		t.Fatal("signed order")
	}
}

func TestSMTLiteralBackwardSubsumptionIndependentFiniteMultisets(t *testing.T) {
	// Base-four lists of length <=3 cover duplicates, complements, both constants,
	// every order, and empty spans. Membership oracle uses an independent bit mask.
	var lists [][]SMTLiteral
	for length := 0; length <= 3; length++ {
		total := 1
		for i := 0; i < length; i++ {
			total *= 4
		}
		for code := 0; code < total; code++ {
			v := code
			row := make([]SMTLiteral, length)
			for i := range row {
				row[i] = SMTLiteralFromIndex(int32(v % 4))
				v /= 4
			}
			lists = append(lists, row)
		}
	}
	for _, a := range lists {
		for _, b := range lists {
			var am, bm uint32
			for _, l := range a {
				am |= 1 << uint32(l.Index())
			}
			for _, l := range b {
				bm |= 1 << uint32(l.Index())
			}
			want := am & ^bm == 0
			if got := SMTBackwardSubsumption(a, b); got != want {
				t.Fatal(a, b, got, want)
			}
		}
	}
	if !SMTBackwardSubsumption([]SMTLiteral{DefaultSMTLiteral(), DefaultSMTLiteral()}, []SMTLiteral{DefaultSMTLiteral()}) {
		t.Fatal("source membership is not multiplicity")
	}
}

func TestSMTLiteralSourceStreamSignedNullAndBooleanConstants(t *testing.T) {
	for _, tc := range []struct {
		lit  SMTLiteral
		text string
	}{
		{SMTTrueLiteral, "true"}, {SMTFalseLiteral, "false"}, {DefaultSMTLiteral(), "p-1"}, {DefaultSMTLiteral().Negated(), "(not p-1)"}, {NewSMTLiteral(6, false), "p6"}, {NewSMTLiteral(6, true), "(not p6)"},
	} {
		f, out, b := numericFileFixture(t)
		before := b.Used()
		b.FailAfter(0)
		paramsMust(t, tc.lit.DisplayNumeric(out))
		if b.Used() != before {
			t.Fatal("inline literal acquired storage")
		}
		b.FailAfter(-1)
		_, e := f.Seek(0, io.SeekStart)
		paramsMust(t, e)
		got, e := io.ReadAll(f)
		paramsMust(t, e)
		if string(got) != tc.text {
			t.Fatalf("got %q want %q", got, tc.text)
		}
		paramsMust(t, out.Close())
		if e = tc.lit.DisplayNumeric(out); e == nil {
			t.Fatal("closed stream accepted")
		}
		if e = tc.lit.DisplayNumeric(nil); e != util.ErrNumericStreamOwner {
			t.Fatal("nil stream", e)
		}
	}
}

func TestSMTLiteralSeparateInsertionFlagsAndEveryOutputReservation(t *testing.T) {
	f, out, _ := numericFileFixture(t)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, out.SetWidth(8))
	paramsMust(t, out.SetFill('_'))
	paramsMust(t, NewSMTLiteral(6, true).DisplayNumeric(out))
	_, e := f.Seek(0, io.SeekStart)
	paramsMust(t, e)
	text, e := io.ReadAll(f)
	paramsMust(t, e)
	if string(text) != "__(not p+6)" || out.State().Width != 0 {
		t.Fatal("literal insertion boundaries", string(text), out.State())
	}
	paramsMust(t, out.Close())
	success := false
	for prefix := int64(0); prefix < 32; prefix++ {
		b := arena.NewBudget(^uint64(0))
		stream, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		l := NewSMTLiteral(1073741823, true)
		saved := l
		b.FailAfter(prefix)
		cause := l.DisplayNumeric(stream)
		stats := stream.Stats()
		if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes || l != saved {
			t.Fatal("immediate cold/value graph", prefix, stats, b.Used())
		}
		got, e := stream.String()
		paramsMust(t, e)
		if !strings.HasPrefix("(not p1073741823)", got) {
			t.Fatal("source insertion prefix", prefix, got, cause)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual output refusal", cause)
		}
		b.FailAfter(-1)
		paramsMust(t, stream.Close())
		if b.Used() != 0 {
			t.Fatal("stream cold refund", b.Used())
		}
		if cause == nil {
			if got != "(not p1073741823)" {
				t.Fatal(got)
			}
			success = true
			break
		}
	}
	if !success {
		t.Fatal("no completed actual output horizon")
	}
}
