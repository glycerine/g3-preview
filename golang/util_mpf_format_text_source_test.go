// Source: src/util/mpf.cpp:1519-1569; selected libstdc++ basic_string.tcc:414-429.
// Source author: Christoph Wintersteiger; library policy is a Go adaptation.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func TestMPFFormatTextActualStreamTemporaryAppendAndLifetime(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	text, e := NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	astTestMust(t, text.AssignLiteral("-"))
	stream, e := text.NewArbitraryFloatFormatStream()
	astTestMust(t, e)
	astTestMust(t, stream.AppendCString("1.25 "))
	astTestMust(t, stream.AppendInt64(-7))
	astTestMust(t, text.AppendStreamString(stream))
	view, e := text.CString()
	astTestMust(t, e)
	length, e := view.Length()
	astTestMust(t, e)
	want := "-1.25 -7"
	if length != uint64(len(want)) {
		t.Fatal("literal res append length", length)
	}
	for i := range want {
		ch, e := view.Byte(uint64(i))
		astTestMust(t, e)
		if ch != want[i] {
			t.Fatal("actual owned bytes", i, ch)
		}
	}
	astTestMust(t, stream.SourceFinalize())
	// Returned res owns its exact bytes after the lexical stream ends.
	ch, e := view.Byte(1)
	astTestMust(t, e)
	if ch != '1' {
		t.Fatal(ch)
	}
	astTestMust(t, text.SourceFinalize())
	if _, e = view.Byte(0); e == nil {
		t.Fatal("borrowed c_str read after actual res lifetime")
	}
	astTestMust(t, text.Close())
	if b.Used() != 0 {
		t.Fatal("actual context end", b.Used())
	}
}

func TestMPFFormatTextEveryTemporaryAndAppendReservationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 16; ordinal++ {
		b := arena.NewBudget(1 << 20)
		text, e := NewArbitraryFloatFormatText(b)
		astTestMust(t, e)
		astTestMust(t, text.AssignLiteral("-"))
		stream, e := text.NewArbitraryFloatFormatStream()
		astTestMust(t, e)
		astTestMust(t, stream.AppendCString("1234567890123456789012345678901234567890"))
		beforeLength := text.result.length
		b.FailAfter(ordinal)
		call := text.AppendStreamString(stream)
		stats := text.stream.NumericTextStream.Stats()
		named := text.header + uint64(len(text.result.external)+len(text.result.pending)+len(text.temporary.external)+len(text.temporary.pending)) + stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
		if b.Used() != named {
			t.Fatal("before-lift exact res/temp/stream cold graph", ordinal, call, b.Used(), named)
		}
		if call != nil && text.result.length != beforeLength {
			t.Fatal("append refusal changed actual res length", ordinal)
		}
		if call == nil {
			completed = true
			if text.result.length != 41 {
				t.Fatal("actual length publication", text.result.length)
			}
		}
		b.FailAfter(-1)
		if call != nil {
			astTestMust(t, text.AppendStreamString(stream))
			if text.result.length != 41 {
				t.Fatal("literal valid retry changed result", ordinal, text.result.length)
			}
		}
		astTestMust(t, text.Close())
		if b.Used() != 0 {
			t.Fatal("exact lexical/own end", ordinal, b.Used())
		}
		if completed {
			t.Log("first successful actual text frontier", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("no successful actual text append")
	}
}

func TestMPFFormatTextLengthCopyPreservesEmbeddedNUL(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	text, e := NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	stream, e := text.NewArbitraryFloatFormatStream()
	astTestMust(t, e)
	// Real put insertion writes NUL as a character; ss.str retains its length.
	s := text.stream.NumericTextStream
	astTestMust(t, s.AppendCString("ab"))
	astTestMust(t, s.AppendByte(0))
	astTestMust(t, s.AppendCString("cd"))
	astTestMust(t, text.AppendStreamString(stream))
	if text.result.length != 5 {
		t.Fatal("ss.str incorrectly used strlen", text.result.length)
	}
	view, e := text.CString()
	astTestMust(t, e)
	n, e := view.Length()
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("returned c_str insertion length must use strlen", n)
	}
	astTestMust(t, text.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Source: src/util/mpf.cpp:1519-1569. Direct numeric caller, without AST adapters.
func TestMPFFormatTextActualNumericalInvocationResult(t *testing.T) {
	b, rt, q, m := mpfFormatNumericOwner(t)
	for _, c := range []struct {
		sig  int32
		exp  int64
		sign bool
		want string
	}{
		{0, 0, false, "1.0 0"}, {1, 0, true, "-1.00000011920928955078125 0"},
		{1, -127, false, "0.00000011920928955078125 -126"},
	} {
		n := mpnum.NewInt(c.sig)
		var a mpnum.ArbitraryFloat
		astTestMust(t, m.SetComponents(&a, 8, 24, c.sign, c.exp, &n))
		text, e := NewArbitraryFloatFormatText(b)
		astTestMust(t, e)
		result, e := m.ToStringInto(&a, text)
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
		if text.numeric.SourceReady() != nil {
			t.Fatal("source epilogue incomplete")
		}
		astTestMust(t, text.SourceFinalize())
		if _, e = result.Byte(0); e == nil {
			t.Fatal("return c_str retained after literal DT")
		}
		astTestMust(t, text.Close())
		astTestMust(t, m.Del(&a))
	}
	astTestMust(t, m.Close())
	astTestMust(t, q.Close())
	astTestMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("direct numerical owners retained", b.Used())
	}
}

func TestMPFFormatTextActualNumericalEveryReservationPrefix(t *testing.T) {
	complete := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b, rt, q, m := mpfFormatNumericOwner(t)
		n := mpnum.NewInt(1)
		var a mpnum.ArbitraryFloat
		astTestMust(t, m.SetComponents(&a, 8, 260, false, 0, &n))
		baseline := b.Used() - m.OwnedStorageBytes() - rt.Stats().TotalBytes
		cacheBefore, e := q.Stats()
		astTestMust(t, e)
		text, e := NewArbitraryFloatFormatText(b)
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.ToStringInto(&a, text)
		t.Logf("actual prefix %d body=%v", ordinal, call)
		mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
		owned := text.header + uint64(len(text.result.external)+len(text.result.pending)+len(text.temporary.external)+len(text.temporary.pending))
		if text.stream.NumericTextStream != nil {
			stats := text.stream.NumericTextStream.Stats()
			if !text.stream.closed {
				owned += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
			}
		}
		want := baseline + m.OwnedStorageBytes() + rt.Stats().TotalBytes + owned
		if b.Used() != want {
			t.Fatal("before-lift actual NUM/text cold graph", ordinal, call, b.Used(), want)
		}
		if call == nil {
			complete = true
		}
		b.FailAfter(-1)
		if closeErr := text.Close(); closeErr != nil {
			t.Fatalf("ordinal=%d body=%v close=%v", ordinal, call, closeErr)
		}
		astTestMust(t, m.Del(&a))
		if sourceErr := m.SourceFinalize(); sourceErr != nil {
			if !errors.Is(sourceErr, arena.ErrInvalidSlot) {
				t.Fatal("actual enclosing source stop", ordinal, sourceErr)
			}
			z := reflect.ValueOf(m).Elem().FieldByName("integer").Elem()
			stage := z.FieldByName("sourceFinalizeStage").Uint()
			if !z.FieldByName("sourceGuard").Bool() || stage < 2 || stage > 5 {
				t.Fatal("unproved source scratch-hole domain", ordinal, stage)
			}
			field := "tmp"
			if stage%2 == 1 {
				field = "arg"
			}
			if z.FieldByName(field).Index(int((stage-2)/2)).Uint() != 0 {
				t.Fatal("source did not reach actual NULL scratch", ordinal, stage)
			}
		}
		if closeErr := m.Close(); closeErr != nil {
			t.Fatalf("parent ordinal=%d body=%v close=%v numericSource=%v", ordinal, call, closeErr, text.numeric.SourceCause())
		}
		cacheAfter, e := q.Stats()
		astTestMust(t, e)
		if cacheAfter != cacheBefore {
			t.Fatal("unrelated borrowed service changed", ordinal, cacheBefore, cacheAfter)
		}
		astTestMust(t, q.Close())
		astTestMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual numeric owner refund", ordinal, b.Used())
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

// Source: src/util/mpf.cpp:1519-1569. The returned string belongs to its
// caller; completed numerical lexical scopes do not extend fm's lifetime.
func TestMPFFormatTextCompletedNumericalResultOutlivesManager(t *testing.T) {
	for _, special := range []bool{false, true} {
		t.Run(map[bool]string{false: "finite", true: "erased-zero"}[special], func(t *testing.T) {
			b, rt, q, m := mpfFormatNumericOwner(t)
			n := mpnum.NewInt(1)
			var a mpnum.ArbitraryFloat
			if special {
				astTestMust(t, m.MkZero(8, 24, true, &a))
			} else {
				astTestMust(t, m.SetComponents(&a, 8, 24, true, 0, &n))
			}
			text, e := NewArbitraryFloatFormatText(b)
			astTestMust(t, e)
			if special {
				// Existing caller result capacity erases its actual string allocation.
				astTestMust(t, text.AssignLiteral("-zero"))
				_, e = m.IsNaN(&a) // The literal top-exponent query uses its existing cache.
				astTestMust(t, e)
				b.FailAfter(0)
			}
			view, e := m.ToStringInto(&a, text)
			astTestMust(t, e)
			if reflect.ValueOf(&text.numeric).Elem().FieldByName("sourceCursor").Uint() != 5 {
				t.Fatal("numerical source suffix incomplete")
			}
			mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
			if special {
				b.FailAfter(-1)
			}
			astTestMust(t, m.Del(&a))
			astTestMust(t, m.SourceFinalize())
			astTestMust(t, m.Close())
			ch, e := view.Byte(0)
			astTestMust(t, e)
			if ch != '-' {
				t.Fatal("caller-owned result changed", ch)
			}
			astTestMust(t, text.Close())
			if _, e = view.Byte(0); e == nil {
				t.Fatal("result survived caller end")
			}
			astTestMust(t, q.Close())
			astTestMust(t, rt.Close())
			if b.Used() != 0 {
				t.Fatal("owned text/context remained", b.Used())
			}
		})
	}
}
