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
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/memory"
)

type arrayProjectionCase struct {
	f                           *fmFixture
	md                          *model.Model
	sort, target                ast.ASTHandle
	witness                     ast.ASTHandle
	additional, lits            *[]ast.ASTRef
	original                    []ast.ASTHandle
	success                     bool
	fresh                       int
	multi, boolean, nestedRange bool
}

var arrayProjectionModes = []string{"select-distinct", "select-same", "select-shared", "select-bool", "select-multi", "select-array-range", "nested-select", "absent", "bare-disequality", "direct-equality", "reverse-equality", "store-equality", "store-repeated", "store-distinct", "store-multi", "select-store-refusal", "store-disequality-refusal"}

func arrayProjectionOwner(t *testing.T, c *arrayProjectionCase) *ArrayProjection {
	t.Helper()
	p, err := NewArrayProjection(c.f.r.manager, c.f.r.config)
	fmMust(t, err)
	t.Cleanup(func() { fmMust(t, p.Close()) })
	return p
}
func arrayCase(t *testing.T, mode string) *arrayProjectionCase {
	t.Helper()
	f := newFMFixture(t)
	m := f.r.manager
	md := predicateModel(t, f)
	multi := strings.HasSuffix(mode, "multi")
	boolean := mode == "select-bool"
	nestedRange := mode == "select-array-range" || mode == "nested-select"
	domains := []ast.ASTHandle{f.integer}
	if multi {
		domains = append(domains, f.integer)
	}
	element := f.integer
	if boolean {
		element = f.boolean
	}
	if nestedRange {
		element = f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
	}
	sort := f.own(m.MkArraySort(ast.SliceArguments(domains), element))
	a, b := f.constant("A", sort), f.constant("B", sort)
	i, j, k, l := f.constant("i", f.integer), f.constant("j", f.integer), f.constant("k", f.integer), f.constant("l", f.integer)
	one, two, zero, three, five, seven := f.number("1"), f.number("2"), f.number("0"), f.number("3"), f.number("5"), f.number("7")
	iValue, jValue := one, two
	if mode == "select-same" || mode == "store-repeated" {
		jValue = one
	}
	for _, pair := range [][2]ast.ASTHandle{{i, iValue}, {j, jValue}, {k, two}, {l, one}} {
		predicateRegister(t, f, md, pair[0], pair[1])
	}
	family, err := m.ArrayFamilyID()
	fmMust(t, err)
	selectAt := func(array ast.ASTHandle, indices ...ast.ASTHandle) ast.ASTHandle {
		args := append([]ast.ASTHandle{array}, indices...)
		return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpSelect, ast.SliceArguments(args)))
	}
	storeAt := func(array ast.ASTHandle, indices []ast.ASTHandle, value ast.ASTHandle) ast.ASTHandle {
		args := append([]ast.ASTHandle{array}, indices...)
		args = append(args, value)
		return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpStore, ast.SliceArguments(args)))
	}
	first, second := []ast.ASTHandle{i}, []ast.ASTHandle{j}
	firstValues, secondValues := []ast.ASTHandle{iValue}, []ast.ASTHandle{jValue}
	if multi {
		first, second = []ast.ASTHandle{i, j}, []ast.ASTHandle{k, l}
		firstValues, secondValues = []ast.ASTHandle{one, two}, []ast.ASTHandle{two, one}
	}
	defaultValue, firstValue, secondValue := zero, three, seven
	if boolean {
		defaultValue = f.own(m.MkFalse())
		firstValue = f.own(m.MkTrue())
		secondValue = defaultValue
	}
	if nestedRange {
		defaultValue = f.own(m.MkConstArray(element, zero))
		firstValue = f.own(m.MkConstArray(element, three))
		secondValue = f.own(m.MkConstArray(element, seven))
	}
	base := f.own(m.MkConstArray(sort, defaultValue))
	modelA := storeAt(base, firstValues, firstValue)
	if mode != "select-same" && mode != "store-repeated" {
		modelA = storeAt(modelA, secondValues, secondValue)
	} else {
		secondValue = firstValue
	}
	modelB := modelA
	var lits []ast.ASTHandle
	success := true
	fresh := 2
	eq := func(x, y ast.ASTHandle) ast.ASTHandle { return f.own(m.MkEq(x, y)) }
	lits = []ast.ASTHandle{eq(selectAt(a, first...), firstValue), f.own(m.MkTrue()), eq(selectAt(a, second...), secondValue)}
	switch mode {
	case "select-shared":
		r := selectAt(a, first...)
		lits = []ast.ASTHandle{eq(r, three), f.own(m.MkArithmeticGE(r, zero))}
		fresh = 1
	case "nested-select":
		lits = []ast.ASTHandle{eq(selectAt(selectAt(a, first...), k), three)}
		fresh = 1
	case "select-bool":
		lits = []ast.ASTHandle{selectAt(a, first...), f.own(m.MkNot(selectAt(a, second...)))}
	case "absent":
		lits = []ast.ASTHandle{f.own(m.MkTrue())}
		fresh = 0
	case "bare-disequality":
		modelB = base
		lits = []ast.ASTHandle{f.own(m.MkNot(eq(a, b)))}
		fresh = 0
	case "direct-equality", "reverse-equality":
		definition := eq(a, b)
		if mode == "reverse-equality" {
			definition = eq(b, a)
		}
		lits = []ast.ASTHandle{definition, eq(selectAt(a, first...), firstValue)}
		fresh = 0
	case "store-equality", "store-multi":
		modelB = storeAt(modelA, firstValues, seven)
		lits = []ast.ASTHandle{eq(storeAt(a, first, seven), b), eq(selectAt(a, first...), firstValue)}
		fresh = 1
	case "store-repeated", "store-distinct":
		modelB = storeAt(storeAt(modelA, firstValues, five), secondValues, seven)
		lits = []ast.ASTHandle{eq(storeAt(storeAt(a, first, five), second, seven), b), eq(selectAt(a, first...), firstValue)}
		fresh = 2
		if mode == "store-repeated" {
			fresh = 1
		}
	case "select-store-refusal":
		lits = []ast.ASTHandle{eq(selectAt(storeAt(a, first, seven), second...), secondValue)}
		success = false
		fresh = 0
	case "store-disequality-refusal":
		modelB = base
		lits = []ast.ASTHandle{f.own(m.MkNot(eq(storeAt(a, first, seven), b)))}
		success = false
		fresh = 0
	}
	predicateRegister(t, f, md, a, modelA)
	predicateRegister(t, f, md, b, modelB)
	// Independent constructive witnesses for the mathematical test cases.
	// They use the symbolic input indices, so the C++ implication applies
	// to every parameter assignment satisfying the native projection.
	witness := storeAt(base, first, firstValue)
	switch mode {
	case "select-distinct", "select-same", "select-bool", "select-multi", "select-array-range":
		witness = storeAt(witness, second, secondValue)
	case "absent":
		witness = base
	case "bare-disequality":
		different := f.own(m.MkArithmeticAdd(selectAt(b, zero), one))
		witness = storeAt(b, []ast.ASTHandle{zero}, different)
	case "direct-equality", "reverse-equality":
		witness = b
	case "store-equality", "store-multi", "store-repeated", "store-distinct":
		witness = storeAt(b, first, firstValue)
	}
	return &arrayProjectionCase{f: f, md: md, sort: sort, target: a, witness: witness, additional: predicateRefs(t, f, i), lits: predicateRefs(t, f, lits...), original: lits, success: success, fresh: fresh, multi: multi, boolean: boolean, nestedRange: nestedRange}
}

func TestArrayProjectionSourceModelsVariablesAndRefusals(t *testing.T) {
	for _, mode := range arrayProjectionModes {
		t.Run(mode, func(t *testing.T) {
			c := arrayCase(t, mode)
			p := arrayProjectionOwner(t, c)
			m := c.f.r.manager
			arithmeticEvaluate(t, c.f, c.md, *c.lits)
			before := len(c.md.Core().ConstantDeclarations)
			initial := predicateValue(t, c.f, (*c.additional)[0])
			ok, err := p.ProjectVariable(c.md.Core(), c.target, c.additional, c.lits)
			fmMust(t, err)
			if ok != c.success {
				t.Fatal("projection result", ok, c.success)
			}
			if len(*c.additional) != 1+c.fresh || predicateValue(t, c.f, (*c.additional)[0]) != initial {
				t.Fatal("fresh variables", len(*c.additional), c.fresh)
			}
			if got := len(c.md.Core().ConstantDeclarations) - before; got != c.fresh {
				t.Fatal("model registrations", got, c.fresh)
			}
			if !ok {
				projectionExpect(t, c.f, *c.lits, c.original)
			} else {
				pred := predicateOwner(t, c.f)
				for _, r := range *c.lits {
					vars := predicateRefs(t, c.f)
					fmMust(t, pred.GetFreeVariables(predicateValue(t, c.f, r), vars))
					for _, v := range *vars {
						if predicateValue(t, c.f, v) == c.target {
							t.Fatal("array target remains")
						}
					}
				}
			}
			arithmeticEvaluate(t, c.f, c.md, *c.lits)
			if mode == "select-distinct" || mode == "select-same" {
				if len(*c.lits) != 4 {
					t.Fatal("partition constraints", len(*c.lits))
				}
			}
			if mode == "select-multi" && len(*c.lits) != 5 {
				t.Fatal("multidimensional partition constraints", len(*c.lits))
			}
			if mode == "select-distinct" {
				d, err := m.GetDecl(predicateValue(t, c.f, (*c.additional)[1]))
				fmMust(t, err)
				v, err := c.md.Core().ConstantInterpretation(d)
				fmMust(t, err)
				text, _, _, err := m.ArithmeticNumeralText(v)
				fmMust(t, err)
				if text != "7" {
					t.Fatal("select DFS registration order", text)
				}
			}
			fmMust(t, p.Close())
			arithmeticEvaluate(t, c.f, c.md, *c.lits)
		})
	}
}

func TestArrayProjectionUnknownIndicesRollback(t *testing.T) {
	f := newFMFixture(t)
	m := f.r.manager
	md := predicateModel(t, f)
	name, err := m.Symbols.Intern("OpaqueIndex")
	fmMust(t, err)
	u := f.own(m.MkUninterpretedSort(name, nil))
	sort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{u}), f.integer))
	a, b := f.constant("A", sort), f.constant("B", sort)
	i, j := f.constant("oi", u), f.constant("oj", u)
	v1, v2 := f.constant("opaque-value-1", u), f.constant("opaque-value-2", u)
	predicateRegister(t, f, md, i, v1)
	predicateRegister(t, f, md, j, v2)
	family, err := m.ArrayFamilyID()
	fmMust(t, err)
	store := func(base, index, value ast.ASTHandle) ast.ASTHandle {
		return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpStore, ast.SliceArguments([]ast.ASTHandle{base, index, value})))
	}
	definition := f.own(m.MkEq(store(store(a, i, f.number("3")), j, f.number("7")), b))
	lits := predicateRefs(t, f, definition, f.own(m.MkTrue()))
	extra := predicateRefs(t, f)
	c := &arrayProjectionCase{f: f, md: md, sort: sort, target: a}
	p := arrayProjectionOwner(t, c)
	comparison, err := p.compareValues(v1, v2)
	fmMust(t, err)
	if comparison != lbool.Undef {
		t.Fatal("unknown index comparison was guessed", comparison)
	}
	ok, err := p.ProjectVariable(md.Core(), a, extra, lits)
	fmMust(t, err)
	if ok || len(*extra) != 0 {
		t.Fatal("unknown store indices accepted")
	}
	projectionExpect(t, f, *lits, []ast.ASTHandle{definition, f.own(m.MkTrue())})
}

func TestArrayProjectionReuseNoopAndOwnerGuards(t *testing.T) {
	c := arrayCase(t, "direct-equality")
	p := arrayProjectionOwner(t, c)
	m := c.f.r.manager
	ok, err := p.ProjectVariable(c.md.Core(), c.target, c.additional, c.lits)
	fmMust(t, err)
	if !ok {
		t.Fatal("direct equality refused")
	}
	again := predicateRefs(t, c.f, c.original...)
	ok, err = p.ProjectVariable(c.md.Core(), c.target, c.additional, again)
	fmMust(t, err)
	if !ok {
		t.Fatal("reuse refused")
	}
	want := make([]ast.ASTHandle, len(*c.lits))
	for i, r := range *c.lits {
		want[i] = predicateValue(t, c.f, r)
	}
	projectionExpect(t, c.f, *again, want)
	ok, err = p.Solve(c.md.Core(), c.additional, c.lits)
	fmMust(t, err)
	if ok {
		t.Fatal("cheap solve hook must remain no-op")
	}
	family, err := p.FamilyID()
	fmMust(t, err)
	expected, err := m.ArrayFamilyID()
	fmMust(t, err)
	if family != expected {
		t.Fatal(family)
	}
	copy := *p
	if _, err = copy.FamilyID(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	other := newFMFixture(t)
	foreign := predicateModel(t, other)
	if _, err = p.ProjectVariable(foreign.Core(), c.target, c.additional, c.lits); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal(err)
	}
	if _, err = p.ProjectVariable(c.md.Core(), c.f.number("1"), c.additional, c.lits); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal(err)
	}
	fmMust(t, p.Close())
	if _, err = p.FamilyID(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
}

func TestArrayProjectionAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "select-distinct", "select-same", "select-shared", "select-bool", "select-multi", "select-array-range", "nested-select", "bare-disequality", "direct-equality", "store-equality", "store-repeated", "store-distinct", "store-multi", "select-store-refusal", "reuse", "trace", "closed-trace", "refusal-trace", "refusal-closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					mode := phase
					if phase == "constructor" || phase == "reuse" || strings.Contains(phase, "trace") {
						mode = "select-distinct"
					}
					if strings.HasPrefix(phase, "refusal-") {
						mode = "select-store-refusal"
					}
					c := arrayCase(t, mode)
					m := c.f.r.manager
					var trace bytes.Buffer
					if strings.Contains(phase, "trace") {
						var w io.Writer = &trace
						if strings.HasSuffix(phase, "closed-trace") {
							w = closedPredicateTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(w)
						tags.Enable("qe")
						t.Cleanup(tags.Finalize)
						c.f.r.config.Trace = tags
					}
					var p *ArrayProjection
					var err error
					if phase != "constructor" {
						p = arrayProjectionOwner(t, c)
					}
					if phase == "reuse" {
						_, err = p.ProjectVariable(c.md.Core(), c.target, c.additional, c.lits)
						fmMust(t, err)
						c.lits = predicateRefs(t, c.f, c.original...)
					}
					m.Budget().FailAfter(stop)
					if phase == "constructor" {
						p, err = NewArrayProjection(m, c.f.r.config)
					} else {
						_, err = p.ProjectVariable(c.md.Core(), c.target, c.additional, c.lits)
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
					t.Fatal("array projection allocation frontier", stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}
