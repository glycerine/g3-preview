package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"testing"
)

func TestAlgebraicCheckpointSourceExceptionAndResume(t *testing.T) {
	f := algebraicSourceFixture(t)
	before := f.limit.Count()
	cooperationCalls := 0
	original := f.core.config.Cooperation
	f.core.config.Cooperation = func() error { cooperationCalls++; return original() }
	f.limit.Cancel()
	e := f.core.checkpoint()
	var specific AlgebraicException
	var base z3exception.DefaultException
	if !errors.As(e, &specific) || !errors.As(e, &base) || e.Error() != "canceled" || base.Message() != "canceled" || specific.HasErrorCode() || !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal("source algebraic cancellation class/message", e)
	}
	if f.limit.Count() != before+1 || f.algebraicTicks != 1 || cooperationCalls != 0 {
		t.Fatal("checkpoint order", f.limit.Count(), before, f.algebraicTicks)
	}
	f.limit.ResetCancel()
	algebraicSourceMust(t, f.core.checkpoint())
	if f.limit.Count() != before+2 || f.algebraicTicks != 2 || cooperationCalls != 1 {
		t.Fatal("resumed checkpoint", f.limit.Count(), f.algebraicTicks)
	}
	// A matching message alone does not classify an arbitrary root-object error
	// as resource cancellation.
	ordinary := newAlgebraicException("canceled")
	if errors.Is(ordinary, ErrUPolynomialCanceled) {
		t.Fatal("message classified as cancellation")
	}
}
