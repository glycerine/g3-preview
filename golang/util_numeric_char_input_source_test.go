// Source: src/util/debug.cpp:73-117, particularly formatted char extraction79-80.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"os"
	"testing"
)

func numericCharInputFixture(t *testing.T, text []byte) (*NumericCharInput, *MemoryStandardStreams, *arena.Budget, *os.File) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	file, e := os.CreateTemp(".", "debug-input-*")
	paramsMust(t, e)
	_, e = file.Write(text)
	paramsMust(t, e)
	_, e = file.Seek(0, 0)
	paramsMust(t, e)
	out, e := os.CreateTemp(".", "debug-input-output-*")
	paramsMust(t, e)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	paramsMust(t, e)
	input, e := NewDebuggerCharInput(streams, file)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, input.Close())
		paramsMust(t, streams.Close())
		file.Close()
		out.Close()
		os.Remove(file.Name())
		os.Remove(out.Name())
		if b.Used() != 0 {
			t.Fatal("input named graph", b.Used())
		}
	})
	return input, streams, b, file
}
func TestNumericCharInputClassicWhitespaceIndependentEveryByte(t *testing.T) {
	for value := 0; value < 256; value++ {
		input, streams, b, _ := numericCharInputFixture(t, []byte{byte(value), 'Q'})
		var dst byte
		before := b.Used()
		b.FailAfter(0)
		ok, e := input.ExtractChar(&dst)
		paramsMust(t, e)
		if !ok {
			t.Fatal(value, "first extraction")
		}
		whitespace := value == 9 || value == 10 || value == 11 || value == 12 || value == 13 || value == 32
		want := byte(value)
		position := uint64(1)
		if whitespace {
			want = 'Q'
			position = 2
		}
		state, e := input.State()
		paramsMust(t, e)
		if dst != want || state.Position != position || state.EOF || state.Fail || state.Bad {
			t.Fatal(value, dst, state)
		}
		if b.Used() != before {
			t.Fatal("extraction allocated")
		}
		numericCharInputPhysical(t, input, streams, b)
		b.FailAfter(-1)
	}
}
func TestNumericCharInputEOFFailureErasesClosedTieAndDestination(t *testing.T) {
	input, streams, _, _ := numericCharInputFixture(t, []byte(" \t\n"))
	dst := byte(0xEE)
	ok, e := input.ExtractChar(&dst)
	paramsMust(t, e)
	if ok || dst != 0xEE {
		t.Fatal("failed extraction altered caller", ok, dst)
	}
	state, e := input.State()
	paramsMust(t, e)
	if !state.EOF || !state.Fail || state.Bad {
		t.Fatal(state)
	}
	out, e := streams.Stdout()
	paramsMust(t, e)
	paramsMust(t, out.Close())
	ok, e = input.ExtractChar(nil)
	paramsMust(t, e)
	if ok {
		t.Fatal("preexisting failed input")
	}
}
func TestNumericCharInputTieFlushBeforeActualFileReadAndClear(t *testing.T) {
	input, streams, _, file := numericCharInputFixture(t, []byte("A"))
	out, e := streams.Stdout()
	paramsMust(t, e)
	paramsMust(t, out.SetWidth(17))
	paramsMust(t, out.Flush())
	if out.State().Width != 17 {
		t.Fatal("unformatted flush consumed width")
	}
	paramsMust(t, out.Close())
	dst := byte(0xEE)
	_, e = input.ExtractChar(&dst)
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("reached closed tie", e)
	}
	pos, e := file.Seek(0, 1)
	paramsMust(t, e)
	if pos != 0 || dst != 0xEE {
		t.Fatal("tie guard after file read", pos, dst)
	}
	old, e := input.SetTie(nil)
	paramsMust(t, e)
	if old != out {
		t.Fatal("actual borrowed tie")
	}
	ok, e := input.ExtractChar(&dst)
	paramsMust(t, e)
	if !ok || dst != 'A' {
		t.Fatal(ok, dst)
	}
	state, e := input.State()
	paramsMust(t, e)
	if state.EOF || state.Fail {
		t.Fatal("last byte not early EOF", state)
	}
	ok, e = input.ExtractChar(&dst)
	paramsMust(t, e)
	if ok || dst != 'A' {
		t.Fatal("EOF destination")
	}
	paramsMust(t, input.Clear())
	_, e = file.Seek(0, 0)
	paramsMust(t, e)
	ok, e = input.ExtractChar(&dst)
	paramsMust(t, e)
	if !ok || dst != 'A' {
		t.Fatal("actual cleared input")
	}
}
func TestNumericCharInputNoSkipAndRealClosedFileBadState(t *testing.T) {
	input, _, _, file := numericCharInputFixture(t, []byte(" A"))
	paramsMust(t, input.SetSkipWhitespace(false))
	dst := byte(0)
	ok, e := input.ExtractChar(&dst)
	paramsMust(t, e)
	if !ok || dst != ' ' {
		t.Fatal("noskipws", ok, dst)
	}
	paramsMust(t, file.Close())
	ok, e = input.ExtractChar(&dst)
	paramsMust(t, e)
	if ok || dst != ' ' {
		t.Fatal("real read failure changed caller")
	}
	state, e := input.State()
	paramsMust(t, e)
	if !state.Bad || !state.Fail || state.EOF {
		t.Fatal(state)
	}
}
func numericCharInputPhysical(t *testing.T, input *NumericCharInput, streams *MemoryStandardStreams, b *arena.Budget) {
	t.Helper()
	var want uint64
	if !input.closed {
		want += input.header
	}
	if !streams.closed {
		want += streams.header
	}
	for _, out := range []*NumericTextStream{streams.stdout, streams.stderr} {
		s := out.Stats()
		want += s.OwnerBytes + s.ScratchBytes + s.OutputOwnerBytes + s.OutputOverflowBytes
	}
	if b.Used() != want {
		t.Fatal("actual input/standard-stream named cold graph", b.Used(), want)
	}
}
func TestNumericCharInputConstructionEveryColdReservationAndForeignCohort(t *testing.T) {
	base, streams, b, file := numericCharInputFixture(t, []byte("A"))
	before := b.Used()
	b.FailAfter(0)
	out, e := NewDebuggerCharInput(streams, file)
	if out != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("actual cold construction refusal", out, e, b.Used(), before)
	}
	b.FailAfter(1)
	out, e = NewDebuggerCharInput(streams, file)
	paramsMust(t, e)
	created := out
	t.Cleanup(func() { paramsMust(t, created.Close()) })
	if out == nil {
		t.Fatal("first successful reservation")
	}
	b.FailAfter(-1)
	paramsMust(t, out.Close())
	numericCharInputPhysical(t, base, streams, b)
	foreign := arena.NewBudget(1 << 20)
	foreignOut, e := NewNumericTextStream(NumericTextStreamConfig{Budget: foreign})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, foreignOut.Close())
		if foreign.Used() != 0 {
			t.Fatal(foreign.Used())
		}
	})
	b.FailAfter(0)
	out, e = NewNumericCharInput(NumericCharInputConfig{Budget: b, BorrowedFile: file, Tie: foreignOut})
	if out != nil || !errors.Is(e, arena.ErrConfig) || b.Used() != before {
		t.Fatal("foreign tie before header reservation", e)
	}
	old, e := base.SetTie(foreignOut)
	if !errors.Is(e, arena.ErrConfig) || old != nil {
		t.Fatal("foreign tie", old, e)
	}
	tie, e := base.Tie()
	paramsMust(t, e)
	if tie != streams.stdout {
		t.Fatal("old borrow replaced")
	}
}
func TestNumericCharInputBadOutputDoesNotFailInputAndCloseKeepsBorrows(t *testing.T) {
	input, streams, b, file := numericCharInputFixture(t, []byte("A"))
	out, e := streams.Stdout()
	paramsMust(t, e)
	paramsMust(t, out.file.Close())
	paramsMust(t, out.AppendByte('X'))
	bad, e := out.Bad()
	paramsMust(t, e)
	if !bad {
		t.Fatal("actual output badbit")
	}
	dst := byte(0)
	b.FailAfter(0)
	ok, e := input.ExtractChar(&dst)
	paramsMust(t, e)
	if !ok || dst != 'A' {
		t.Fatal("nonthrowing tied output badbit", ok, dst)
	}
	state, e := input.State()
	paramsMust(t, e)
	if state.Bad || state.Fail || state.EOF {
		t.Fatal("output state copied to input", state)
	}
	numericCharInputPhysical(t, input, streams, b)
	b.FailAfter(-1)
	paramsMust(t, input.Close())
	numericCharInputPhysical(t, input, streams, b)
	_, e = file.Seek(0, 0)
	paramsMust(t, e)
	paramsMust(t, out.Flush())
	paramsMust(t, out.AppendByte('Y'))
}
func TestNumericCharInputCopiedOwnerAndReachedNilDestination(t *testing.T) {
	input, streams, b, file := numericCharInputFixture(t, []byte("A"))
	copy := *input
	if _, e := copy.ExtractChar(nil); !errors.Is(e, ErrNumericCharInputOwner) {
		t.Fatal("copied owner", e)
	}
	pos, e := file.Seek(0, 1)
	paramsMust(t, e)
	if pos != 0 {
		t.Fatal("copied owner read provider")
	}
	_, e = input.ExtractChar(nil)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual assignment guard after consumed char", e)
	}
	state, e := input.State()
	paramsMust(t, e)
	if state.Position != 1 || state.Bad || state.Fail || state.EOF {
		t.Fatal("reached null ref guard domain", state)
	}
	numericCharInputPhysical(t, input, streams, b)
}
