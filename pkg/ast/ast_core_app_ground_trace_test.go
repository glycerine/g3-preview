package ast

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

type astAppTraceRefusingWriter struct{}

func (astAppTraceRefusingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestASTAppGroundTraceAfterApplicationConstruction(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	sort, err := m.MkBoolSort()
	astTestMust(t, err)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, variable)
	constant, err := m.MkConst(astTestSymbol(t, m, "ground-const"), sort)
	astTestMust(t, err)
	astTestPin(t, m, constant)
	pair, err := m.MkIff(variable, constant)
	astTestMust(t, err)
	astTestPin(t, m, pair)
	decl, err := m.GetDecl(pair)
	astTestMust(t, err)
	tags.Enable("app_ground")
	result, err := m.MkApp(decl, []ASTHandle{variable, constant})
	astTestMust(t, err)
	if result != pair {
		t.Fatal("source structural interning", result, pair)
	}
	if tags.ShouldTrace("app_ground") {
		id, err := m.GetASTID(result)
		astTestMust(t, err)
		printer, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
		astTestMust(t, err)
		astTestMust(t, m.DisplayLowLevel(printer, result, true, true))
		printed, err := printer.String()
		astTestMust(t, err)
		astTestMust(t, printer.Close())
		want := "-------- [app_ground] mk_app src/ast/ast.cpp:2207 ---------\n" +
			"ground: 0 id: " + strconv.FormatUint(uint64(id), 10) + "\n" + printed + "\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("source app-ground record: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased app-ground trace emitted", output.String())
	}
	output.Reset()
	if _, err := m.MkApp(decl, []ASTHandle{constant, constant}); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("app_ground") {
		if got := output.String(); !strings.Contains(got, "ground: 1 id: ") || strings.Count(got, "[app_ground] mk_app") != 1 {
			t.Fatal("ground result record", got)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased ground result emitted", output.String())
	}
	output.Reset()
	tags.Enable("chainable")
	if _, err := m.MkApp(decl, []ASTHandle{variable, constant, variable}); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("app_ground") {
		got := output.String()
		chain := strings.Index(got, "[chainable] mk_app")
		first := strings.Index(got, "[app_ground] mk_app")
		last := strings.LastIndex(got, "[app_ground] mk_app")
		if chain < 0 || first <= chain || last <= first || strings.Count(got, "[app_ground] mk_app") != 2 {
			t.Fatal("nested conjunction and outer application trace order", got)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased nested application trace emitted", output.String())
	}
	refused := udiag.NewTraceTagsWithWriter(astAppTraceRefusingWriter{})
	refused.Enable("app_ground")
	m.traceTags = refused
	if again, err := m.MkApp(decl, []ASTHandle{variable, constant}); err != nil || again != result {
		t.Fatal("borrowed trace stream refusal changed application", err, again)
	}
}
