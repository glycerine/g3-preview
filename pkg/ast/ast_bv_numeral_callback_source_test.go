// Source: src/ast/bv_decl_plugin.cpp:143-155; src/util/rational.cpp:32-54;
// src/util/memory_manager.cpp:274-289.
package ast

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Actual returning Go ErrorTransport profile; no native C++ callback equivalence.
// The bounded child protects the parent if a held service mutex blocks reentry.
func TestASTBitVectorNumericalPowerRealCallbackOwnerGuard(t *testing.T) {
	if os.Getenv("G3_BV_NUMERAL_CALLBACK_CHILD") == "1" {
		b := arena.NewBudget(1 << 28)
		var q, other *util.RationalService
		var rt *util.Runtime
		var e error
		var target, foreign util.OwnedRational
		active, calls := false, 0
		rt, e = util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
			if active {
				calls++
				if !rt.FailureTransportActive() {
					t.Fatal("real dynamic transport not stamped")
				}
				if own := q.Close(); !errors.Is(own, arena.ErrInvalidSlot) {
					t.Fatal("executing borrowed numerical owner changed", own)
				}
				if source := q.SourceFinalize(); !errors.Is(source, arena.ErrInvalidSlot) {
					t.Fatal("executing source numerical owner changed", source)
				}
				if init := q.InitializeSource(); !errors.Is(init, arena.ErrInvalidSlot) {
					t.Fatal("executing initialization blocked on held mutex", init)
				}
				if _, mask := q.SourceEndedStaticMask(); !errors.Is(mask, arena.ErrInvalidSlot) {
					t.Fatal("executing static inspection blocked on held mutex", mask)
				}
				if copy := q.CopyFrom(&target, other, &foreign); !errors.Is(copy, arena.ErrInvalidSlot) {
					t.Fatal("executing two-owner copy blocked on held mutex", copy)
				}
				if _, read := q.Stats(); !errors.Is(read, arena.ErrInvalidSlot) {
					t.Fatal("executing numerical read blocked on held mutex", read)
				}

			}
			return f
		}})
		astTestMust(t, e)
		q, e = util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		other, e = util.PrepareRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		target, e = q.NewValue()
		astTestMust(t, e)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Budget: b, Symbols: sy, Numerals: q})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused physical anchor checkpoint") }, Cooperation: func() error { panic("unused physical anchor cooperation") }})
		astTestMust(t, e)
		// Exact actual header-inclusive TLS frontier, before any limit is installed.
		pad := int64(99990) - rt.State().LocalSize - 8
		if pad <= 0 {
			t.Fatal("actual callback setup", rt.State())
		}
		padding, e := rt.Allocate(uint64(pad))
		astTestMust(t, e)
		astTestMust(t, rt.SetMaxSize(1))
		active = true
		_, e = m.RegisterBitVectorPlugin()
		active = false
		if rt.FailureTransportActive() {
			t.Fatal("dynamic transport stamp survived return")
		}
		if e == nil || calls == 0 {
			t.Fatal("actual numerical allocation did not reach transport", e, calls, rt.State())
		}
		upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, CallerBlocks: []util.Block{padding}})
		// The actual caller owns q and its lexical recovery, unlike the borrowed AST.
		astTestMust(t, rt.SetMaxSize(0))
		astTestMust(t, q.CleanupLexical())
		astTestMust(t, m.Close())
		astTestMust(t, anchor.Close())
		astTestMust(t, q.Close())
		astTestMust(t, other.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, rt.Deallocate(&padding))
		astTestMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("exact caller context end", b.Used())
		}

		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTBitVectorNumericalPowerRealCallbackOwnerGuard$", "-test.count=1")
	cmd.Env = append(os.Environ(), "G3_BV_NUMERAL_CALLBACK_CHILD=1")
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("real numerical callback child: %v; %s", e, output)
	}
}
