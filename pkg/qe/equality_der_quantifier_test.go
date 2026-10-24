package qe_test

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestEqualityDERQuantifierCorePatternsParameterSnapshotAndProofs(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, ignore := range []bool{false, true} {
			t.Run(fmt.Sprintf("proofs=%v/ignore=%v", proofs, ignore), func(t *testing.T) {
				f := setupProofs(t, proofs)
				x, y := f.variable(t, 0, f.integer), f.variable(t, 1, f.integer)
				trigger := termFunction(t, f, "quantifier-trigger", f.integer, y, x)
				pattern, err := f.m.MkPattern([]ast.ASTHandle{trigger})
				must(t, err)
				f.pin(t, pattern)
				names := []usymbol.Symbol{}
				for _, s := range []string{"eliminated", "remaining"} {
					n, err := f.m.Symbols.Intern(s)
					must(t, err)
					names = append(names, n)
				}
				q, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer, f.integer}, Names: names, Body: f.equality(t, y, x), Patterns: []ast.ASTHandle{pattern}})
				must(t, err)
				f.pin(t, q)
				params, err := f.config.Context.NewValue()
				must(t, err)
				t.Cleanup(func() { must(t, f.config.Context.Del(&params)) })
				key := "ignore_patterns_on_ground_qbody"
				must(t, f.config.Context.SetBoolName(&params, &key, ignore))
				d := equalityReducer(t, f, &params, nil)
				must(t, f.config.Context.SetBoolName(&params, &key, !ignore))
				refs := f.refs(t, ast.ASTHandle{}, ast.ASTHandle{})
				must(t, d.ReduceQuantifier(q, refs[0], refs[1]))
				core := f.get(t, refs[0])
				n, err := f.m.GetNumDeclarations(core)
				must(t, err)
				if n != 2 {
					t.Fatal("core removed declaration slots before constant un-abstraction")
				}
				body, err := f.m.GetQuantifierBody(core)
				must(t, err)
				if !f.m.IsTrue(body) {
					t.Fatal("core did not remove defining equality")
				}
				pat, err := f.m.GetPattern(core, 0)
				must(t, err)
				call := f.arg(t, pat, 0)
				if f.arg(t, call, 0) != x || f.arg(t, call, 1) != x {
					t.Fatal("pattern substitution lost remaining binder")
				}
				must(t, d.ApplyQuantifier(q, refs[0], refs[1]))
				must(t, d.Close())
				got := f.get(t, refs[0])
				if ignore {
					if !f.m.IsTrue(got) {
						t.Fatal("constructor parameter snapshot was not retained")
					}
				} else {
					n, err := f.m.GetNumDeclarations(got)
					must(t, err)
					if n != 1 {
						t.Fatal("pattern-only variable was discarded", n)
					}
				}
				proof := f.get(t, refs[1])
				if proofs {
					fact, err := f.m.GetProofFact(proof)
					must(t, err)
					if f.arg(t, fact, 0) != q || f.arg(t, fact, 1) != got {
						t.Fatal("quantifier proof chain lost endpoints")
					}
				} else if !proof.IsNull() {
					t.Fatal("proof created with proofs disabled")
				}
			})
		}
	}
}

func TestEqualityDERUniversalDisequalitySourceRefusalAndIdentityProof(t *testing.T) {
	f := setupProofs(t, true)
	x := f.variable(t, 0, f.integer)
	three := f.numeral(t, "3")
	body, err := f.m.MkOrArray([]ast.ASTHandle{f.not(t, f.equality(t, x, three)), predicate(t, f, "universal-p", x)})
	must(t, err)
	f.pin(t, body)
	name, err := f.m.Symbols.Intern("universal-x")
	must(t, err)
	q, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: body})
	must(t, err)
	f.pin(t, q)
	identity, err := f.m.MkProofReflexivity(q)
	must(t, err)
	f.pin(t, identity)
	refs := f.refs(t, ast.ASTHandle{}, identity)
	d := equalityReducer(t, f, nil, nil)
	must(t, d.ReduceQuantifier(q, refs[0], refs[1]))
	if f.get(t, refs[0]) != q || f.get(t, refs[1]) != identity {
		t.Fatal("source core refusal changed input/proof")
	}
	must(t, d.ApplyQuantifier(q, refs[0], refs[1]))
	if f.get(t, refs[0]) != q || !f.get(t, refs[1]).IsNull() {
		t.Fatal("operator() did not reset proof on identity")
	}
}

func TestEqualityDERDatatypeFlatteningAndConstructorRefusal(t *testing.T) {
	for _, mode := range []string{"same-constructor", "different-constructor", "recognizer-accessors"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			sym := func(s string) usymbol.Symbol { n, err := f.m.Symbols.Intern(s); must(t, err); return n }
			sorts, err := f.m.MkDatatypes([]ast.DatatypeDefinition{{Name: sym("DERPair"), Constructors: []ast.DatatypeConstructorDefinition{
				{Name: sym("empty-pair"), Recognizer: sym("is-empty-pair")},
				{Name: sym("pair"), Recognizer: sym("is-pair"), Accessors: []ast.DatatypeAccessorDefinition{{Name: sym("first"), Range: f.integer}, {Name: sym("second"), Range: f.integer}}},
			}}})
			must(t, err)
			f.pin(t, sorts[0])
			constructors, err := f.m.DatatypeConstructors(sorts[0])
			must(t, err)
			x, y := f.variable(t, 0, f.integer), f.variable(t, 1, f.integer)
			left, err := f.m.MkApp(constructors[1], []ast.ASTHandle{x, y})
			must(t, err)
			f.pin(t, left)
			right, err := f.m.MkApp(constructors[1], []ast.ASTHandle{f.numeral(t, "3"), f.numeral(t, "5")})
			must(t, err)
			f.pin(t, right)
			if mode == "different-constructor" {
				right, err = f.m.MkApp(constructors[0], nil)
				must(t, err)
				f.pin(t, right)
			}
			if mode == "recognizer-accessors" {
				right = f.constant(t, "unknown-pair", sorts[0])
			}
			conjs := conjunctions(t, f, f.equality(t, left, right), predicate(t, f, "fields-p", x, y))
			d := equalityReducer(t, f, nil, selector(t, f, []uint32{0, 1}, true))
			must(t, d.EliminateConjunctions(conjs))
			if mode == "different-constructor" {
				if len(*conjs) != 1 || !f.m.IsFalse(f.get(t, (*conjs)[0])) {
					t.Fatal("distinct constructors did not refute equality")
				}
				return
			}
			first, second := f.numeral(t, "3"), f.numeral(t, "5")
			if mode == "recognizer-accessors" {
				accessors, err := f.m.DatatypeConstructorAccessors(constructors[1])
				must(t, err)
				first, err = f.m.MkApp(accessors[0], []ast.ASTHandle{right})
				must(t, err)
				f.pin(t, first)
				second, err = f.m.MkApp(accessors[1], []ast.ASTHandle{right})
				must(t, err)
				f.pin(t, second)
			}
			want := predicate(t, f, "fields-p", first, second)
			found := false
			for _, r := range *conjs {
				if f.get(t, r) == want {
					found = true
				}
			}
			if !found {
				t.Fatal("datatype fields not substituted")
			}
			if mode == "recognizer-accessors" && len(*conjs) != 2 {
				t.Fatal("constructor recognizer constraint was lost")
			}
		})
	}
}
