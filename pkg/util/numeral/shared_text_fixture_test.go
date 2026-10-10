package numeral

import (
	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
	"os"
	"testing"
)

func numericFileFixture(t *testing.T) (*os.File, *utext.NumericTextStream, *arena.Budget) {
	t.Helper()
	f, e := os.CreateTemp(".", "numeric-file-*")
	if e != nil {
		t.Fatal(e)
	}
	b := arena.NewBudget(1 << 20)
	s, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b, BorrowedFile: f})
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
