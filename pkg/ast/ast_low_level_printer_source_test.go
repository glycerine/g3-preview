// Source: src/ast/ast_ll_pp.cpp:168-184,217-236,292-297;
// src/ast/for_each_ast.h:40-125.
package ast

import (
	bigint "math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
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
	if m.LowLevel.Active || m.LowLevel.capacity != 0 || m.LowLevel.marks.exprs.bytes != 0 || m.LowLevel.marks.decls.bytes != 0 {
		t.Fatal("source lexical stack/mark DT lost roots")
	}
}

func TestASTLowLevelPrinterDeclarationMarksUseActualDeclarationIndex(t *testing.T) {
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
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, m.DisplayLowLevel(out, a, true, true))
	got, e := out.String()
	astTestMust(t, e)
	if got != "bounded-decl\n" {
		t.Fatal("source declaration bitmap uses get_decl_id, not tagged global id", got)
	}
	if b.Used() != m.Stats().AllocatedBytes {
		t.Fatal("source lexical bitmap graph", b.Used(), m.Stats())
	}
}

func TestASTLowLevelPrinterNativeLargeNumeralFormatting(t *testing.T) {
	bn, ba, bs := arena.NewBudget(1<<28), arena.NewBudget(1<<24), arena.NewBudget(1<<20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: bn, ThreadLocal: true, SizeTBits: 64})
	paramsMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: bs})
	paramsMust(t, e)
	anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused") }, Cooperation: func() error { panic("unused") }})
	paramsMust(t, e)
	scoped, e := util.NewScopedInt(anchor.IntegerManager())
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
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: ba})
	paramsMust(t, e)
	paramsMust(t, m.DisplayLowLevel(out, app, true, true))
	text, e := out.String()
	paramsMust(t, e)
	want := new(bigint.Int).Add(new(bigint.Int).Lsh(bigint.NewInt(1), 64*32), bigint.NewInt(7)).String()
	if !strings.Contains(text, want) || m.LowLevel.Active || m.LowLevel.numeralInsertion.Active() {
		t.Fatal("native big numeral output/lifetime", text)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
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
