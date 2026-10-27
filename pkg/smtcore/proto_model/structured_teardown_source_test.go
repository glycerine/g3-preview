package protomodel

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func structuredOwnerFixture(t *testing.T, kind string) (*ast.ASTManager, ValueFactory, *structuredFactory, ast.ASTHandle) {
	t.Helper()
	m, p := protoFixture(t)
	var f ValueFactory
	var base *structuredFactory
	var sort ast.ASTHandle
	if kind == "array" {
		af, err := NewArrayFactory(p)
		protoMust(t, err)
		f = af
		base = &af.structuredFactory
		boolean, err := m.MkBoolSort()
		protoMust(t, err)
		sort, err = m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
		protoMust(t, err)
	} else {
		df, err := NewDatatypeFactory(p)
		protoMust(t, err)
		f = df
		base = &df.structuredFactory
		sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: protoSymbol(t, m, "StructuredLifetime"), Constructors: []ast.DatatypeConstructorDefinition{{Name: protoSymbol(t, m, "structured-only"), Recognizer: protoSymbol(t, m, "is-structured-only")}}}})
		protoMust(t, err)
		sort = sorts[0]
	}
	protoPin(t, m, sort)
	protoMust(t, p.RegisterFactory(f))
	return m, f, base, sort
}

func TestStructuredFactoryClosedRegistryCannotRevive(t *testing.T) {
	for _, kind := range []string{"array", "datatype"} {
		t.Run(kind, func(t *testing.T) {
			m, f, base, sort := structuredOwnerFixture(t, kind)
			value, err := m.MkConst(protoSymbol(t, m, "held-structured-value"), sort)
			protoMust(t, err)
			protoPin(t, m, value)
			protoMust(t, f.RegisterValue(value))
			before, err := m.GetRefCount(value)
			protoMust(t, err)
			protoMust(t, f.Close())
			after, err := m.GetRefCount(value)
			protoMust(t, err)
			if after+1 != before || base.manager != nil || base.model != nil || base.sets != nil || len(base.values) != 0 || len(base.sorts) != 0 {
				t.Fatal("factory retained owner state or consumed the caller reference")
			}
			refs := m.Stats().LiveReferences
			if _, err := f.SomeValue(sort); !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("closed SomeValue", err)
			}
			first, second := protoTestOutputRefs(t, m)
			afterOutputs := m.Stats().LiveReferences
			if ok, err := f.SomeValues(sort, first, second); ok || !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("closed SomeValues", ok, err)
			}
			if _, err := f.FreshValue(sort); !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("closed FreshValue", err)
			}
			if err := f.RegisterValue(value); !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("closed RegisterValue", err)
			}
			protoMust(t, f.Close())
			if m.Stats().LiveReferences != afterOutputs || afterOutputs != refs+2 {
				t.Fatal("closed operations acquired new source references")
			}
		})
	}
}

func TestStructuredFactoryValueSetActualAssertionOwner(t *testing.T) {
	for _, kind := range []string{"array", "datatype"} {
		for _, warm := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", kind, warm), func(t *testing.T) {
				m, _, base, sort := structuredOwnerFixture(t, kind)
				if warm {
					_, err := base.valueSet(sort)
					protoMust(t, err)
				}
				live := diagnostic.BorrowProcessDebugContext()
				d, _ := scalarFactoryDebugActions(t, "T")
				protoMust(t, d.Close())
				set, err := base.valueSet(sort)
				if diagnostic.DebugBuild {
					if !errors.Is(err, arena.ErrClosed) || set != nil {
						t.Fatal("line30 must consult the actual assertion owner", set, err)
					}
				} else {
					protoMust(t, err)
					if set == nil {
						t.Fatal("erased source assertion lost the set")
					}
				}
				if len(base.sorts) != 1 || base.sets[sort] == nil {
					t.Fatal("source map/sort publication prefix was rolled back")
				}
				_, err = diagnostic.BindProcessDebugContext(live)
				protoMust(t, err)
				count, err := m.GetRefCount(sort)
				protoMust(t, err)
				if count < 2 {
					t.Fatal("factory lost its sort acquisition")
				}
			})
		}
	}
}

type structuredCloseDuringTrace struct {
	bytes.Buffer
	factory   ValueFactory
	attempted bool
	failure   error
}

func (w *structuredCloseDuringTrace) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *structuredCloseDuringTrace) Write(p []byte) (int, error) {
	if !w.attempted {
		w.attempted = true
		w.failure = w.factory.Close()
	}
	return w.Buffer.Write(p)
}

func TestStructuredFactoryRetryRetainsUnconsumedReferences(t *testing.T) {
	owner, _, config := protoFixtureWithConfig(t)
	output := &structuredCloseDuringTrace{}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: owner.Symbols, Numerals: owner.NumeralService(), Budget: owner.Budget(), TraceTags: tags})
	protoMust(t, err)
	protoMust(t, m.RegisterDeclarationPlugins())
	t.Cleanup(func() { protoMust(t, m.Close()) })
	md, err := NewProtoModel(m, config)
	protoMust(t, err)
	t.Cleanup(func() { protoMust(t, md.Close()) })
	f, err := NewArrayFactory(md)
	protoMust(t, err)
	protoMust(t, md.RegisterFactory(f))
	output.factory = f
	boolean, err := m.MkBoolSort()
	protoMust(t, err)
	sort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	protoMust(t, err)
	var values []ast.ASTHandle
	for i := 0; i < 2; i++ {
		value, err := m.MkConst(protoSymbol(t, m, fmt.Sprintf("unconsumed-owner-%d", i)), sort)
		protoMust(t, err)
		protoMust(t, f.RegisterValue(value))
		values = append(values, value)
	}
	sorts := append([]ast.ASTRef(nil), f.sorts...)
	refs := append([]ast.ASTRef(nil), f.values...)
	floatSort, err := m.FloatingPointSort(3, 4)
	protoMust(t, err)
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		return
	}
	// The native special-value constructor uses the existing guarded numerical
	// mutation scope. Its real AST trace callback reaches public DelRef while
	// that guard is active, without consuming any reference token.
	_, err = m.MkFPAZeroOfSort(floatSort, false)
	protoMust(t, err)
	if !output.attempted || !errors.Is(output.failure, ast.ErrASTPending) {
		t.Fatal("did not reach unconsumed manager guard", output.attempted, output.failure)
	}
	if f.sets != nil || f.manager != m {
		t.Fatal("failed Close lost its continuation or left registry active")
	}
	for i, ref := range sorts {
		if f.sorts[i] != ref {
			t.Fatal("unconsumed sort token changed")
		}
	}
	for i, ref := range refs {
		if f.values[i] != ref {
			t.Fatal("unconsumed value token changed")
		}
	}
	if err = f.RegisterValue(values[0]); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("half-closed registry revived", err)
	}
	protoMust(t, f.Close())
	after := output.String()
	protoMust(t, f.Close())
	if output.String() != after || f.manager != nil {
		t.Fatal("retry replayed a completed deletion")
	}
	for _, value := range values {
		if _, err := m.GetKind(value); err == nil {
			t.Fatal("retry leaked its unconsumed value")
		}
	}
}
