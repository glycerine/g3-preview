package ast

// Source: src/ast/bv_decl_plugin.cpp:437-468,523-524,490-491,604-606.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTBitVectorOriginalBooleanBitConstructors(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	for _, width := range []uint32{0, 1, 8, 65} {
		domain := make([]ASTHandle, width)
		for i := range domain {
			domain[i] = b
		}
		d, e := m.MkFamilyFuncDecl(fid, 56, []ASTParameter{{}}, domain, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		rangeSort, e := m.GetRange(d)
		astTestMust(t, e)
		bits, ok, e := m.bitVectorSortWidth(fid, rangeSort)
		astTestMust(t, e)
		if !ok || bits != int32(width) {
			t.Fatal("mkbv result width", width, bits)
		}
		arity, e := m.GetArity(d)
		astTestMust(t, e)
		if arity != width {
			t.Fatal("mkbv domain", width, arity)
		}
		if width == 0 {
			continue
		}
		for _, index := range []int32{0, int32(width - 1)} {
			p, e := m.NewParameterInt(index)
			astTestMust(t, e)
			extract, e := m.MkFamilyFuncDecl(fid, 55, []ASTParameter{p}, []ASTHandle{rangeSort}, ASTHandle{})
			astTestMust(t, e)
			astTestPin(t, m, extract)
			r, e := m.GetRange(extract)
			astTestMust(t, e)
			if r != b {
				t.Fatal("bit2bool result")
			}
			m.Budget().FailAfter(0)
			again, e := m.MkFamilyFuncDecl(fid, 55, []ASTParameter{p}, []ASTHandle{rangeSort}, ASTHandle{})
			m.Budget().FailAfter(-1)
			astTestMust(t, e)
			if again != extract {
				t.Fatal("nested bit cache identity")
			}
			astTestMust(t, m.DelParameter(&p))
		}
	}
}

func TestASTBitVectorBooleanBitsApplicationsAndValidation(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	args := make([]ASTHandle, 18)
	for i := range args {
		args[i] = truth
	}
	value, e := m.MkFamilyApp(fid, 56, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, value)
	p, e := m.NewParameterInt(17)
	astTestMust(t, e)
	bit, e := m.MkFamilyApp(fid, 55, []ASTParameter{p}, []ASTHandle{value})
	astTestMust(t, e)
	astTestPin(t, m, bit)
	astTestMust(t, m.DelParameter(&p))
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	_, e = m.MkFamilyFuncDecl(fid, 56, nil, []ASTHandle{s}, ASTHandle{})
	if e == nil || e.Error() != "invalid mkbv operator" {
		t.Fatal("mkbv domain validation", e)
	}
	for _, index := range []int32{8, 99} {
		p, e := m.NewParameterInt(index)
		astTestMust(t, e)
		_, e = m.MkFamilyFuncDecl(fid, 55, []ASTParameter{p}, []ASTHandle{s}, ASTHandle{})
		if e == nil || e.Error() != "invalid bit2bool declaration" {
			t.Fatal("bit index guard", index, e)
		}
		astTestMust(t, m.DelParameter(&p))
	}
}

func TestASTBitVectorBooleanBitsEveryCacheReservationAndFinalize(t *testing.T) {
	for _, kind := range []int32{55, 56} {
		success := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			func() {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				s, e := m.BitVectorSort(65)
				astTestMust(t, e)
				p, e := m.NewParameterInt(64)
				astTestMust(t, e)
				ps := []ASTParameter{p}
				domain := []ASTHandle{s}
				if kind == 56 {
					ps = nil
					b, e := m.MkBoolSort()
					astTestMust(t, e)
					domain = make([]ASTHandle, 65)
					for i := range domain {
						domain[i] = b
					}
				}
				m.Budget().FailAfter(ordinal)
				_, body := m.MkFamilyFuncDecl(fid, kind, ps, domain, ASTHandle{})
				m.Budget().FailAfter(-1)
				if body != nil && !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal(kind, ordinal, body)
				}
				if body == nil {
					success = true
				}
				astTestMust(t, m.DelParameter(&p))
				astTestMust(t, m.SourceFinalize())
				if m.bitVectorPlugin != nil {
					t.Fatal("bit constructor cache owner retained")
				}
			}()
			if success {
				break
			}
		}
		if !success {
			t.Fatal("cache reservation horizon", kind)
		}
	}
}

func TestASTBitVectorBit2BoolGrowthPreservesRowsAndCheckedNegativeIndex(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	p, e := m.NewParameterInt(3)
	astTestMust(t, e)
	first, e := m.MkFamilyFuncDecl(fid, 55, []ASTParameter{p}, []ASTHandle{s}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, first)
	astTestMust(t, m.DelParameter(&p))
	s65, e := m.BitVectorSort(65)
	astTestMust(t, e)
	p, e = m.NewParameterInt(64)
	astTestMust(t, e)
	_, e = m.MkFamilyFuncDecl(fid, 55, []ASTParameter{p}, []ASTHandle{s65}, ASTHandle{})
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&p))
	saved := m.bitVectorPlugin.bit2bool[8][3]
	if m.external(saved) != first {
		t.Fatal("outer vector growth lost inner row ownership")
	}
	negative, e := m.NewParameterInt(-1)
	astTestMust(t, e)
	_, e = m.MkFamilyFuncDecl(fid, 55, []ASTParameter{negative}, []ASTHandle{s}, ASTHandle{})
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("checked source undefined index", e)
	}
	if len(m.bitVectorPlugin.bit2bool) != 66 || len(m.bitVectorPlugin.bit2bool[8]) != 8 {
		t.Fatal("original resize prefix must remain")
	}
	astTestMust(t, m.DelParameter(&negative))
}

func TestASTBitVectorBit2BoolPublishedBeforeRejectedINC(t *testing.T) {
	if !ASTSourceDebug {
		t.Skip("source debug counter boundary")
	}
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	parameter, e := m.NewParameterInt(2)
	astTestMust(t, e)
	ps := []ASTParameter{parameter}
	node, e := m.pluginDecl(m.bitVectorPlugin.names[8], fid, 55, ps, []ASTHandle{sort}, m.boolSort, ASTFuncFlags{}, false)
	astTestMust(t, e)
	m.nodeAt(node.slot).rc = ^uint32(0)
	_, e = m.MkFamilyFuncDecl(fid, 55, ps, []ASTHandle{sort}, ASTHandle{})
	p := m.bitVectorPlugin
	if e != ErrASTDomain || p.fixedPending != node || p.fixedPendingKind != 55 {
		t.Fatal("actual failed nested cache INC", e, p.fixedPending)
	}
	saved := p.bit2bool[8][2]
	if saved != node {
		t.Fatal("assignment must precede INC")
	}
	if e = p.finalizeFixedPending(false); e != ErrASTPending {
		t.Fatal("source pending acquisition guard", e)
	}
	astTestMust(t, p.finalizeFixedPending(true))
	if m.nodeAt(node.slot).rc != ^uint32(0) {
		t.Fatal("unacquired nested pointer was decremented")
	}
	m.nodeAt(node.slot).rc = 0
	_, e = m.MkFamilyFuncDecl(fid, 55, ps, []ASTHandle{sort}, ASTHandle{})
	astTestMust(t, e)
	astTestMust(t, m.DelParameter(&parameter))
}
