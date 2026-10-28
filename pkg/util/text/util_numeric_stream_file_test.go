package text

// Source: src/util/memory_manager.cpp:49-72; src/util/memory_manager.h:49-71.

import (
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func numericFileFixture(t *testing.T) (*os.File, *NumericTextStream, *arena.Budget) {
	t.Helper()
	f, e := os.CreateTemp(".", "numeric-file-*")
	if e != nil {
		t.Fatal(e)
	}
	b := arena.NewBudget(1 << 20)
	s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		s.Close()
		f.Close()
		os.Remove(f.Name())
		if b.Used() != 0 {
			t.Error("file stream cold graph leak", b.Used())
		}
	})
	return f, s, b
}
func TestNumericFileStreamExactFormattingAndBorrow(t *testing.T) {
	f, s, b := numericFileFixture(t)
	if s.Buffer != nil || b.Used() != uint64(reflect.TypeOf(NumericTextStream{}).Size()) {
		t.Fatal("unused shadow output buffer", s.Stats(), b.Used())
	}
	textMust(t, s.SetWidth(5))
	textMust(t, s.SetFill('_'))
	textMust(t, s.AppendCString("A\x00ignored"))
	textMust(t, s.SetShowPos(true))
	textMust(t, s.AppendInt64(12))
	textMust(t, s.AppendByte('\n'))
	_, e := f.Seek(0, io.SeekStart)
	textMust(t, e)
	got, e := io.ReadAll(f)
	textMust(t, e)
	if string(got) != "____A+12\n" {
		t.Fatal(string(got))
	}
	st := s.Stats()
	if st.Position != uint64(len(got)) || st.OutputCapacity != 0 || st.OutputOwnerBytes != 0 || st.OutputOverflowBytes != 0 {
		t.Fatal("physical file output stats", st)
	}
	if _, e = s.String(); !errors.Is(e, ErrNumericStreamFile) {
		t.Fatal("file has no retained output", e)
	}
	if e = s.ResetText(); !errors.Is(e, ErrNumericStreamFile) {
		t.Fatal("no synthetic file reset", e)
	}
	textMust(t, s.Close())
	if _, e = f.Write([]byte("caller")); e != nil {
		t.Fatal("stream closed borrowed file", e)
	}
}
func TestNumericFileStreamActualWriteFailureSetsNonthrowingBadbit(t *testing.T) {
	f, s, b := numericFileFixture(t)
	textMust(t, f.Close())
	used := b.Used()
	textMust(t, s.AppendCString("first"))
	bad, e := s.Bad()
	textMust(t, e)
	if !bad {
		t.Fatal("closed physical provider must set badbit")
	}
	textMust(t, s.SetWidth(7))
	textMust(t, s.AppendInt64(3))
	textMust(t, s.AppendCString("next"))
	if s.State().Width != 7 || s.Stats().Position != 0 || used != b.Used() {
		t.Fatal("failed sentry consumed width or reserved storage", s.State(), s.Stats())
	}
}
func TestNumericFileStreamEveryColdReservation(t *testing.T) {
	f, e := os.CreateTemp(".", "numeric-file-cold-*")
	textMust(t, e)
	defer f.Close()
	defer os.Remove(f.Name())
	b := arena.NewBudget(1 << 20)
	b.FailAfter(0)
	s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != 0 {
		t.Fatal(s, e, b.Used())
	}
	b.FailAfter(1)
	s, e = NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	textMust(t, e)
	if s.Buffer != nil || b.Used() != s.Stats().OwnerBytes {
		t.Fatal("one actual cold header", s.Stats(), b.Used())
	}
	b.FailAfter(-1)
	textMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
