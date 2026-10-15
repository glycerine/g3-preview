package ast

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTQuantifierPatternSourceTraceBeforeRegistration(t *testing.T) {
	c, g, _, budget := dynAckSourceFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: budget, TraceTags: tags})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.Close()) }()
	previous, err := uparams.BindProcessGlobalParamsService(g)
	paramsMust(t, err)
	defer func() { _, e := uparams.BindProcessGlobalParamsService(previous); paramsMust(t, e) }()
	sort, err := m.MkBoolSort()
	paramsMust(t, err)
	truth, err := m.MkFamilyApp(0, ASTOpTrue, nil, nil)
	paramsMust(t, err)
	pattern, err := m.MkFamilyApp(2, 0, nil, []ASTHandle{truth})
	paramsMust(t, err)
	tags.Enable("ast")
	spec := ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{sort}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: truth, Patterns: []ASTHandle{pattern}}
	q, err := m.MkQuantifier(spec)
	paramsMust(t, err)
	if q == (ASTHandle{}) {
		t.Fatal("missing quantifier")
	}
	if ASTSourceDebug && tags.ShouldTrace("ast") {
		got := output.String()
		if !strings.HasPrefix(got, "-------- [ast] mk_quantifier src/ast/ast.cpp:2358 ---------\n0 ") ||
			!strings.Contains(got, "(true)\n------------------------------------------------\n") ||
			strings.Index(got, "(true)\n------------------------------------------------\n") > strings.LastIndex(got, "register_node_core") {
			t.Fatalf("pattern trace missing or late: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased pattern trace emitted %q", output.String())
	}
	output.Reset()
	spec.Patterns = []ASTHandle{truth}
	_, err = m.MkQuantifier(spec)
	if ASTSourceDebug {
		if err != ErrASTDomain {
			t.Fatalf("invalid pattern was not refused: %v", err)
		}
		if tags.ShouldTrace("ast") && (!strings.HasPrefix(output.String(), "-------- [ast] mk_quantifier src/ast/ast.cpp:2358 ---------\n0 ") ||
			!strings.HasSuffix(output.String(), "true\n------------------------------------------------\n")) {
			t.Fatalf("invalid pattern was not traced before refusal: %q", output.String())
		}
	} else {
		paramsMust(t, err)
	}
	output.Reset()
	variable, err := m.MkVar(0, sort)
	paramsMust(t, err)
	badPattern, err := m.MkFamilyApp(2, 0, nil, []ASTHandle{variable})
	paramsMust(t, err)
	output.Reset()
	spec.Patterns = []ASTHandle{badPattern}
	_, err = m.MkQuantifier(spec)
	if ASTSourceDebug {
		if err != ErrASTDomain {
			t.Fatalf("pattern with variable child was not refused after TRACE: %v, output %q", err, output.String())
		}
		if tags.ShouldTrace("ast") && !strings.HasSuffix(output.String(), "(pattern (:var 0))\n------------------------------------------------\n") {
			t.Fatalf("pattern fallback source TRACE missing: %q", output.String())
		}
	} else {
		paramsMust(t, err)
	}
}
