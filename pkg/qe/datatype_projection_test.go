package qe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type datatypeProjectionFixture struct {
	f                                      *fmFixture
	md                                     *model.Model
	pair, list, color                      ast.ASTHandle
	pairCtor, nilCtor, consCtor, red, blue ast.ASTHandle
	pairFields, listFields                 []ast.ASTHandle
	consTest                               ast.ASTHandle
}

func newDatatypeProjectionFixture(t *testing.T) *datatypeProjectionFixture {
	t.Helper()
	f := newFMFixture(t)
	m := f.r.manager
	md := predicateModel(t, f)
	sym := func(s string) usymbol.Symbol { n, e := m.Symbols.Intern(s); fmMust(t, e); return n }
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{
		{Name: sym("DPair"), Constructors: []ast.DatatypeConstructorDefinition{{Name: sym("pair"), Recognizer: sym("is-pair"), Accessors: []ast.DatatypeAccessorDefinition{{Name: sym("first"), Range: f.integer}, {Name: sym("second"), Range: f.boolean}}}}},
		{Name: sym("DList"), Constructors: []ast.DatatypeConstructorDefinition{{Name: sym("nil"), Recognizer: sym("is-nil")}, {Name: sym("cons"), Recognizer: sym("is-cons"), Accessors: []ast.DatatypeAccessorDefinition{{Name: sym("head"), Range: f.integer}, {Name: sym("tail"), Recursive: true, RecursiveIndex: 1}}}}},
		{Name: sym("DColor"), Constructors: []ast.DatatypeConstructorDefinition{{Name: sym("red"), Recognizer: sym("is-red")}, {Name: sym("blue"), Recognizer: sym("is-blue")}}},
	})
	fmMust(t, err)
	for _, s := range sorts {
		f.own(s, nil)
	}
	constructors := make([][]ast.ASTHandle, 3)
	for i, s := range sorts {
		constructors[i], err = m.DatatypeConstructors(s)
		fmMust(t, err)
		for _, d := range constructors[i] {
			f.own(d, nil)
		}
	}
	d := &datatypeProjectionFixture{f: f, md: md, pair: sorts[0], list: sorts[1], color: sorts[2], pairCtor: constructors[0][0], nilCtor: constructors[1][0], consCtor: constructors[1][1], red: constructors[2][0], blue: constructors[2][1]}
	d.pairFields, err = m.DatatypeConstructorAccessors(d.pairCtor)
	fmMust(t, err)
	d.listFields, err = m.DatatypeConstructorAccessors(d.consCtor)
	fmMust(t, err)
	for _, a := range append(append([]ast.ASTHandle(nil), d.pairFields...), d.listFields...) {
		f.own(a, nil)
	}
	d.consTest, err = m.DatatypeConstructorRecognizer(d.consCtor)
	fmMust(t, err)
	f.own(d.consTest, nil)
	return d
}
func (d *datatypeProjectionFixture) app(decl ast.ASTHandle, args ...ast.ASTHandle) ast.ASTHandle {
	return d.f.own(d.f.r.manager.MkApp(decl, args))
}
func datatypeProjectionOwner(t *testing.T, d *datatypeProjectionFixture) *DatatypeProjection {
	t.Helper()
	p, err := NewDatatypeProjection(d.f.r.manager, d.f.r.config)
	fmMust(t, err)
	t.Cleanup(func() { fmMust(t, p.Close()) })
	return p
}

type datatypeProjectionCase struct {
	d                 *datatypeProjectionFixture
	target            ast.ASTHandle
	additional, forms *[]ast.ASTRef
	original          []ast.ASTHandle
	fresh             int
}

var datatypeProjectionModes = []string{"pair", "enum", "recursive-direct", "recursive-variable", "recursive-same", "recursive-nested", "recursive-distinct", "recursive-fallback", "incomplete-absent"}

func datatypeCase(t *testing.T, mode string) *datatypeProjectionCase {
	t.Helper()
	d := newDatatypeProjectionFixture(t)
	f := d.f
	m := f.r.manager
	x, y := f.constant("dx", d.list), f.constant("dy", d.list)
	nilValue := d.app(d.nilCtor)
	one, two, three := f.number("1"), f.number("2"), f.number("3")
	predicateRegister(t, f, d.md, x, nilValue)
	predicateRegister(t, f, d.md, y, nilValue)
	var forms []ast.ASTHandle
	fresh := 0
	target := x
	notCons := func(h ast.ASTHandle) ast.ASTHandle { return f.own(m.MkNot(d.app(d.consTest, h))) }
	switch mode {
	case "pair":
		target = f.constant("dp", d.pair)
		value := d.app(d.pairCtor, three, f.own(m.MkTrue()))
		predicateRegister(t, f, d.md, target, value)
		forms = []ast.ASTHandle{f.own(m.MkArithmeticGE(d.app(d.pairFields[0], target), one)), d.app(d.pairFields[1], target)}
		fresh = 2
	case "enum":
		target = f.constant("dc", d.color)
		predicateRegister(t, f, d.md, target, d.app(d.blue))
		forms = []ast.ASTHandle{f.own(m.MkNot(f.own(m.MkEq(target, d.app(d.red)))))}
	case "recursive-direct":
		forms = []ast.ASTHandle{f.own(m.MkEq(x, y)), notCons(x)}
	case "recursive-variable", "recursive-distinct":
		value := d.app(d.consCtor, one, nilValue)
		predicateRegister(t, f, d.md, y, value)
		left := d.app(d.consCtor, one, x)
		eq := f.own(m.MkEq(left, y))
		if mode == "recursive-distinct" {
			eq = f.own(m.MkNot(f.own(m.MkDistinct([]ast.ASTHandle{left, y}))))
		}
		forms = []ast.ASTHandle{eq, notCons(x)}
	case "recursive-same":
		forms = []ast.ASTHandle{f.own(m.MkEq(d.app(d.consCtor, one, x), d.app(d.consCtor, one, nilValue))), notCons(x)}
	case "recursive-nested":
		value := d.app(d.consCtor, one, d.app(d.consCtor, two, nilValue))
		predicateRegister(t, f, d.md, y, value)
		forms = []ast.ASTHandle{f.own(m.MkEq(d.app(d.consCtor, one, d.app(d.consCtor, two, x)), y)), notCons(x)}
	case "recursive-fallback":
		predicateRegister(t, f, d.md, x, d.app(d.consCtor, three, nilValue))
		forms = []ast.ASTHandle{d.app(d.consTest, x), f.own(m.MkArithmeticGE(d.app(d.listFields[0], x), one))}
		fresh = 2
	case "incomplete-absent":
		target = f.constant("uninterpreted-model", d.list)
		forms = []ast.ASTHandle{f.own(m.MkTrue())}
	}
	return &datatypeProjectionCase{d: d, target: target, additional: predicateRefs(t, f, y), forms: predicateRefs(t, f, forms...), original: forms, fresh: fresh}
}

func TestDatatypeProjectionConstructorsModelsAndVariables(t *testing.T) {
	for _, mode := range datatypeProjectionModes {
		t.Run(mode, func(t *testing.T) {
			c := datatypeCase(t, mode)
			p := datatypeProjectionOwner(t, c.d)
			f := c.d.f
			m := f.r.manager
			arithmeticEvaluate(t, f, c.d.md, *c.forms)
			before := len(c.d.md.Core().ConstantDeclarations)
			extra := predicateValue(t, f, (*c.additional)[0])
			ok, err := p.ProjectVariable(c.d.md.Core(), c.target, c.additional, c.forms)
			fmMust(t, err)
			if !ok || len(*c.additional) != 1 || predicateValue(t, f, (*c.additional)[0]) != extra {
				t.Fatal("projector changed extra-variable vector", ok, len(*c.additional))
			}
			if got := len(c.d.md.Core().ConstantDeclarations) - before; got != c.fresh {
				t.Fatal("fresh model fields", got, c.fresh)
			}
			arithmeticEvaluate(t, f, c.d.md, *c.forms)
			pre := predicateOwner(t, f)
			for _, r := range *c.forms {
				vars := predicateRefs(t, f)
				fmMust(t, pre.GetFreeVariables(predicateValue(t, f, r), vars))
				for _, v := range *vars {
					if predicateValue(t, f, v) == c.target {
						t.Fatal("target remains in output")
					}
				}
			}
			if mode == "recursive-variable" || mode == "recursive-distinct" {
				if len(*c.forms) != 3 {
					t.Fatal("missing sibling or recognizer side condition", len(*c.forms))
				}
			}
			if mode == "recursive-nested" && len(*c.forms) != 5 {
				t.Fatal("nested side condition order", len(*c.forms))
			}
			if mode == "recursive-same" {
				if len(*c.forms) != 2 || !m.IsTrue(predicateValue(t, f, (*c.forms)[0])) {
					t.Fatal("same-constructor accessor reduction")
				}
			}
			fmMust(t, p.Close())
			arithmeticEvaluate(t, f, c.d.md, *c.forms)
		})
	}
}

func TestDatatypeProjectionForeignFieldLifting(t *testing.T) {
	for _, mode := range []string{"left", "right", "both", "boolean-only", "datatype-only"} {
		t.Run(mode, func(t *testing.T) {
			d := newDatatypeProjectionFixture(t)
			f := d.f
			m := f.r.manager
			p := datatypeProjectionOwner(t, d)
			x, y := f.constant("fx", f.integer), f.constant("fy", f.integer)
			b, c := f.constant("fb", f.boolean), f.constant("fc", f.boolean)
			r := f.constant("fr", d.pair)
			truth := f.own(m.MkTrue())
			three := f.number("3")
			for _, h := range []ast.ASTHandle{x, y} {
				predicateRegister(t, f, d.md, h, three)
			}
			for _, h := range []ast.ASTHandle{b, c} {
				predicateRegister(t, f, d.md, h, truth)
			}
			predicateRegister(t, f, d.md, r, d.app(d.pairCtor, three, truth))
			left, right := d.app(d.pairCtor, x, b), r
			vars := predicateRefs(t, f, x, b, r)
			if mode == "right" {
				left, right = right, left
			}
			if mode == "both" {
				right = d.app(d.pairCtor, y, c)
				vars = predicateRefs(t, f, x, y)
			}
			if mode == "boolean-only" {
				left = d.app(d.pairCtor, three, b)
				vars = predicateRefs(t, f, b)
			}
			if mode == "datatype-only" {
				vars = predicateRefs(t, f, r)
			}
			forms := predicateRefs(t, f, f.own(m.MkEq(left, right)))
			before := len(d.md.Core().ConstantDeclarations)
			changed, err := p.Solve(d.md.Core(), vars, forms)
			fmMust(t, err)
			if changed != (mode != "boolean-only" && mode != "datatype-only") {
				t.Fatal(changed)
			}
			if len(d.md.Core().ConstantDeclarations) != before {
				t.Fatal("cheap solve changed model")
			}
			arithmeticEvaluate(t, f, d.md, *forms)
			if mode == "left" || mode == "right" {
				if len(*forms) != 3 {
					t.Fatal("field/recognizer count", len(*forms))
				}
			}
			if mode == "both" {
				if len(*forms) != 2 {
					t.Fatal("constructor equality was not decomposed")
				}
				eq := predicateValue(t, f, (*forms)[1])
				a, err := m.GetArg(eq, 0)
				fmMust(t, err)
				if a != y {
					t.Fatal("source right-constructor preference lost")
				}
			}
		})
	}
}

func TestDatatypeProjectionAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "pair", "enum", "recursive-direct", "recursive-variable", "recursive-same", "recursive-nested", "recursive-distinct", "recursive-fallback", "reuse", "foreign", "trace", "closed-trace", "pair-trace", "pair-closed-trace", "foreign-trace", "foreign-closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					mode := phase
					if phase == "constructor" || phase == "reuse" || phase == "foreign" || strings.Contains(phase, "trace") {
						mode = "recursive-variable"
					}
					if strings.HasPrefix(phase, "pair-") {
						mode = "pair"
					}
					c := datatypeCase(t, mode)
					f := c.d.f
					m := f.r.manager
					var trace bytes.Buffer
					if strings.Contains(phase, "trace") {
						var w io.Writer = &trace
						if strings.Contains(phase, "closed-trace") {
							w = closedPredicateTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(w)
						tags.Enable("qe")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					var p *DatatypeProjection
					var err error
					if phase != "constructor" {
						p = datatypeProjectionOwner(t, c.d)
					}
					if phase == "reuse" {
						_, err = p.ProjectVariable(c.d.md.Core(), c.target, c.additional, c.forms)
						fmMust(t, err)
						c.forms = predicateRefs(t, f, c.original...)
					}
					foreign := f.constant("foreign", f.integer)
					vars := predicateRefs(t, f, foreign)
					if strings.HasPrefix(phase, "foreign") {
						r := f.constant("foreign-pair", c.d.pair)
						c.forms = predicateRefs(t, f, f.own(m.MkEq(c.d.app(c.d.pairCtor, foreign, f.own(m.MkTrue())), r)))
					}
					m.Budget().FailAfter(stop)
					if phase == "constructor" {
						p, err = NewDatatypeProjection(m, f.r.config)
					} else if strings.HasPrefix(phase, "foreign") {
						_, err = p.Solve(c.d.md.Core(), vars, c.forms)
					} else {
						_, err = p.ProjectVariable(c.d.md.Core(), c.target, c.additional, c.forms)
					}
					m.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if phase == "constructor" && p != nil {
						fmMust(t, p.Close())
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						var memout memory.Failure
						if !errors.As(err, &memout) || memout.Kind != memory.FailureMemout {
							t.Fatalf("%v\n%s", err, trace.String())
						}
					}
				})
				if t.Failed() {
					t.Fatal("datatype projection frontier", stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}

func TestDatatypeProjectionReuseTraceAndOwnerGuards(t *testing.T) {
	c := datatypeCase(t, "recursive-variable")
	f := c.d.f
	m := f.r.manager
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("qe")
	t.Cleanup(tags.Finalize)
	f.r.config.Trace = tags
	p := datatypeProjectionOwner(t, c.d)
	ok, err := p.ProjectVariable(c.d.md.Core(), c.target, c.additional, c.forms)
	fmMust(t, err)
	if !ok {
		t.Fatal("projection refused")
	}
	if tags.ShouldTrace("qe") && !strings.Contains(out.String(), "operator() src/qe/qe_datatypes.cpp:48") {
		t.Fatal(out.String())
	}
	again := predicateRefs(t, f, c.original...)
	ok, err = p.ProjectVariable(c.d.md.Core(), c.target, c.additional, again)
	fmMust(t, err)
	if !ok {
		t.Fatal("owner reuse refused")
	}
	want := make([]ast.ASTHandle, len(*c.forms))
	for i, r := range *c.forms {
		want[i] = predicateValue(t, f, r)
	}
	projectionExpect(t, f, *again, want)
	family, err := p.FamilyID()
	fmMust(t, err)
	expected, err := m.DatatypeFamilyID()
	fmMust(t, err)
	if family != expected {
		t.Fatal(family)
	}
	copy := *p
	if _, err = copy.FamilyID(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	foreign := newFMFixture(t)
	md := predicateModel(t, foreign)
	if _, err = p.ProjectVariable(md.Core(), c.target, c.additional, c.forms); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	integer := f.constant("not-datatype", f.integer)
	if _, err = p.ProjectVariable(c.d.md.Core(), integer, c.additional, c.forms); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal(err)
	}
	fmMust(t, p.Close())
	if _, err = p.FamilyID(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
}
