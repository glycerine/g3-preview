package diagnostic

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestNumericCharInputClosedTieFailsAtReachedFlush(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	serviceMust(t, e)
	serviceMust(t, out.Close())
	file, e := os.CreateTemp(t.TempDir(), "closed-input-*")
	serviceMust(t, e)
	serviceMust(t, file.Close())
	input, e := NewNumericCharInput(NumericCharInputConfig{Budget: b, BorrowedFile: file, Tie: out})
	serviceMust(t, e) // Identity/config binding does not flush or require an open tie.
	defer input.Close()
	before, e := input.State()
	serviceMust(t, e)
	invalid := utext.NumericTextStream{Budget: b}
	if _, e = input.SetTie(&invalid); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("uninitialized output owner must not replace the tie", e)
	}
	dst := byte('Q')
	ok, e := input.ExtractChar(&dst)
	if ok || !errors.Is(e, arena.ErrClosed) || dst != 'Q' {
		t.Fatal("reached tie flush must precede file/destination access", ok, e, dst)
	}
	after, e := input.State()
	serviceMust(t, e)
	if after != before {
		t.Fatal("tie refusal consumed input or changed extraction flags", before, after)
	}
	serviceMust(t, input.Close())
	if b.Used() != 0 {
		t.Fatal("closed tie/input retained budget", b.Used())
	}
}
