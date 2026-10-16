package model

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// The checked deleted-ID guard makes the real AST final release consume its
// reference and then fail. It exercises the owner's retry cursor without
// claiming a naturally reached source deletion failure.
func TestFunctionInterpretationConsumedASTReleaseRetry(t *testing.T) {
	if !ast.ASTSourceDebug {
		t.Skip("source deleted-ID assertion is checked-build only")
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	budget := arena.NewBudget(1 << 30)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	must(err)
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{
		Integers: unumeral.IntegerConfig{
			Cells: arena.Config{Budget: budget},
			Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
		},
	})
	must(err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	must(err)
	t.Cleanup(func() {
		must(manager.Close())
		must(numerals.Close())
		must(symbols.Close())
		if budget.Used() != 0 {
			t.Fatalf("owner graph retained %d bytes", budget.Used())
		}
	})
	must(manager.EnableDebugRefCount())
	boolean, err := manager.MkBoolSort()
	must(err)
	value, err := manager.MkVar(101, boolean)
	must(err)
	function, err := NewFunctionInterpretation(manager, 0)
	must(err)
	must(function.SetElse(value))
	id, err := manager.GetASTID(value)
	must(err)
	manager.DebugFreeIndices[id] = struct{}{}
	err = function.Close()
	if !errors.Is(err, ast.ErrASTDomain) || !function.fallback.IsNull() || function.closed {
		t.Fatalf("consumed AST release must clear fallback before returning: err=%v fallback=%v closed=%v", err, function.fallback, function.closed)
	}
	delete(manager.DebugFreeIndices, id)
	must(function.Close())
	if !function.closed || function.bytes != 0 {
		t.Fatalf("owning retry did not complete: closed=%v bytes=%d", function.closed, function.bytes)
	}
	cacheValue, err := manager.MkFreshConst("cache-value", boolean)
	must(err)
	cached, err := NewFunctionInterpretation(manager, 0)
	must(err)
	must(manager.IncRef(cacheValue))
	cached.cached = cacheValue
	cacheID, err := manager.GetASTID(cacheValue)
	must(err)
	manager.DebugFreeIndices[cacheID] = struct{}{}
	err = cached.resetCache()
	if !errors.Is(err, ast.ErrASTDomain) || !cached.cached.IsNull() {
		t.Fatalf("consumed cache release must clear the cache: err=%v cached=%v", err, cached.cached)
	}
	delete(manager.DebugFreeIndices, cacheID)
	must(cached.Close())
	oldValue, err := manager.MkVar(102, boolean)
	must(err)
	newValue, err := manager.MkVar(103, boolean)
	must(err)
	replacing, err := NewFunctionInterpretation(manager, 0)
	must(err)
	must(replacing.SetElse(oldValue))
	oldID, err := manager.GetASTID(oldValue)
	must(err)
	manager.DebugFreeIndices[oldID] = struct{}{}
	err = replacing.SetElse(newValue)
	if !errors.Is(err, ast.ErrASTDomain) || !replacing.fallback.IsNull() || replacing.replacement != newValue {
		t.Fatalf("stopped replacement lost consumed/incoming ownership: err=%v fallback=%v incoming=%v", err, replacing.fallback, replacing.replacement)
	}
	if _, err = replacing.Else(); !errors.Is(err, ast.ErrASTPending) {
		t.Fatalf("partly replaced owner was reopened: %v", err)
	}
	delete(manager.DebugFreeIndices, oldID)
	must(replacing.Close())
	if !replacing.closed || !replacing.replacement.IsNull() {
		t.Fatalf("replacement cleanup did not finish: closed=%v incoming=%v", replacing.closed, replacing.replacement)
	}
	oldResult, err := manager.MkFreshConst("old-result", boolean)
	must(err)
	newResult, err := manager.MkFreshConst("new-result", boolean)
	must(err)
	entry, err := newFunctionEntry(manager, nil, oldResult)
	must(err)
	resultID, err := manager.GetASTID(oldResult)
	must(err)
	manager.DebugFreeIndices[resultID] = struct{}{}
	err = entry.setResult(newResult)
	if !errors.Is(err, ast.ErrASTDomain) || entry.resultOwned || !entry.result.IsNull() || entry.replacement != newResult {
		t.Fatalf("stopped result replacement lost consumed/incoming ownership: err=%v result=%v owned=%v incoming=%v", err, entry.result, entry.resultOwned, entry.replacement)
	}
	if _, err = entry.Result(); !errors.Is(err, ast.ErrASTPending) {
		t.Fatalf("partly replaced entry was reopened: %v", err)
	}
	delete(manager.DebugFreeIndices, resultID)
	must(entry.close())
	if !entry.closed || !entry.replacement.IsNull() {
		t.Fatalf("entry replacement cleanup did not finish: closed=%v incoming=%v", entry.closed, entry.replacement)
	}
	argument, err := manager.MkFreshConst("entry-arg", boolean)
	must(err)
	valueResult, err := manager.MkFreshConst("entry-value", boolean)
	must(err)
	entry, err = newFunctionEntry(manager, []ast.ASTHandle{argument}, valueResult)
	must(err)
	argumentID, err := manager.GetASTID(argument)
	must(err)
	manager.DebugFreeIndices[argumentID] = struct{}{}
	err = entry.close()
	if !errors.Is(err, ast.ErrASTDomain) || entry.drain != 1 || !entry.args[0].IsNull() || entry.closed {
		t.Fatalf("consumed argument must advance destructor cursor: err=%v drain=%d argument=%v closed=%v", err, entry.drain, entry.args[0], entry.closed)
	}
	delete(manager.DebugFreeIndices, argumentID)
	must(entry.close())
	if !entry.closed || entry.resultOwned {
		t.Fatalf("entry retry did not finish result release: closed=%v resultOwned=%v", entry.closed, entry.resultOwned)
	}
}
