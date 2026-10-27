package protomodel

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestDatatypeFactoryUtilityStopsBeforeBaseTeardown(t *testing.T) {
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
	f, err := NewDatatypeFactory(md)
	protoMust(t, err)
	protoMust(t, md.RegisterFactory(f))
	output.factory = f
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: protoSymbol(t, m, "MemberBeforeBase"), Constructors: []ast.DatatypeConstructorDefinition{{Name: protoSymbol(t, m, "member-only"), Recognizer: protoSymbol(t, m, "is-member-only")}}}})
	protoMust(t, err)
	sort := sorts[0]
	protoPin(t, m, sort)
	// SomeValue populates both the real datatype utility and base registry.
	value, err := f.SomeValue(sort)
	protoMust(t, err)
	protoPin(t, m, value)
	first, second := protoTestOutputRefs(t, m)
	sortRefs := append([]ast.ASTRef(nil), f.sorts...)
	valueRefs := append([]ast.ASTRef(nil), f.values...)
	floatSort, err := m.FloatingPointSort(3, 4)
	protoMust(t, err)
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		t.Skip("requires source trace build")
	}
	_, err = m.MkFPAZeroOfSort(floatSort, false)
	protoMust(t, err)
	if !output.attempted || !errors.Is(output.failure, ast.ErrASTPending) {
		t.Fatal("utility did not stop at the actual manager guard", output.attempted, output.failure)
	}
	if f.last != nil || f.sets == nil || f.manager != m || !reflect.DeepEqual(f.sorts, sortRefs) || !reflect.DeepEqual(f.values, valueRefs) {
		t.Fatal("base teardown ran before utility member completed")
	}
	if _, err := f.SomeValue(sort); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closing SomeValue", err)
	}
	if _, err := f.FreshValue(sort); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closing FreshValue", err)
	}
	if ok, err := f.SomeValues(sort, first, second); ok || !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closing SomeValues", ok, err)
	}
	if err := f.RegisterValue(value); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closing RegisterValue", err)
	}
	before, err := m.GetRefCount(value)
	protoMust(t, err)
	protoMust(t, f.Close())
	after, err := m.GetRefCount(value)
	protoMust(t, err)
	if after+1 != before || f.manager != nil || f.sets != nil || len(f.sorts) != 0 || len(f.values) != 0 {
		t.Fatal("retry did not retire base owners exactly once")
	}
	protoMust(t, f.Close())
}
