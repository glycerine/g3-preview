package qe_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
)

func TestEqualityDERCPPProjectionNormalForms(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, mode := range []string{"chain", "cycle", "value-preference", "constant-preference", "integer-minus-one", "real-fraction", "ite"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			sort, sortText := f.integer, "Int"
			if mode == "real-fraction" {
				family, err := f.m.ArithmeticFamilyID()
				must(t, err)
				sort, err = f.m.MkFamilySort(family, 0, nil)
				must(t, err)
				f.pin(t, sort)
				sortText = "Real"
			}
			num := func(text string) ast.ASTHandle {
				n, err := f.m.NumeralService().NewValue()
				must(t, err)
				must(t, f.m.NumeralService().SetString(&n, text))
				h, err := f.m.ArithmeticNumeral(&n, sortText == "Int")
				must(t, err)
				f.pin(t, h)
				must(t, f.m.NumeralService().Del(&n))
				return h
			}
			x, y := f.variable(t, 0, sort), f.variable(t, 1, sort)
			three := num("3")
			forms := []ast.ASTHandle{f.equality(t, x, y), f.equality(t, y, three), predicate(t, f, "projection_p", x, y)}
			indices := []uint32{0, 1}
			declarations := "(declare-fun projection_p (" + sortText + " " + sortText + ") Bool)\n"
			switch mode {
			case "cycle":
				forms[1] = f.equality(t, y, x)
			case "value-preference", "constant-preference":
				k := f.constant(t, "projection_k", sort)
				forms = []ast.ASTHandle{f.equality(t, x, termFunction(t, f, "projection_f", sort, y)), f.equality(t, x, k), predicate(t, f, "projection_p", x, y)}
				declarations += "(declare-const projection_k " + sortText + ")\n(declare-fun projection_f (" + sortText + ") " + sortText + ")\n"
				if mode == "value-preference" {
					forms = append(forms, f.equality(t, x, three))
				}
				indices = []uint32{0}
			case "integer-minus-one", "real-fraction":
				coefficient := "-1"
				if mode == "real-fraction" {
					coefficient = "7/3"
				}
				product, err := f.m.MkArithmeticMul(num(coefficient), x)
				must(t, err)
				f.pin(t, product)
				forms[0] = f.equality(t, product, y)
			case "ite":
				condition := f.constant(t, "projection_condition", f.boolean)
				choice, err := f.m.MkIte(condition, f.equality(t, x, three), f.equality(t, x, num("5")))
				must(t, err)
				f.pin(t, choice)
				forms[0] = choice
				declarations += "(declare-const projection_condition Bool)\n"
			}
			// Public QE-lite first flattens AND by replacing the parent slot
			// with its last child. Rotate the source input to recover the raw
			// eq::der vector order used here.
			sourceForms := append([]ast.ASTHandle(nil), forms[1:]...)
			sourceForms = append(sourceForms, forms[0])
			body, err := f.m.MkAndArray(sourceForms)
			must(t, err)
			f.pin(t, body)
			original := exists(t, f, []ast.ASTHandle{sort, sort}, body)
			conjs := conjunctions(t, f, forms...)
			d := equalityReducer(t, f, nil, selector(t, f, indices, true))
			must(t, d.EliminateConjunctions(conjs))
			var outputs []ast.ASTHandle
			for _, r := range *conjs {
				outputs = append(outputs, f.get(t, r))
			}
			body, err = f.m.MkAndArray(outputs)
			must(t, err)
			f.pin(t, body)
			projected := exists(t, f, []ast.ASTHandle{sort, sort}, body)
			normalize := func(h ast.ASTHandle, strategy string) string {
				t.Helper()
				script := "(set-logic ALL)\n" + declarations + "(assert " + render(t, f, h) + ")\n(apply " + strategy + ")\n"
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				var out, diagnostics bytes.Buffer
				cmd.Stdout, cmd.Stderr = &out, &diagnostics
				if err := cmd.Run(); err != nil {
					t.Fatal(err, out.String(), diagnostics.String(), script)
				}
				return regexp.MustCompile(`:depth [0-9]+`).ReplaceAllString(strings.Join(strings.Fields(out.String()), " "), ":depth")
			}
			want, got := normalize(original, "(then qe-light simplify)"), normalize(projected, "simplify")
			if got != want {
				t.Fatalf("Go equality projection/source QE-lite mismatch\nGo phase: %s\nC++ QE-lite: %s", got, want)
			}
		})
	}
}
