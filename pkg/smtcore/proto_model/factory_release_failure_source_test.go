//go:build g3checked && g3debug

package protomodel

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/cooperate"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func factoryReleaseTLSMinimum(t *testing.T, r *umemory.Runtime) {
	t.Helper()
	first, err := r.Allocate(8)
	protoMust(t, err)
	second, err := r.Allocate(8)
	protoMust(t, err)
	delta := int64(-1000000)
	request := uint64(16) + uint64(delta) - 8
	if err = r.Reallocate(&first, request); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("first TLS frontier", err)
	}
	protoMust(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal("TLS synchronization", state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if err = r.Reallocate(&second, request); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("minimum TLS frontier", err)
	}
	protoMust(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal("TLS accounting state", state)
	}
}

func TestFactoryRealNumeralReleaseFailureRetry(t *testing.T) {
	for _, kind := range []string{"arith", "bv", "datalog", "user", "array", "datatype"} {
		t.Run(kind, func(t *testing.T) {
			b := arena.NewBudget(1 << 28)
			rt, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
			protoMust(t, err)
			cfg := unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
			q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: cfg})
			protoMust(t, err)
			symbols, err := usymbol.NewManager(usymbol.Config{Budget: b})
			protoMust(t, err)
			m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: q, Budget: b})
			protoMust(t, err)
			_, err = m.RegisterArithmeticPlugin()
			protoMust(t, err)
			scoped, err := unumeral.NewScopedInt(q.Manager.Z)
			protoMust(t, err)
			protoMust(t, scoped.SetInt64(1))
			raw, err := scoped.Get()
			protoMust(t, err)
			protoMust(t, q.Manager.Z.Mul2kInPlace(raw, 1600000))
			rational, err := q.NewFromScopedInt(scoped)
			protoMust(t, err)
			large, err := m.ArithmeticNumeral(&rational, false)
			protoMust(t, err)
			real, err := m.GetSort(large)
			protoMust(t, err)
			var f ValueFactory
			var md *ProtoModel
			var closeServices func()
			if kind == "array" || kind == "datatype" {
				protoMust(t, m.RegisterDeclarationPlugins())
				md, closeServices = structuredReleaseModel(t, m, rt)
			}
			var sort ast.ASTHandle
			switch kind {
			case "arith":
				f, err = NewArithmeticFactory(m)
				sort = real
			case "bv":
				_, err = m.RegisterBitVectorPlugin()
				protoMust(t, err)
				f, err = NewBitVectorFactory(m)
				protoMust(t, err)
				sort, err = m.BitVectorSort(8)
			case "datalog":
				f, err = NewDatalogFactory(m)
				protoMust(t, err)
				sort, err = m.MkDatalogFiniteSort(protoSymbol(t, m, "release-domain"), 8)
			case "array":
				f, err = NewArrayFactory(md)
				protoMust(t, err)
				boolean, e := m.MkBoolSort()
				protoMust(t, e)
				sort, err = m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
			case "datatype":
				f, err = NewDatatypeFactory(md)
				protoMust(t, err)
				sorts, e := m.MkDatatypes([]ast.DatatypeDefinition{{Name: protoSymbol(t, m, "release-datatype"), Constructors: []ast.DatatypeConstructorDefinition{{Name: protoSymbol(t, m, "release-dt-value"), Recognizer: protoSymbol(t, m, "is-release-dt-value")}}}})
				protoMust(t, e)
				sort = sorts[0]
			case "user":
				f = newUserSortFactory(m)
				sort, err = m.MkUninterpretedSort(protoSymbol(t, m, "release-universe"), nil)
			}
			protoMust(t, err)
			if md != nil {
				protoMust(t, md.RegisterFactory(f))
			}
			value := large
			if kind != "arith" {
				// RegisterValue accepts an expression of the factory's sort. A real
				// arithmetic child gives every retained factory the same numerical
				// release frontier without changing any NUM or owner payload.
				declaration, err := m.MkFuncDecl(protoSymbol(t, m, "value-with-numeral"), []ast.ASTHandle{real}, sort, ast.ASTFuncDeclInfo{})
				protoMust(t, err)
				value, err = m.MkApp(declaration, []ast.ASTHandle{large})
				protoMust(t, err)
			}
			protoMust(t, f.RegisterValue(value))
			protoMust(t, q.Del(&rational))
			protoMust(t, scoped.Close())
			later, err := m.MkConst(protoSymbol(t, m, "independent-later-value"), sort)
			protoMust(t, err)
			protoMust(t, f.RegisterValue(later))
			held, err := m.NewRef(later)
			protoMust(t, err)
			beforeCount, err := m.GetRefCount(later)
			protoMust(t, err)
			beforeBlocks, beforeRefs := rt.Stats().LiveBlocks, m.Stats().LiveReferences
			factoryReleaseTLSMinimum(t, rt)
			stopped := f.Close()
			if !errors.Is(stopped, arena.ErrBounds) || rt.Stats().LiveBlocks >= beforeBlocks {
				t.Fatal("did not reach a consumed physical factory NUM release failure", stopped, rt.State())
			}
			// DelRef reports the NUM error after consuming its AST reference. The
			// remaining numerical deletion belongs to the borrowed AST manager.
			if m.Stats().LiveReferences+3 != beforeRefs {
				t.Fatal("factory reference retirement prefix", m.Stats(), beforeRefs)
			}
			count, err := m.GetRefCount(later)
			protoMust(t, err)
			if count+1 != beforeCount {
				t.Fatal("independent later owner changed", count, beforeCount)
			}
			recovery, err := rt.Allocate(400000)
			protoMust(t, err)
			managerBeforeRetry := m.Stats()
			stateBeforeRetry := rt.State()
			retry := f.Close()
			rootsAfter := m.Stats().LiveReferences
			again := f.Close()
			if m.Stats() != managerBeforeRetry || rt.State() != stateBeforeRetry {
				t.Fatal("factory retry changed the borrowed manager deletion graph or numerical owner")
			}
			if m.Stats().LiveReferences != rootsAfter {
				t.Fatal("retry replayed a consumed AST decrement")
			}
			// Preserve the real numerical owner and finish its own deletion graph;
			// factory retry must not fabricate a replacement AST reference.
			finish := func(close func() error) {
				t.Helper()
				for attempts := 0; attempts < 128; attempts++ {
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
			protoMust(t, m.DelRef(&held))
			if md != nil {
				finish(md.Close)
			}
			finish(m.Close)
			if closeServices != nil {
				closeServices()
			}
			finish(q.Close)
			finish(symbols.Close)
			if err = rt.Deallocate(&recovery); err != nil && (!errors.Is(err, arena.ErrBounds) || recovery != (umemory.Block{})) {
				t.Fatal(err)
			}
			finish(rt.Close)
			if b.Used() != 0 {
				t.Fatal("fixture retained bytes", b.Used())
			}
			if retry != nil || again != nil {
				t.Fatal("owning retry revisited a retired AST reference", retry, again)
			}
		})
	}
}

// Build the actual model/evaluator owner needed by structured constructors,
// sharing the physical numeral Runtime used by the release frontier above.
func structuredReleaseModel(t *testing.T, m *ast.ASTManager, rt *umemory.Runtime) (*ProtoModel, func()) {
	t.Helper()
	b := m.Budget()
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: b})
	protoMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: b})
	protoMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	protoMust(t, err)
	protoMust(t, globals.InitializeSource())
	params, err := context.NewValue()
	protoMust(t, err)
	cooperation, err := cooperate.NewCooperation(b)
	protoMust(t, err)
	md, err := NewProtoModel(m, model.ModelEvaluatorConfig{Context: context, Globals: globals, Params: &params, Memory: rt, Cooperation: cooperation})
	protoMust(t, err)
	return md, func() {
		protoMust(t, cooperation.Close())
		protoMust(t, context.Del(&params))
		protoMust(t, globals.Close())
		protoMust(t, catalog.Close())
		protoMust(t, context.Close())
	}
}
