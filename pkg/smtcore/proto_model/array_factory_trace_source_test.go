package protomodel

// Source: src/smt/proto_model/array_factory.cpp:64,79,96,108-109,117,152,163,203.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func arrayFactoryTraceFixture(t *testing.T, out io.Writer) (*ast.ASTManager, *ProtoModel, *ArrayFactory, *diagnostic.TraceTags) {
	t.Helper()
	owner, _, config := protoFixtureWithConfig(t)
	previous, err := uparams.BindProcessGlobalParamsService(config.Globals)
	protoMust(t, err)
	t.Cleanup(func() {
		_, err := uparams.BindProcessGlobalParamsService(previous)
		protoMust(t, err)
	})
	tags := diagnostic.NewTraceTagsWithWriter(out)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{
		Symbols: owner.Symbols, Numerals: owner.NumeralService(), Budget: owner.Budget(), TraceTags: tags,
	})
	protoMust(t, err)
	protoMust(t, m.RegisterDeclarationPlugins())
	t.Cleanup(func() { protoMust(t, m.Close()) })
	p, err := NewProtoModel(m, config)
	protoMust(t, err)
	t.Cleanup(func() { protoMust(t, p.Close()) })
	f, err := NewArrayFactory(p)
	protoMust(t, err)
	protoMust(t, p.RegisterFactory(f))
	return m, p, f, tags
}

func TestArrayFactorySourceTraces(t *testing.T) {
	var out bytes.Buffer
	m, _, f, tags := arrayFactoryTraceFixture(t, &out)
	tags.Enable("array_factory")
	tags.Enable("array_factory_bug")
	boolean, err := m.MkBoolSort()
	protoMust(t, err)
	domain, err := m.MkUninterpretedSort(protoSymbol(t, m, "ArrayTraceDomain"), nil)
	protoMust(t, err)
	protoPin(t, m, domain)
	sort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{domain}), boolean)
	protoMust(t, err)
	protoPin(t, m, sort)
	value, err := f.SomeValue(sort)
	protoMust(t, err)
	if value.IsNull() {
		t.Fatal("SomeValue refused an array")
	}
	fresh, err := f.FreshValue(sort)
	protoMust(t, err)
	if fresh.IsNull() {
		t.Fatal("fresh domain indices did not produce a fresh array")
	}

	finiteSort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	protoMust(t, err)
	protoPin(t, m, finiteSort)
	_, err = f.SomeValue(finiteSort)
	protoMust(t, err)
	fresh, err = f.FreshValue(finiteSort)
	protoMust(t, err)
	if !fresh.IsNull() {
		t.Fatal("finite domain unexpectedly produced a fresh array")
	}
	if !tags.ShouldTrace("array_factory") || !tags.ShouldTrace("array_factory_bug") {
		if out.Len() != 0 {
			t.Fatal("erased array traces emitted output")
		}
		return
	}
	for _, want := range []string{
		"[array_factory] get_some_value src/smt/proto_model/array_factory.cpp:64",
		"[array_factory_bug] get_fresh_value src/smt/proto_model/array_factory.cpp:152",
		"array fresh value: using fresh index",
		"[array_factory_bug] get_fresh_value src/smt/proto_model/array_factory.cpp:163",
		"v1:",
		"[array_factory_bug] get_fresh_value src/smt/proto_model/array_factory.cpp:203",
		"failed to build fresh array value",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing source trace %q in\n%s", want, out.String())
		}
	}
}

func TestArrayFactoryTraceBadbitDoesNotChangeSomeValue(t *testing.T) {
	w := &datatypeFactoryTraceStopWriter{at: "get_some_value src/smt/proto_model/array_factory.cpp:64", action: func() {}, refuse: true}
	m, _, f, tags := arrayFactoryTraceFixture(t, w)
	tags.Enable("array_factory")
	boolean, err := m.MkBoolSort()
	protoMust(t, err)
	sort, err := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	protoMust(t, err)
	protoPin(t, m, sort)
	value, err := f.SomeValue(sort)
	protoMust(t, err)
	if value.IsNull() || len(f.sets[sort].values) != 1 {
		t.Fatal("trace stream refusal changed SomeValue publication")
	}
	if tags.ShouldTrace("array_factory") && !w.failed {
		t.Fatal("test writer did not refuse the source trace")
	}
}

func TestArrayFactorySourceAssertionGates(t *testing.T) {
	sites := []struct {
		line      int32
		condition string
	}{
		{79, "!m_sort2value_set.find(s, set) || set->size() == 0"},
		{96, "m_sort2value_set.find(s, set) && set->size() == 2"},
		{108, "set != 0"},
		{109, "set->size() > 0"},
		{117, "set->size() >= 2"},
	}
	for _, enabled := range []bool{false, true} {
		for _, site := range sites {
			t.Run(fmt.Sprintf("%t/%d", enabled, site.line), func(t *testing.T) {
				d, _ := scalarFactoryDebugActions(t, "T")
				protoMust(t, d.EnableAssertions(enabled))
				evaluated := false
				err := valueFactoryAssert("src/smt/proto_model/array_factory.cpp", site.line, site.condition, func() bool {
					evaluated = true
					return false
				})
				if diagnostic.DebugBuild && enabled {
					var ex z3exception.DefaultException
					if !errors.As(err, &ex) || ex.Message() != "assertion violation" || !evaluated {
						t.Fatal("enabled source assertion", err, evaluated)
					}
					return
				}
				protoMust(t, err)
				if evaluated {
					t.Fatal("erased or disabled source assertion evaluated its condition")
				}
			})
		}
	}
}
