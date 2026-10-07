// Source: src/ast/array_decl_plugin.cpp:86-113; src/util/memory_manager.cpp:274-289.
package ast

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTArrayFiniteRealCallbackOwnerGuard(t *testing.T) {
	if os.Getenv("G3_ARRAY_FINITE_CALLBACK_CHILD") == "1" {
		b := arena.NewBudget(1 << 28)
		var m *ASTManager
		var q *util.RationalService
		active, calls := false, 0
		budgetMode := os.Getenv("G3_ARRAY_CALLBACK_BUDGET") != ""
		countMode := os.Getenv("G3_ARRAY_CALLBACK_COUNT") != "" || budgetMode
		rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: !countMode, SizeTBits: 64, Transport: func(f util.Failure) error {
			if active {
				calls++
				if own := m.Close(); !errors.Is(own, ErrASTPending) {
					t.Fatal("actual executing Array caller", own)
				}
				if own := q.Close(); !errors.Is(own, arena.ErrInvalidSlot) {
					t.Fatal("actual executing numerical child", own)
				}
			}
			return f
		}})
		astTestMust(t, e)
		q, e = util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
		astTestMust(t, e)
		m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		family, e := m.RegisterArrayPlugin()
		astTestMust(t, e)
		domains := []uint64{128}
		switch os.Getenv("G3_ARRAY_CALLBACK_CASE") {
		case "convert":
			domains = []uint64{^uint64(0)}
		case "multiply":
			domains = []uint64{65536, 65536}
		}
		var ps []ASTParameter
		for i, n := range domains {
			domain := astArrayFiniteTestSort(t, m, n, fmt.Sprintf("callback-domain-%d-%d", i, n))
			p, e := m.NewParameterAST(domain)
			astTestMust(t, e)
			ps = append(ps, p)
		}
		rp, e := m.NewParameterAST(m.external(m.boolSort))
		astTestMust(t, e)
		ps = append(ps, rp)
		anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused Array physical anchor") }, Cooperation: func() error { panic("unused Array physical anchor") }})
		astTestMust(t, e)
		var padding util.Block
		if budgetMode {
			ordinal, e := strconv.ParseInt(os.Getenv("G3_ARRAY_CALLBACK_BUDGET"), 10, 64)
			astTestMust(t, e)
			b.FailAfter(ordinal)
		} else if countMode {
			ordinal, e := strconv.ParseUint(os.Getenv("G3_ARRAY_CALLBACK_COUNT"), 10, 64)
			astTestMust(t, e)
			astTestMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+ordinal))
		} else {
			pad := int64(99990) - rt.State().LocalSize - 8
			if pad <= 0 {
				t.Fatal("actual numerical TLS frontier", rt.State())
			}
			padding, e = rt.Allocate(uint64(pad))
			astTestMust(t, e)
			astTestMust(t, rt.SetMaxSize(1))
		}
		active = true
		_, call := m.MkFamilySort(family, 0, ps)
		active = false
		if !countMode && (call == nil || calls == 0) {
			t.Fatal("actual numerical Power did not dispatch", call, calls, rt.State())
		}
		if budgetMode {
			if call == nil {
				fmt.Println("ARRAY_BUDGET_SOURCE_COMPLETE")
			} else if !errors.Is(call, arena.ErrInjectedFailure) {
				var failure util.Failure
				if !errors.As(call, &failure) || failure.Kind != util.FailureMemout {
					t.Fatal("actual constructor/getter/return reservation refusal", call)
				}
			}
		} else if countMode {
			if call == nil {
				fmt.Println("ARRAY_COUNT_SOURCE_COMPLETE")
			} else {
				var f util.Failure
				if !errors.As(call, &f) || f.Kind != util.FailureAllocExceeded || calls == 0 {
					t.Fatal("literal reached count failure", call, calls)
				}
				frame := m.ArrayCardinality
				if frame == nil || frame.bodyCause == nil || !frame.sourceDone || frame.sourceCause != nil {
					t.Fatal("actual caller source prefix", frame, call)
				}
				if os.Getenv("G3_ARRAY_CALLBACK_COUNT") == "3" && frame.tokens[3] == (util.OwnedRational{}) {
					t.Fatal("actual returned power constructor tuple lost before later Copy refusal")
				}
				for _, owned := range frame.Acquired {
					if owned {
						t.Fatal("source-completed values remain owned", frame)
					}
				}
			}
		}
		extras := upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}}
		if !countMode {
			extras.CallerBlocks = []util.Block{padding}
		}
		upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, extras)
		if budgetMode {
			b.FailAfter(-1)
		}
		astTestMust(t, rt.SetMaxSize(0))
		astTestMust(t, rt.SetMaxCount(0))
		astTestMust(t, q.CleanupLexical())
		foreign, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetUint64(&foreign, 29))
		frame := m.ArrayCardinality
		var body, source error
		var tuples [4]util.OwnedRational
		if frame != nil {
			body, source, tuples = frame.bodyCause, frame.sourceCause, frame.tokens
		}
		astTestMust(t, m.Close())
		if frame != nil && (frame.bodyCause != body || frame.sourceCause != source || frame.tokens != tuples || frame.header != 0) {
			t.Fatal("immutable exact lexical history at owning context end", frame)
		}
		n, e := q.Uint64(&foreign)
		astTestMust(t, e)
		if n != 29 {
			t.Fatal("borrowed numerical owner foreign value changed", n)
		}
		astTestMust(t, q.Del(&foreign))
		astTestMust(t, anchor.Close())
		astTestMust(t, q.Close())
		astTestMust(t, sy.Close())
		if !countMode {
			astTestMust(t, rt.Deallocate(&padding))
		}
		astTestMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual enclosing contexts ended", b.Used())
		}
		return
	}
	cases := []string{"power", "convert", "multiply"}
	if os.Getenv("G3_ARRAY_CALLBACK_COUNT_CAMPAIGN") == "1" {
		cases = []string{"power"}
	}
	for _, which := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTArrayFiniteRealCallbackOwnerGuard$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_ARRAY_FINITE_CALLBACK_CHILD=1", "G3_ARRAY_CALLBACK_CASE="+which)
		output, e := cmd.CombinedOutput()
		cancel()
		if e != nil {
			t.Fatalf("real finite-Array %s callback child: %v; %s", which, e, output)
		}
	}
}

func TestASTArrayFiniteEveryRuntimeCountFailureTransport(t *testing.T) {
	reached := false
	for ordinal := 0; ordinal < 40; ordinal++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTArrayFiniteRealCallbackOwnerGuard$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_ARRAY_FINITE_CALLBACK_CHILD=1", "G3_ARRAY_CALLBACK_CASE=power", "G3_ARRAY_CALLBACK_COUNT="+strconv.Itoa(ordinal))
		output, e := cmd.CombinedOutput()
		cancel()
		if e != nil {
			t.Fatalf("real count prefix %d: %v; %s", ordinal, e, output)
		}
		if strings.Contains(string(output), "ARRAY_COUNT_SOURCE_COMPLETE") {
			if ordinal == 0 {
				t.Fatal("no refusal prefix")
			}
			reached = true
			t.Log("first source count success", ordinal)
			break
		}
	}
	if !reached {
		t.Fatal("count horizon did not reach actual source completion")
	}
}

func TestASTArrayFiniteEveryActualBodyBudgetReservation(t *testing.T) {
	reached := false
	for ordinal := 0; ordinal < 128; ordinal++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTArrayFiniteRealCallbackOwnerGuard$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_ARRAY_FINITE_CALLBACK_CHILD=1", "G3_ARRAY_CALLBACK_CASE=power", "G3_ARRAY_CALLBACK_BUDGET="+strconv.Itoa(ordinal))
		output, e := cmd.CombinedOutput()
		cancel()
		if e != nil {
			t.Fatalf("actual body reservation prefix %d: %v; %s", ordinal, e, output)
		}
		if strings.Contains(string(output), "ARRAY_BUDGET_SOURCE_COMPLETE") {
			if ordinal == 0 {
				t.Fatal("no actual reservation refusal")
			}
			reached = true
			t.Log("first source all-body Budget success", ordinal)
			break
		}
	}
	if !reached {
		t.Fatal("actual body Budget horizon never completed")
	}
}
