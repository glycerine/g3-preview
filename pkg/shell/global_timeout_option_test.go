package shell

import (
	"errors"
	"testing"
	"time"
)

func TestShellGlobalTimeoutOptionSourceBranch(t *testing.T) {
	s := NewTimeoutService(nil)
	defer func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := ApplyGlobalTimeoutOption(nil, s); !errors.Is(err, ErrMissingGlobalTimeout) {
		t.Fatal("missing colon argument", err)
	}
	if s.timer != nil {
		t.Fatal("missing option changed timeout slot")
	}
	for _, argument := range []string{"", "not-a-number", "0", "-", "4294967295garbage\x00ignored"} {
		if err := ApplyGlobalTimeoutOption(&argument, s); err != nil {
			t.Fatal("source decimal prefix", argument, err)
		}
		if s.timer == nil {
			t.Fatal("source set_timeout did not own disabled timer", argument)
		}
	}
	argument := "  +12suffix"
	if err := ApplyGlobalTimeoutOption(&argument, s); err != nil {
		t.Fatal(err)
	}
	if s.timer == nil {
		t.Fatal("positive option did not replace timer")
	}
	argument = "9223372036854775807"
	if err := ApplyGlobalTimeoutOption(&argument, s); !errors.Is(err, ErrGlobalTimeoutRange) {
		t.Fatal("undefined source multiplication", err)
	}
	if s.timer == nil {
		t.Fatal("refused option changed prior timer")
	}
	argument = "-9223372036854775808"
	if err := ApplyGlobalTimeoutOption(&argument, s); !errors.Is(err, ErrGlobalTimeoutRange) {
		t.Fatal("negative undefined multiplication", err)
	}
	argument = "0"
	if err := ApplyGlobalTimeoutOption(&argument, s); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-s.Events():
		t.Fatal("disabled replacement fired", err)
	case <-time.After(20 * time.Millisecond):
	}
}
