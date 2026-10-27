package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTContextDisplayClauseRegionsAndPackedWatchOrder(t *testing.T) {
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	astTestMust(t, s.SetBinaryClauseOptimization(true))
	var literals []SMTLiteral
	var ids []uint32
	for _, name := range []string{"dump-a", "dump-b", "dump-c"} {
		term := smtRelevancyBool(t, m, name)
		astTestMust(t, s.Internalize(term, true))
		v, _ := s.BoolVar(term)
		literals = append(literals, NewSMTLiteral(v, false))
		id, err := m.GetASTID(term)
		astTestMust(t, err)
		ids = append(ids, id)
	}
	a, b, c := literals[0], literals[1], literals[2]
	astTestMust(t, s.addClause([]SMTLiteral{a, c}, true, ast.ASTHandle{}))
	astTestMust(t, s.addClause([]SMTLiteral{a, b.Negated()}, true, ast.ASTHandle{}))
	astTestMust(t, s.addClause([]SMTLiteral{a, b, c}, true, ast.ASTHandle{}))
	astTestMust(t, s.addClauseKind([]SMTLiteral{a.Negated(), b, c.Negated()}, true, SMTClauseAuxLemma, ast.ASTHandle{}))
	var out bytes.Buffer
	astTestMust(t, s.DisplayBinaryClauses(&out))
	binary := fmt.Sprintf("binary clauses:\n(clause #%d (not #%d))\n(clause #%d #%d)\n", ids[0], ids[1], ids[0], ids[2])
	if out.String() != binary {
		t.Fatal(out.String(), binary)
	}
	out.Reset()
	astTestMust(t, s.DisplayClauseSections(&out))
	// Source selects the last undefined watch twice, swapping C then A
	// to the front; display preserves the resulting stored literal order.
	want := fmt.Sprintf("auxiliary clauses:\n(clause #%d #%d #%d)\nlemmas:\n(clause (not #%d) (not #%d) #%d)\n", ids[0], ids[1], ids[2], ids[2], ids[0], ids[1])
	if out.String() != want {
		t.Fatal(out.String(), want)
	}
	// Scoped binary clauses use ordinary clause objects and belong only to
	// the auxiliary vector, while base packed clauses remain in watches.
	astTestMust(t, s.PushScope())
	astTestMust(t, s.addClause([]SMTLiteral{a.Negated(), b}, true, ast.ASTHandle{}))
	out.Reset()
	astTestMust(t, s.DisplayClauseSections(&out))
	scoped := fmt.Sprintf("(clause (not #%d) #%d)\n", ids[0], ids[1])
	if !bytes.Contains(out.Bytes(), []byte(scoped)) {
		t.Fatal("scoped ordinary binary clause missing", out.String())
	}
	astTestMust(t, s.PopScopes(1))
	out.Reset()
	astTestMust(t, s.DisplayClauseSections(&out))
	if out.String() != want {
		t.Fatal("popped clause still displayed", out.String())
	}
	out.Reset()
	astTestMust(t, s.DisplayBinaryClauses(&out))
	if out.String() != binary {
		t.Fatal("scope changed packed clauses", out.String())
	}
}

func TestSMTContextDisplayDeclarationRegistryFiltersAndSorts(t *testing.T) {
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	sort, terms := smtUFConstants(t, m, "dump-registry-", 2)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "dump-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	g, err := m.MkFuncDecl(astTestSymbol(t, m, "dump-g"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	fa, err := m.MkApp(f, terms[:1])
	astTestMust(t, err)
	fb, err := m.MkApp(f, terms[1:])
	astTestMust(t, err)
	gb, err := m.MkApp(g, terms[1:])
	astTestMust(t, err)
	astTestMust(t, s.Internalize(gb, false))
	_, err = s.MkEnode(fa, true, false, true)
	astTestMust(t, err)
	astTestMust(t, s.Internalize(fb, false))
	// Equality is present in the broader checker index but excluded from
	// the native declaration registry alongside constants/suppressed enodes.
	eq, err := m.MkEq(terms[0], terms[1])
	astTestMust(t, err)
	astTestMust(t, s.Internalize(terms[0], false))
	_, err = s.MkEnode(eq, false, false, false)
	astTestMust(t, err)
	fid, err := m.GetASTID(f)
	astTestMust(t, err)
	gid, err := m.GetASTID(g)
	astTestMust(t, err)
	fbid, err := m.GetASTID(fb)
	astTestMust(t, err)
	gbid, err := m.GetASTID(gb)
	astTestMust(t, err)
	var out bytes.Buffer
	astTestMust(t, s.DisplayDeclarationEnodes(&out))
	want := fmt.Sprintf("decl2enodes:\nid %d -> #%d\nid %d -> #%d\n", fid-(1<<31), fbid, gid-(1<<31), gbid)
	if out.String() != want {
		t.Fatal(out.String(), want)
	}
	astTestMust(t, s.PushScope())
	ga, err := m.MkApp(g, terms[:1])
	astTestMust(t, err)
	astTestMust(t, s.Internalize(ga, false))
	gaid, err := m.GetASTID(ga)
	astTestMust(t, err)
	out.Reset()
	astTestMust(t, s.DisplayDeclarationEnodes(&out))
	scoped := fmt.Sprintf("decl2enodes:\nid %d -> #%d\nid %d -> #%d #%d\n", fid-(1<<31), fbid, gid-(1<<31), gbid, gaid)
	if out.String() != scoped {
		t.Fatal(out.String(), scoped)
	}
	astTestMust(t, s.PopScopes(1))
	out.Reset()
	astTestMust(t, s.DisplayDeclarationEnodes(&out))
	if out.String() != want {
		t.Fatal("scoped registry entry survived", out.String())
	}
}

func TestSMTContextDisplayRelevancyAndSectionWriterPrefixes(t *testing.T) {
	m := astTestManager(t)
	_, p, s := smtInternalizerFixture(t, m, nil)
	astTestMust(t, s.SetBinaryClauseOptimization(true))
	a := smtRelevancyBool(t, m, "dump-prefix-a")
	b := smtRelevancyBool(t, m, "dump-prefix-b")
	astTestMust(t, s.Internalize(a, true))
	astTestMust(t, s.Internalize(b, true))
	av, _ := s.BoolVar(a)
	bv, _ := s.BoolVar(b)
	aid, err := m.GetASTID(a)
	astTestMust(t, err)
	bid, err := m.GetASTID(b)
	astTestMust(t, err)
	astTestMust(t, p.MarkRelevant(b))
	astTestMust(t, s.PushScope())
	astTestMust(t, p.MarkRelevant(a))
	var out bytes.Buffer
	astTestMust(t, s.DisplayRelevantExpressions(&out))
	if want := fmt.Sprintf("relevant exprs:\n#%d #%d \n", bid, aid); out.String() != want {
		t.Fatal(out.String(), want)
	}
	astTestMust(t, s.PopScopes(1))
	out.Reset()
	astTestMust(t, s.DisplayRelevantExpressions(&out))
	if want := fmt.Sprintf("relevant exprs:\n#%d \n", bid); out.String() != want {
		t.Fatal(out.String(), want)
	}
	p.enabled = false
	out.Reset()
	astTestMust(t, s.DisplayRelevantExpressions(&out))
	if out.Len() != 0 {
		t.Fatal("disabled relevance emitted a section")
	}
	p.enabled = true
	astTestMust(t, s.addClause([]SMTLiteral{NewSMTLiteral(av, false), NewSMTLiteral(bv, false)}, true, ast.ASTHandle{}))
	astTestMust(t, s.PushScope())
	astTestMust(t, s.addClause([]SMTLiteral{NewSMTLiteral(av, true), NewSMTLiteral(bv, true)}, true, ast.ASTHandle{}))
	sort, terms := smtUFConstants(t, m, "dump-prefix-uf-", 1)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "dump-prefix-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	app, err := m.MkApp(f, terms)
	astTestMust(t, err)
	astTestMust(t, s.Internalize(app, false))
	methods := []func(io.Writer) error{s.DisplayClauseSections, s.DisplayBinaryClauses, s.DisplayRelevantExpressions, s.DisplayDeclarationEnodes}
	before := m.Stats()
	for _, display := range methods {
		out.Reset()
		astTestMust(t, display(&out))
		want := out.String()
		if want == "" {
			t.Fatal("empty prefix fixture")
		}
		for cut := 0; cut < len(want); cut++ {
			w := &smtLRAOutputPrefix{remaining: cut}
			if err := display(w); !errors.Is(err, io.ErrClosedPipe) || w.text.String() != want[:cut] {
				t.Fatal(cut, err, w.text.String(), want[:cut])
			}
		}
		if err := display(smtLRAShortOutput{}); !errors.Is(err, io.ErrShortWrite) {
			t.Fatal(err)
		}
		if err := display(nil); !errors.Is(err, ast.ErrASTHandle) {
			t.Fatal(err)
		}
	}
	if m.Stats() != before {
		t.Fatal("section display acquired AST ownership", before, m.Stats())
	}
}

func TestSMTContextDisplaySectionsEmptyAndNilOwner(t *testing.T) {
	m := astTestManager(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	var out bytes.Buffer
	for _, display := range []func(io.Writer) error{s.DisplayClauseSections, s.DisplayBinaryClauses, s.DisplayRelevantExpressions} {
		astTestMust(t, display(&out))
	}
	if out.Len() != 0 {
		t.Fatal("empty optional sections emitted headers", out.String())
	}
	astTestMust(t, s.DisplayDeclarationEnodes(&out))
	if out.String() != "decl2enodes:\n" {
		t.Fatal(out.String())
	}
	var absent *SMTInternalizerState
	for _, display := range []func(io.Writer) error{absent.DisplayClauseSections, absent.DisplayBinaryClauses, absent.DisplayRelevantExpressions, absent.DisplayDeclarationEnodes} {
		if err := display(io.Discard); !errors.Is(err, ast.ErrASTHandle) {
			t.Fatal("nil owner", err)
		}
	}
}
