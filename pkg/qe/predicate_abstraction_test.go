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
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type closedPredicateTrace struct{}

func (closedPredicateTrace) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func predicateOwner(t *testing.T, f *fmFixture) *PredicateAbstraction {
	t.Helper()
	p, err := NewPredicateAbstraction(f.r.manager, f.r.config)
	fmMust(t, err)
	t.Cleanup(func() { fmMust(t, p.Close()) })
	return p
}
func predicateRefs(t *testing.T, f *fmFixture, hs ...ast.ASTHandle) *[]ast.ASTRef {
	t.Helper()
	rs := new([]ast.ASTRef)
	t.Cleanup(func() {
		for i := range *rs {
			fmMust(t, f.r.manager.DelRef(&(*rs)[i]))
		}
	})
	for _, h := range hs {
		r, err := f.r.manager.NewRef(h)
		fmMust(t, err)
		*rs = append(*rs, r)
	}
	return rs
}
func predicateModel(t *testing.T, f *fmFixture) *model.Model {
	t.Helper()
	// Source model evaluators borrow a fully installed AST manager. The FM
	// fixture normally installs only arithmetic, which violates BV utility
	// construction assertions in source-debug builds.
	fmMust(t, f.r.manager.RegisterDeclarationPlugins())
	md, err := model.NewModel(f.r.manager)
	fmMust(t, err)
	t.Cleanup(func() { fmMust(t, md.Close()) })
	return md
}
func predicateRegister(t *testing.T, f *fmFixture, md *model.Model, h, v ast.ASTHandle) {
	t.Helper()
	d, err := f.r.manager.GetDecl(h)
	fmMust(t, err)
	fmMust(t, md.Core().RegisterConstant(d, v))
}
func predicateValue(t *testing.T, f *fmFixture, r ast.ASTRef) ast.ASTHandle {
	t.Helper()
	h, err := f.r.manager.RefGet(r)
	fmMust(t, err)
	return h
}

func TestMaxLevelSentinelMergeAndDisplay(t *testing.T) {
	l := EmptyMaxLevel()
	if l.Max() != NoQuantifierLevel {
		t.Fatal(l)
	}
	l.Merge(MaxLevel{3, NoQuantifierLevel})
	l.Merge(MaxLevel{1, 4})
	if l != (MaxLevel{3, 4}) || l.Max() != 4 {
		t.Fatal(l)
	}
	var out bytes.Buffer
	fmMust(t, l.Display(&out))
	if out.String() != "e3 a4 " {
		t.Fatal(out.String())
	}
	if err := l.Display(nil); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
}

func TestPredicateAbstractionLevelsRoundTripAndReset(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p := predicateOwner(t, f)
	x, y := f.constant("x", f.integer), f.constant("y", f.integer)
	b := f.constant("b", f.boolean)
	fmMust(t, p.SetExpressionLevel(x, MaxLevel{0, NoQuantifierLevel}))
	fmMust(t, p.SetExpressionLevel(y, MaxLevel{NoQuantifierLevel, 3}))
	fmMust(t, p.SetExpressionLevel(b, MaxLevel{2, NoQuantifierLevel}))
	atom := f.predicate("P", x)
	eq := f.own(m.MkEq(x, y))
	formula := f.own(m.MkAndArray([]ast.ASTHandle{atom, eq, b}))
	d, err := m.GetDecl(atom)
	fmMust(t, err)
	fmMust(t, p.SetDeclarationLevel(d, MaxLevel{1, NoQuantifierLevel}))
	l, err := p.ComputeLevel(formula)
	fmMust(t, err)
	if l != (MaxLevel{2, 3}) {
		t.Fatal(l)
	}
	defs := predicateRefs(t, f)
	level := EmptyMaxLevel()
	fmMust(t, p.AbstractAtoms(formula, &level, defs))
	if level != (MaxLevel{2, 3}) || len(*defs) != 2 {
		t.Fatal(level, len(*defs))
	}
	// Source traversal pops the last Boolean child first, then equality.
	first := predicateValue(t, f, (*defs)[0])
	rhs, err := m.GetArg(first, 1)
	fmMust(t, err)
	if rhs != eq {
		t.Fatal("definition order", rhs, eq)
	}
	out := predicateRefs(t, f, ast.ASTHandle{})
	fmMust(t, p.MakeAbstract(formula, (*out)[0]))
	abstract := predicateValue(t, f, (*out)[0])
	if abstract == formula {
		t.Fatal("atoms were not replaced")
	}
	stats, err := p.CollectStatistics()
	fmMust(t, err)
	if stats["qsat num predicates"] != 3 {
		t.Fatal(stats)
	}
	fmMust(t, p.PredicatesToLiterals(out))
	if predicateValue(t, f, (*out)[0]) != formula {
		t.Fatal("concretization lost original DAG")
	}
	fmMust(t, p.Reset())
	stats, err = p.CollectStatistics()
	fmMust(t, err)
	if stats["qsat num predicates"] != 0 {
		t.Fatal(stats)
	}
	l, err = p.ComputeLevel(atom)
	fmMust(t, err)
	if l != (MaxLevel{1, NoQuantifierLevel}) {
		t.Fatal("declaration level did not survive reset", l)
	}
}

func TestPredicateAbstractionSimultaneousMapsAndPredicateSuppression(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p := predicateOwner(t, f)
	x := f.constant("x", f.integer)
	atom := f.predicate("P", x)
	a, b := f.constant("a", f.boolean), f.constant("b", f.boolean)
	fmMust(t, p.AddPredicate(a, b))
	fmMust(t, p.AddPredicate(b, atom))
	out := predicateRefs(t, f, a)
	fmMust(t, p.PredicatesToLiterals(out))
	if predicateValue(t, f, (*out)[0]) != b {
		t.Fatal("replacement recursively expanded")
	}
	fmMust(t, p.MakeAbstract(atom, (*out)[0]))
	if predicateValue(t, f, (*out)[0]) != b {
		t.Fatal("reverse replacement lost identity")
	}
	fmMust(t, p.Reset())
	d, err := m.GetDecl(atom)
	fmMust(t, err)
	fmMust(t, p.SetExpressionLevel(x, MaxLevel{2, NoQuantifierLevel}))
	fmMust(t, p.SetDeclarationLevel(d, MaxLevel{0, NoQuantifierLevel}))
	defs := predicateRefs(t, f)
	level := EmptyMaxLevel()
	fmMust(t, p.AbstractAtoms(atom, &level, defs))
	if len(*defs) != 1 || len(p.predicates) != 0 || level.Max() != 2 {
		t.Fatal("lower-level predicate declaration was scheduled", len(*defs), len(p.predicates), level)
	}
}

func TestPredicateAbstractionScopedModelAssumptions(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p := predicateOwner(t, f)
	md := predicateModel(t, f)
	truth := f.own(m.MkTrue())
	falsehood := f.own(m.MkFalse())
	vars := []ast.ASTHandle{f.constant("p0", f.boolean), f.constant("p2", f.boolean), f.constant("p4", f.boolean), f.constant("mixed4", f.boolean)}
	levels := []MaxLevel{{0, NoQuantifierLevel}, {2, NoQuantifierLevel}, {4, NoQuantifierLevel}, {4, 2}}
	for i, h := range vars {
		fmMust(t, p.SetExpressionLevel(h, levels[i]))
		fmMust(t, p.Insert(h, levels[i]))
		v := truth
		if i == 0 {
			v = falsehood
		}
		predicateRegister(t, f, md, h, v)
	}
	fmMust(t, p.Push())
	out := predicateRefs(t, f, truth)
	fmMust(t, p.GetAssumptions(md.Core(), out))
	if len(*out) != 4 {
		t.Fatal("alternating-level assumptions", len(*out))
	}
	negative, inner, err := m.IsNot(predicateValue(t, f, (*out)[1]))
	fmMust(t, err)
	if !negative || inner != vars[0] {
		t.Fatal("model polarity lost")
	}
	if predicateValue(t, f, (*out)[2]) != vars[1] || predicateValue(t, f, (*out)[3]) != vars[2] {
		t.Fatal("wrong future levels")
	}
	nilModel := predicateRefs(t, f)
	fmMust(t, p.GetAssumptions(nil, nilModel))
	if len(*nilModel) != 1 {
		t.Fatal("future assumptions persisted")
	}
	fmMust(t, p.Push())
	next := predicateRefs(t, f)
	fmMust(t, p.GetAssumptions(md.Core(), next))
	if len(*next) != 1 {
		t.Fatal("wrong scope parity", len(*next))
	}
	fmMust(t, p.Pop(2))
	empty := predicateRefs(t, f)
	fmMust(t, p.GetAssumptions(nil, empty))
	if len(*empty) != 0 {
		t.Fatal("scope assumptions escaped")
	}
	if err = p.Pop(0); !errors.Is(err, arena.ErrBounds) {
		t.Fatal(err)
	}
}

func TestPredicateAbstractionAssumptionDefinitionsAndModelFiltering(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p := predicateOwner(t, f)
	md := predicateModel(t, f)
	x := f.constant("x", f.integer)
	fmMust(t, p.SetExpressionLevel(x, MaxLevel{0, NoQuantifierLevel}))
	atom := f.own(m.MkArithmeticGE(x, f.number("0")))
	negative := f.own(m.MkNot(atom))
	defs := predicateRefs(t, f)
	out := predicateRefs(t, f, ast.ASTHandle{}, ast.ASTHandle{})
	level := MaxLevel{2, NoQuantifierLevel}
	fmMust(t, p.MakeAssumptionLiteral(negative, md.Core(), level, defs, (*out)[0]))
	if len(*defs) != 2 {
		t.Fatal("missing atom and assumption definitions", len(*defs))
	}
	neg, pred, err := m.IsNot(predicateValue(t, f, (*out)[0]))
	fmMust(t, err)
	if !neg {
		t.Fatal("negative assumption lost polarity")
	}
	d, err := m.GetDecl(pred)
	fmMust(t, err)
	v, err := md.Core().ConstantInterpretation(d)
	fmMust(t, err)
	if !m.IsFalse(v) {
		t.Fatal("new assumption interpretation")
	}
	fmMust(t, p.PredicatesToAssumptions(predicateValue(t, f, (*out)[0]), (*out)[1]))
	if predicateValue(t, f, (*out)[1]) != negative {
		t.Fatal("assumption concretization")
	}
	fmMust(t, p.MakeAssumptionLiteral(atom, md.Core(), level, defs, (*out)[1]))
	if predicateValue(t, f, (*out)[1]) != pred || len(*defs) != 2 {
		t.Fatal("assumption reused incorrectly")
	}
	keep := f.constant("keep", f.integer)
	predicateRegister(t, f, md, keep, f.number("7"))
	filter, err := p.FilterModelConverter()
	fmMust(t, err)
	fmMust(t, filter.IncRef())
	t.Cleanup(func() { fmMust(t, filter.DecRef()) })
	fmMust(t, p.Close())
	var mr tactic.ModelRef
	t.Cleanup(func() { fmMust(t, mr.Close()) })
	fmMust(t, mr.Set(md.Core()))
	fmMust(t, filter.ApplyUnindexed(&mr))
	filtered, err := mr.Get()
	fmMust(t, err)
	kd, err := m.GetDecl(keep)
	fmMust(t, err)
	v, err = filtered.ConstantInterpretation(kd)
	fmMust(t, err)
	if v.IsNull() {
		t.Fatal("filter removed user constant")
	}
	v, err = filtered.ConstantInterpretation(d)
	fmMust(t, err)
	if !v.IsNull() {
		t.Fatal("filter exposed fresh assumption")
	}
}

func TestPredicateAbstractionFreeVariablesValidationAndGuards(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	p := predicateOwner(t, f)
	x := f.constant("x", f.integer)
	bound := f.variable(0, f.integer)
	body := f.predicate("P", x, bound)
	// The extractor visits only quantifier bodies; bound variables and
	// function declarations are not returned as free constants.
	name, err := m.Symbols.Intern("bound")
	fmMust(t, err)
	patternOnly := f.constant("pattern-only", f.integer)
	pattern := f.own(m.MkPattern([]ast.ASTHandle{f.predicate("trigger", patternOnly, bound)}))
	q := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: body, Patterns: []ast.ASTHandle{pattern}}))
	vars := predicateRefs(t, f)
	fmMust(t, p.GetFreeVariables(q, vars))
	if len(*vars) != 1 || predicateValue(t, f, (*vars)[0]) != x {
		t.Fatal("free-variable extraction", len(*vars))
	}
	a, b := f.constant("a", f.boolean), f.constant("b", f.boolean)
	fmMust(t, p.AddPredicate(a, b))
	md := predicateModel(t, f)
	truth := f.own(m.MkTrue())
	falsehood := f.own(m.MkFalse())
	predicateRegister(t, f, md, a, truth)
	predicateRegister(t, f, md, b, truth)
	valid, err := p.ValidateDefinitions(md.Core())
	fmMust(t, err)
	if !valid {
		t.Fatal("equal definitions rejected")
	}
	predicateRegister(t, f, md, b, falsehood)
	valid, err = p.ValidateDefinitions(md.Core())
	fmMust(t, err)
	if valid {
		t.Fatal("mismatched definitions accepted")
	}
	copy := *p
	if _, err = copy.ComputeLevel(a); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	if _, err = p.ComputeLevel(bound); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal(err)
	}
	fmMust(t, p.Close())
	if err = p.Push(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
}

func TestPredicateAbstractionAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "abstract", "rebuild", "assumption", "model-assumptions", "validate", "display", "trace", "closed-trace", "model-trace", "close"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 2048 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newFMFixture(t)
					m := f.r.manager
					var trace bytes.Buffer
					if strings.Contains(phase, "trace") {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = closedPredicateTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					x := f.constant("x", f.integer)
					b := f.constant("b", f.boolean)
					atom := f.predicate("P", x)
					formula := f.own(m.MkAndArray([]ast.ASTHandle{atom, b}))
					defs := predicateRefs(t, f)
					out := predicateRefs(t, f, ast.ASTHandle{})
					if phase == "model-assumptions" || phase == "model-trace" {
						// Vector outputs append assumptions, unlike scalar output
						// slots. A null prefix violates source display's domain.
						out = predicateRefs(t, f)
					}
					md := predicateModel(t, f)
					var p *PredicateAbstraction
					var err error
					if phase != "constructor" {
						p = predicateOwner(t, f)
						fmMust(t, p.SetExpressionLevel(x, MaxLevel{0, NoQuantifierLevel}))
						fmMust(t, p.SetExpressionLevel(b, MaxLevel{0, NoQuantifierLevel}))
						if phase != "abstract" && phase != "assumption" {
							l := EmptyMaxLevel()
							fmMust(t, p.AbstractAtoms(formula, &l, defs))
							fmMust(t, p.Push())
						}
					}
					m.Budget().FailAfter(stop)
					switch phase {
					case "constructor":
						p, err = NewPredicateAbstraction(m, f.r.config)
					case "abstract":
						l := EmptyMaxLevel()
						err = p.AbstractAtoms(formula, &l, defs)
					case "rebuild":
						err = p.MakeAbstract(formula, (*out)[0])
					case "assumption":
						err = p.MakeAssumptionLiteral(atom, md.Core(), MaxLevel{1, NoQuantifierLevel}, defs, (*out)[0])
					case "model-assumptions":
						err = p.GetAssumptions(md.Core(), out)
					case "validate":
						_, err = p.ValidateDefinitions(md.Core())
					case "display":
						var w bytes.Buffer
						err = p.Display(&w)
					case "trace", "closed-trace":
						err = p.MakeAssumptionLiteral(atom, md.Core(), MaxLevel{1, NoQuantifierLevel}, defs, (*out)[0])
					case "model-trace":
						err = p.GetAssumptions(md.Core(), out)
					case "close":
						err = p.Close()
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
					t.Fatal("predicate allocation frontier", stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}

func TestPredicateAbstractionDisplay(t *testing.T) {
	f := newFMFixture(t)
	p := predicateOwner(t, f)
	a, b := f.constant("a", f.boolean), f.constant("b", f.boolean)
	fmMust(t, p.AddPredicate(a, b))
	fmMust(t, p.SetExpressionLevel(a, MaxLevel{1, NoQuantifierLevel}))
	fmMust(t, p.Insert(a, MaxLevel{1, NoQuantifierLevel}))
	var out bytes.Buffer
	fmMust(t, p.Display(&out))
	if !strings.Contains(out.String(), "a |-> b\n") || !strings.Contains(out.String(), "level 1\na := b\n") {
		t.Fatal(out.String())
	}
	refs := predicateRefs(t, f, a)
	out.Reset()
	fmMust(t, p.DisplayAssumptions(&out, *refs))
	if out.String() != "a - e1  : b\n" {
		t.Fatal(out.String())
	}
}
