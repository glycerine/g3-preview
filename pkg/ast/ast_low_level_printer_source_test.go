package ast

// Source: src/ast/ast_ll_pp.cpp:168-184,217-236,292-297;
// src/ast/for_each_ast.h:40-125.

import (
	"errors"
	bigint "math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTLowLevelPrinterActualPostorderAndBorrowedStreamFlags(t *testing.T) {
	m := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(7, bs)
	astTestMust(t, e)
	astTestPin(t, m, v)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{bs}, bs, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	a, e := m.MkApp(d, []ASTHandle{v})
	astTestMust(t, e)
	astTestPin(t, m, a)
	b := arena.NewBudget(1 << 20)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, out.SetShowPos(true))
	astTestMust(t, m.DisplayLowLevel(out, a, true, true))
	id, e := m.GetASTID(v)
	astTestMust(t, e)
	want := "#" + strconv.FormatUint(uint64(id), 10) + " := (:var 7 Bool)\n(f #" + strconv.FormatUint(uint64(id), 10) + ")\n"
	got, e := out.String()
	astTestMust(t, e)
	if got != want {
		t.Fatal("literal postorder, unsigned field formatting", got, want)
	}
	if m.LowLevel.Active || m.LowLevel.stack != nil || m.LowLevel.marks != nil {
		t.Fatal("source lexical stack/mark DT lost roots")
	}
}

func TestASTLowLevelPrinterNativeDeclarationIdentityMarks(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 20)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	defer m.Close()
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "bounded-decl"), nil, bs, ASTFuncDeclInfo{})
	astTestMust(t, e)
	a, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(a))
	defer func() { astTestMust(t, m.DecRef(a)) }()
	sb := arena.NewBudget(1 << 20)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: sb})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, m.DisplayLowLevel(out, a, true, true))
	got, e := out.String()
	astTestMust(t, e)
	if got != "bounded-decl\n" {
		t.Fatal("native declaration identity produces exact compact output", got)
	}
	if b.Used() != m.Stats().AllocatedBytes {
		t.Fatal("native lexical teardown budget", b.Used(), m.Stats())
	}
}

func TestASTLowLevelPrinterNativeLargeNumeralFormatting(t *testing.T) {
	bn, ba, bs := arena.NewBudget(1<<28), arena.NewBudget(1<<24), arena.NewBudget(1<<20)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: bn, ThreadLocal: true, SizeTBits: 64})
	paramsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs})
	paramsMust(t, e)
	anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused") }, Cooperation: func() error { panic("unused") }})
	paramsMust(t, e)
	scoped, e := unumeral.NewScopedInt(anchor.IntegerManager())
	paramsMust(t, e)
	value, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 65)
	limbs[0], limbs[64] = 7, 1
	paramsMust(t, anchor.IntegerManager().SetLimbs(value, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: ba})
	paramsMust(t, e)
	sort, e := m.MkBoolSort()
	paramsMust(t, e)
	rat, e := m.NewParameterRational(&big)
	paramsMust(t, e)
	integer, e := m.NewParameterInt(1)
	paramsMust(t, e)
	arith, e := sy.Intern("arith")
	paramsMust(t, e)
	family, e := m.families.MkID(arith)
	paramsMust(t, e)
	info, e := m.NewFuncDeclInfo(family, 0, []ASTParameter{rat, integer}, ASTFuncFlags{}, false)
	paramsMust(t, e)
	symbol, e := sy.Intern("numeral")
	paramsMust(t, e)
	decl, e := m.MkFuncDecl(symbol, nil, sort, info)
	paramsMust(t, e)
	app, e := m.MkApp(decl, nil)
	paramsMust(t, e)
	paramsMust(t, m.IncRef(app))
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: ba})
	paramsMust(t, e)
	paramsMust(t, m.DisplayLowLevel(out, app, true, true))
	text, e := out.String()
	paramsMust(t, e)
	want := new(bigint.Int).Add(new(bigint.Int).Lsh(bigint.NewInt(1), 64*32), bigint.NewInt(7)).String()
	if !strings.Contains(text, want) || m.LowLevel.Active || m.LowLevel.numeralInsertion.Active() {
		t.Fatal("native big numeral output/lifetime", text)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*unumeral.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	paramsMust(t, m.SourceFinalize())
	paramsMust(t, m.Close())
	paramsMust(t, out.Close())
	paramsMust(t, q.Del(&big))
	paramsMust(t, anchor.Close())
	paramsMust(t, q.Close())
	paramsMust(t, r.Close())
	paramsMust(t, sy.Close())
	if bn.Used() != 0 || ba.Used() != 0 || bs.Used() != 0 {
		t.Fatal("owner refunds", bn.Used(), ba.Used(), bs.Used())
	}
}

func TestASTLowLevelPrinterNativeWideAndDeepPostorder(t *testing.T) {
	m := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	arith, e := m.Symbols.Intern("arith")
	astTestMust(t, e)
	_, e = m.families.MkID(arith)
	astTestMust(t, e)
	keys := markTestNodes(t, m, 257)
	var args, domains []ASTHandle
	for _, key := range keys {
		args = append(args, key, key)
		domains = append(domains, bs, bs)
	}
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "wide"), domains, bs, ASTFuncDeclInfo{})
	astTestMust(t, e)
	root, e := m.MkApp(decl, args)
	astTestMust(t, e)
	astTestPin(t, m, root)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: arena.NewBudget(1 << 24)})
	astTestMust(t, e)
	defer out.Close()
	ref := func(h ASTHandle) string {
		id, e := m.GetASTID(h)
		astTestMust(t, e)
		return "#" + strconv.FormatUint(uint64(id), 10)
	}
	var want strings.Builder
	for i := len(keys) - 1; i >= 0; i-- {
		want.WriteString(ref(keys[i]) + " := (:var " + strconv.Itoa(i+30000) + " Bool)\n")
	}
	want.WriteString("(wide")
	for _, key := range args {
		want.WriteString(" " + ref(key))
	}
	want.WriteString(")\n")
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	e = m.DisplayLowLevel(out, root, true, true)
	m.BudgetField.FailAfter(-1)
	astTestMust(t, e)
	got, e := out.String()
	astTestMust(t, e)
	if got != want.String() {
		t.Fatal("wide shared postorder differs", got)
	}
	for _, key := range keys {
		rc, e := m.GetRefCount(key)
		astTestMust(t, e)
		if rc != 2 {
			t.Fatal("borrowed traversal changed root references", rc)
		}
	}
	if m.BudgetField.Used() != before || m.LowLevel.Active || m.LowLevel.stack != nil || m.LowLevel.marks != nil {
		t.Fatal("native wide lexical storage retained or charged")
	}
	// A separately shaped chain exercises stack depth and quantifier traversal.
	leaf, e := m.MkVar(0, bs)
	astTestMust(t, e)
	chain := []ASTHandle{leaf}
	name := astTestSymbol(t, m, "x")
	for i := 0; i < 257; i++ {
		q, e := m.MkQuantifier(ASTQuantifierSpec{Forall: true, Sorts: []ASTHandle{bs}, Names: []usymbol.Symbol{name}, Body: chain[len(chain)-1]})
		astTestMust(t, e)
		chain = append(chain, q)
	}
	astTestPin(t, m, chain[len(chain)-1])
	want.Reset()
	want.WriteString(ref(leaf) + " := (:var 0 Bool)\n")
	for i := 1; i < len(chain); i++ {
		if i+1 < len(chain) {
			want.WriteString(ref(chain[i]) + " := ")
		}
		want.WriteString("(forall (vars (x Bool)) " + ref(chain[i-1]) + ")\n")
	}
	for attempt := 0; attempt < 2; attempt++ {
		astTestMust(t, out.ResetText())
		before = m.BudgetField.Used()
		m.BudgetField.FailAfter(0)
		e = m.DisplayLowLevel(out, chain[len(chain)-1], true, true)
		m.BudgetField.FailAfter(-1)
		astTestMust(t, e)
		got, e = out.String()
		astTestMust(t, e)
		if got != want.String() {
			t.Fatal("deep quantifier postorder differs", got)
		}
		if m.BudgetField.Used() != before || m.LowLevel.Active || m.LowLevel.stack != nil || m.LowLevel.marks != nil {
			t.Fatal("native deep lexical storage retained or charged")
		}
	}
	astTestMust(t, out.Close())
	e = m.DisplayLowLevel(out, chain[len(chain)-1], true, true)
	if !errors.Is(e, arena.ErrClosed) || m.LowLevel.Active || m.LowLevel.stack != nil || m.LowLevel.marks != nil {
		t.Fatal("output failure retained native lexical traversal", e)
	}
	fresh, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: arena.NewBudget(1 << 24)})
	astTestMust(t, e)
	defer fresh.Close()
	astTestMust(t, m.DisplayLowLevel(fresh, chain[len(chain)-1], true, true))
	got, e = fresh.String()
	astTestMust(t, e)
	if got != want.String() {
		t.Fatal("failed traversal poisoned subsequent invocation", got)
	}
}
