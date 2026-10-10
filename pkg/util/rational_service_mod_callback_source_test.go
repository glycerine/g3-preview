// Source: src/util/rational.h; src/util/mpq.cpp; src/util/memory_manager.cpp:274-285.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Returning Go transport is an adapter profile, not a native callback claim.
func TestRationalServiceModActualTransportRefusesHeldOwner(t *testing.T) {
	if os.Getenv("G3_RATIONAL_MOD_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRationalServiceModActualTransportRefusesHeldOwner$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_RATIONAL_MOD_CALLBACK_CHILD=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("actual Mod callback: %v; %s", e, out)
		}
		return
	}
	b := arena.NewBudget(1 << 28)
	var s *RationalService
	active, calls := false, 0
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			used, stage := b.Used(), s.cleanupStage
			if e := s.CloseChecked(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("own reentry", e)
			}
			if _, e := s.Stats(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("read reentry", e)
			}
			if s.closed || s.closing || s.cleanupStage != stage || b.Used() != used {
				t.Fatal("callback mutated held owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = rt
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e = NewRationalService(cfg)
	serviceMust(t, e)
	a, e := s.NewValue()
	serviceMust(t, e)
	divisor, e := s.NewValue()
	serviceMust(t, e)
	words := make([]uint32, 2000)
	for i := range words {
		words[i] = ^uint32(0)
	}
	serviceMust(t, s.Manager.Z.SetLimbs(&a.Raw.Num, words))
	serviceMust(t, s.SetInt64(&divisor, 3))
	pad := int64(99990) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("setup TLS frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	out, cause := s.Mod(&a, &divisor)
	active = false
	if calls == 0 || cause == nil {
		t.Fatal("actual Mod allocation transport not reached", calls, cause)
	}
	if s.closed || s.closing {
		t.Fatal("held owner retired")
	}
	roots := []*Int{&a.Raw.Num, &a.Raw.Den, &divisor.Raw.Num, &divisor.Raw.Den, &out.Raw.Num, &out.Raw.Den}
	for i := uint8(0); i < s.Lexical.Count; i++ {
		roots = append(roots, &s.Lexical.Roots[i].Num, &s.Lexical.Roots[i].Den)
	}
	integerPhysicalAuditWithRaw(t, s.Manager.Z, []umemory.Block{held}, roots...)
	if want := s.HeaderBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z); b.Used() != want {
		t.Fatal("actual cold graph before limit lift", b.Used(), want)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, s.CleanupLexical())
	if out.ownerID != 0 {
		serviceMust(t, s.CleanupValue(&out))
	}
	serviceMust(t, s.CleanupValue(&a))
	serviceMust(t, s.CleanupValue(&divisor))
	serviceMust(t, s.CloseChecked())
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("actual owner end", b.Used())
	}
}
