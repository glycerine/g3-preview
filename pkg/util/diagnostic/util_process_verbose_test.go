package diagnostic

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestProcessVerboseGlobalsDefaultAndBorrowedSelection(t *testing.T) {
	oldLevel, err := ProcessVerbosityLevel()
	if err != nil {
		t.Fatal(err)
	}
	oldStream, err := BorrowProcessVerboseStream()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := SetProcessVerboseStream(oldStream); err != nil {
			t.Error(err)
		}
		if err := SetProcessVerbosityLevel(oldLevel); err != nil {
			t.Error(err)
		}
	})
	if oldLevel != 0 || oldStream.File != os.Stderr {
		t.Fatal("source process defaults", oldLevel, oldStream.File)
	}
	if err := SetProcessVerbosityLevel(7); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		level uint32
		want  bool
	}{{0, true}, {7, true}, {8, false}} {
		got, err := ProcessVerboseEnabled(tc.level)
		if err != nil || got != tc.want {
			t.Fatal("source IF_VERBOSE gate", tc, got, err)
		}
	}
	b := arena.NewBudget(1024)
	other, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	if err := SetProcessVerboseStream(other); err != nil {
		t.Fatal(err)
	}
	if got, err := BorrowProcessVerboseStream(); err != nil || got != other {
		t.Fatal("replacement stream pointer", got, err)
	}
	if err := SetProcessVerboseStream(nil); !errors.Is(err, arena.ErrConfig) {
		t.Fatal("nil ostream reference", err)
	}
	if got, err := BorrowProcessVerboseStream(); err != nil || got != other {
		t.Fatal("nil replacement changed selection", got, err)
	}
	if err := SetProcessVerboseStream(oldStream); err != nil {
		t.Fatal(err)
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
	if b.Used() != 0 {
		t.Fatal("replacement owner retained", b.Used())
	}
}
