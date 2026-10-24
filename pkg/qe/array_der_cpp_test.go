package qe_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func render(t *testing.T, f *fixture, h ast.ASTHandle) string {
	t.Helper()
	format, err := f.m.SMT2PPExprTree(h, nil)
	must(t, err)
	fm, err := f.m.GetFormatManager()
	must(t, err)
	ref, err := fm.NewRef(format)
	must(t, err)
	defer func() { must(t, fm.DelRef(&ref)) }()
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: f.budget})
	must(t, err)
	defer func() { must(t, stream.Close()) }()
	p := f.config.Context.EmptyValue()
	must(t, f.m.PrintFormat(stream, format, f.config.Context, &p, f.config.Globals))
	text, err := stream.String()
	must(t, err)
	return text
}

func exists(t *testing.T, f *fixture, sorts []ast.ASTHandle, body ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	var names []usymbol.Symbol
	for i := range sorts {
		name, err := f.m.Symbols.Intern(fmt.Sprintf("eliminate_%d", i))
		must(t, err)
		names = append(names, name)
	}
	h, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: sorts, Names: names, Body: body})
	must(t, err)
	return f.pin(t, h)
}

func TestArrayDERCPPProjectionNormalForms(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"select", "select-reverse", "select-multiple-indices", "select-nested-quantifier", "disequality", "disequality-multiple-indices"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			multi := strings.Contains(mode, "multiple-indices")
			domains := []ast.ASTHandle{f.integer}
			arrayText := "(Array Int Int)"
			if multi {
				domains = append(domains, f.integer)
				arrayText = "(Array Int Int Int)"
			}
			sort := f.arraySort(t, domains, f.integer)
			var forms []ast.ASTHandle
			var boundSorts []ast.ASTHandle
			var declarations string
			indices := []uint32{0}
			if strings.HasPrefix(mode, "select") {
				a := f.variable(t, 0, sort)
				key := f.constant(t, "key", f.integer)
				other := f.constant(t, "other", f.integer)
				args, next := []ast.ASTHandle{a, key}, []ast.ASTHandle{a, other}
				if multi {
					args = append(args, other)
					next = append(next, key)
				}
				seven, nine := f.numeral(t, "7"), f.numeral(t, "9")
				definition := f.equality(t, f.selectArray(t, args...), seven)
				if mode == "select-reverse" {
					definition = f.equality(t, seven, f.selectArray(t, args...))
				}
				remaining := f.equality(t, f.selectArray(t, next...), nine)
				if mode == "select-nested-quantifier" {
					shifted := f.variable(t, 1, sort)
					name, err := f.m.Symbols.Intern("inner_unused")
					must(t, err)
					remaining, err = f.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: f.equality(t, f.selectArray(t, shifted, other), nine)})
					must(t, err)
					f.pin(t, remaining)
				}
				forms = []ast.ASTHandle{definition, remaining}
				boundSorts = []ast.ASTHandle{sort}
				declarations = "(declare-const key Int)\n(declare-const other Int)\n"
			} else {
				a, b := f.constant(t, "array_a", sort), f.constant(t, "array_b", sort)
				x := f.variable(t, 0, f.integer)
				args, other := []ast.ASTHandle{a, x}, []ast.ASTHandle{b, x}
				boundSorts = []ast.ASTHandle{f.integer}
				if multi {
					y := f.variable(t, 1, f.integer)
					args = append(args, y)
					other = append(other, y)
					indices = append(indices, 1)
					boundSorts = append(boundSorts, f.integer)
				}
				forms = []ast.ASTHandle{f.not(t, f.equality(t, f.selectArray(t, args...), f.selectArray(t, other...)))}
				declarations = "(declare-const array_a " + arrayText + ")\n(declare-const array_b " + arrayText + ")\n"
			}
			// Source flatten_and replaces its input slot with the last
			// appended child. With these two-child inputs, reverse the input
			// conjunction so ar::der receives exactly the native phase order.
			sourceForms := append([]ast.ASTHandle(nil), forms...)
			if len(sourceForms) == 2 {
				sourceForms[0], sourceForms[1] = sourceForms[1], sourceForms[0]
			}
			body, err := f.m.MkAndArray(sourceForms)
			must(t, err)
			f.pin(t, body)
			original := exists(t, f, boundSorts, body)
			conjs := f.refs(t, forms...)
			d := reducer(t, f, selector(t, f, indices, true))
			must(t, d.Apply(conjs))
			must(t, d.Close())
			var outputs []ast.ASTHandle
			for _, ref := range conjs {
				outputs = append(outputs, f.get(t, ref))
			}
			afterBody, err := f.m.MkAndArray(outputs)
			must(t, err)
			f.pin(t, afterBody)
			projected := exists(t, f, boundSorts, afterBody)
			// C++ SMT can return UNKNOWN on the raw quantified-array
			// equivalence query. Compare the actual source QE-lite result to
			// the Go array pass followed only by source simplification. The
			// latter receives no additional elimination pass to repair it.
			normalize := func(form ast.ASTHandle, strategy string) string {
				t.Helper()
				script := "(set-logic ALL)\n" + declarations + "(assert " + render(t, f, form) + ")\n(apply " + strategy + ")\n"
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				var out, diagnostics bytes.Buffer
				cmd.Stdout = &out
				cmd.Stderr = &diagnostics
				if err := cmd.Run(); err != nil {
					t.Fatal(err, out.String(), diagnostics.String(), script)
				}
				text := strings.Join(strings.Fields(out.String()), " ")
				// QE-lite plus simplify and simplify alone have different
				// tactic depths; preserve all formula and precision content.
				return regexp.MustCompile(`:depth [0-9]+`).ReplaceAllString(text, ":depth")
			}
			want := normalize(original, "(then qe-light simplify)")
			got := normalize(projected, "simplify")
			if got != want {
				t.Fatalf("Go array projection/source QE-lite mismatch\nGo phase: %s\nC++ QE-lite: %s", got, want)
			}
		})
	}
}
