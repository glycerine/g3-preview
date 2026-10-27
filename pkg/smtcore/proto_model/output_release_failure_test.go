//go:build g3checked && g3debug

package protomodel

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Exercise the actual incoming-INC / old-DEC / publication boundary using a
// real NUM release failure. No AST/NUM descriptors or accounting fields change
// through test-only writes; the existing public Runtime frontier is used.
func TestFactoryOutputsRealOldNumeralReleaseFailure(t *testing.T) {
	for _, kind := range []string{"fpa", "string"} {
		t.Run(kind, func(t *testing.T) {
			budget := arena.NewBudget(1 << 28)
			rt, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
			protoMust(t, err)
			cfg := unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
			q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: cfg})
			protoMust(t, err)
			symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
			protoMust(t, err)
			m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: q, Budget: budget})
			protoMust(t, err)
			protoMust(t, m.RegisterDeclarationPlugins())
			var f ValueFactory
			var sort, zero ast.ASTHandle
			if kind == "fpa" {
				family, e := m.RegisterFloatingPointPlugin()
				protoMust(t, e)
				f, err = NewFloatingPointFactory(m, family)
				protoMust(t, err)
				sort, err = m.FloatingPointSort(8, 24)
				protoMust(t, err)
				zero, err = f.SomeValue(sort)
				protoMust(t, err)
			} else {
				family, e := m.RegisterSequencePlugin()
				protoMust(t, e)
				f = &StringFactory{manager: m, family: family, strings: make(map[string]struct{})}
				zero, err = m.MkString("value 1")
				protoMust(t, err)
				sort, err = m.GetSort(zero)
				protoMust(t, err)
			}
			keep, err := m.NewRef(zero)
			protoMust(t, err)
			scoped, err := unumeral.NewScopedInt(q.Manager.Z)
			protoMust(t, err)
			protoMust(t, scoped.SetInt64(1))
			raw, err := scoped.Get()
			protoMust(t, err)
			protoMust(t, q.Manager.Z.Mul2kInPlace(raw, 1600000))
			rational, err := q.NewFromScopedInt(scoped)
			protoMust(t, err)
			old, err := m.ArithmeticNumeral(&rational, false)
			protoMust(t, err)
			first, err := m.NewRef(old)
			protoMust(t, err)
			protoMust(t, q.Del(&rational))
			protoMust(t, scoped.Close())
			truth, err := m.MkTrue()
			protoMust(t, err)
			second, err := m.NewRef(truth)
			protoMust(t, err)
			before, err := m.GetRefCount(zero)
			protoMust(t, err)
			blocks := rt.Stats().LiveBlocks
			factoryReleaseTLSMinimum(t, rt)
			ok, call := f.SomeValues(sort, first, second)
			if ok || !errors.Is(call, arena.ErrBounds) || rt.Stats().LiveBlocks >= blocks {
				t.Fatal("did not reach consumed physical old-output NUM release", ok, call, rt.State())
			}
			after, err := m.GetRefCount(zero)
			protoMust(t, err)
			if after != before+1 {
				t.Fatal("incoming reference was not preserved before publication", before, after)
			}
			untouched, err := m.RefGet(second)
			protoMust(t, err)
			if untouched != truth {
				t.Fatal("continued to second output after old first release failed")
			}
			protoMust(t, f.Close())
			after, err = m.GetRefCount(zero)
			protoMust(t, err)
			if after != before+1 {
				t.Fatal("factory close consumed caller's pending incoming reference")
			}
			recovery, err := rt.Allocate(400000)
			protoMust(t, err)
			protoMust(t, m.DelRef(&first))
			after, err = m.GetRefCount(zero)
			protoMust(t, err)
			if after != before {
				t.Fatal("caller cleanup did not consume its one pending acquisition", before, after)
			}
			protoMust(t, m.DelRef(&second))
			protoMust(t, m.DelRef(&keep))
			finish := func(close func() error) {
				t.Helper()
				for attempt := 0; attempt < 128; attempt++ {
					before := rt.Stats().LiveBlocks
					err := close()
					if err == nil {
						return
					}
					if !errors.Is(err, arena.ErrBounds) || rt.Stats().LiveBlocks >= before {
						t.Fatal("cleanup made no physical progress", err, rt.State())
					}
				}
				t.Fatal("cleanup horizon")
			}
			finish(m.Close)
			finish(q.Close)
			finish(symbols.Close)
			if err = rt.Deallocate(&recovery); err != nil && (!errors.Is(err, arena.ErrBounds) || recovery != (umemory.Block{})) {
				t.Fatal(err)
			}
			finish(rt.Close)
			if budget.Used() != 0 {
				t.Fatal("retained output/NUM storage", budget.Used())
			}
		})
	}
}
