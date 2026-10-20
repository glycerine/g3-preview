package rewriter

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func bigIntTest(text string) *big.Int {
	v, _ := new(big.Int).SetString(text, 10)
	return v
}

func bit2IntArithmeticNumeral(t *testing.T, m *ast.ASTManager, value string) ast.ASTHandle {
	t.Helper()
	s := m.NumeralService()
	n, err := s.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, s.Del(&n)) }()
	astTestMust(t, s.SetString(&n, value))
	h, err := m.ArithmeticNumeral(&n, true)
	astTestMust(t, err)
	astTestPin(t, m, h)
	return h
}

func TestBit2IntExtractionAlignmentAndCacheFoundation(t *testing.T) {
	m, _ := bvAddFixture(t)
	b, err := NewBit2Int(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	if bit2IntNumeralBits(bigIntTest("0")) != 1 || bit2IntNumeralBits(bigIntTest("-1")) != 1 || bit2IntNumeralBits(bigIntTest("8")) != 4 {
		t.Fatal("numeral bit widths")
	}

	three := bvAddNum(t, m, 3, 3)
	bv2int, err := m.MkBitVectorToInt(three)
	astTestMust(t, err)
	astTestPin(t, m, bv2int)
	if width, err := b.getB2ISize(bv2int); err != nil || width != 3 {
		t.Fatal("bv2int size", width, err)
	}
	call := bit2IntCall{owner: b}
	defer func() { astTestMust(t, call.close()) }()
	extracted, matched, err := b.extractBV(&call, bv2int)
	astTestMust(t, err)
	if !matched || extracted.bits != three || extracted.width != 3 || extracted.sign {
		t.Fatal("bv2int extraction", matched, extracted)
	}

	minusFive := bit2IntArithmeticNumeral(t, m, "-5")
	extracted, matched, err = b.extractBV(&call, minusFive)
	astTestMust(t, err)
	if !matched || extracted.width != 3 || !extracted.sign {
		t.Fatal("integer extraction", matched, extracted)
	}
	text, matched, err := m.BitVectorNumeralText(extracted.bits)
	astTestMust(t, err)
	if !matched || text != "3" {
		t.Fatal("negative numeral bit-vector normalization", matched, text)
	}

	fiveWide := bvAddNum(t, m, 7, 5)
	left, right, err := b.alignSizes(&call, three, fiveWide)
	astTestMust(t, err)
	if lw, err := m.BitVectorSize(left); err != nil || lw != 5 {
		t.Fatal("left aligned width", lw, err)
	}
	if rw, err := m.BitVectorSize(right); err != nil || rw != 5 || right != fiveWide {
		t.Fatal("right aligned width", rw, right, err)
	}

	astTestMust(t, b.cacheResult(bv2int, left))
	if len(b.cache) != 1 || len(b.keys) != 1 {
		t.Fatal("cache insertion", len(b.cache), len(b.keys))
	}
	astTestMust(t, b.flushCache())
	if len(b.cache) != 0 || len(b.keys) != 0 {
		t.Fatal("cache flush", len(b.cache), len(b.keys))
	}
}

func TestBit2IntArithmeticReductions(t *testing.T) {
	m, _ := bvAddFixture(t)
	b, err := NewBit2Int(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	call := bit2IntCall{owner: b}
	defer func() { astTestMust(t, call.close()) }()
	three := bvAddNum(t, m, 3, 3)
	five := bvAddNum(t, m, 5, 4)
	zero := bvAddNum(t, m, 0, 3)
	b3, err := m.MkBitVectorToInt(three)
	astTestMust(t, err)
	astTestPin(t, m, b3)
	b5, err := m.MkBitVectorToInt(five)
	astTestMust(t, err)
	astTestPin(t, m, b5)
	b0, err := m.MkBitVectorToInt(zero)
	astTestMust(t, err)
	astTestPin(t, m, b0)

	if result, ok, err := b.mkAdd(&call, b0, b5); err != nil || !ok || result != b5 {
		t.Fatal("zero addition", result, ok, err)
	}
	sum, ok, err := b.mkAdd(&call, b3, b5)
	astTestMust(t, err)
	if !ok {
		t.Fatal("addition declined")
	}
	if yes, err := m.IsBitVectorOperation(sum, 58); err != nil || !yes {
		t.Fatal("addition bv2int", yes, err)
	}
	sumBits, err := m.GetArg(sum, 0)
	astTestMust(t, err)
	if yes, err := m.IsBitVectorOperation(sumBits, 4); err != nil || !yes {
		t.Fatal("addition BV add", yes, err)
	}
	if width, err := m.BitVectorSize(sumBits); err != nil || width != 5 {
		t.Fatal("addition carry width", width, err)
	}

	for _, row := range []struct {
		mode bit2IntComparison
		kind int32
	}{
		{bit2IntLT, ast.ASTOpNot},
		{bit2IntLE, 22},
		{bit2IntEQ, ast.ASTOpEq},
	} {
		result, ok, err := b.mkComp(&call, row.mode, b3, b5)
		astTestMust(t, err)
		if !ok {
			t.Fatal("comparison declined", row.mode)
		}
		kind, err := m.GetDeclKind(result)
		astTestMust(t, err)
		if kind != row.kind {
			t.Fatal("comparison kind", row.mode, kind, row.kind)
		}
	}

	minusTwo := bit2IntArithmeticNumeral(t, m, "-2")
	product, ok, err := b.mkMul(&call, minusTwo, b3)
	astTestMust(t, err)
	if !ok {
		t.Fatal("multiplication declined")
	}
	if family, err := m.GetFamilyID(product); err != nil || family != b.arithFamily {
		t.Fatal("negative product family", family, err)
	}
	if kind, err := m.GetDeclKind(product); err != nil || kind != 8 {
		t.Fatal("negative product sign", kind, err)
	}
	unsigned, err := m.GetArg(product, 0)
	astTestMust(t, err)
	bits, err := m.GetArg(unsigned, 0)
	astTestMust(t, err)
	if yes, err := m.IsBitVectorOperation(bits, 6); err != nil || !yes {
		t.Fatal("product BV multiply", yes, err)
	}
	if width, err := m.BitVectorSize(bits); err != nil || width != 6 {
		t.Fatal("product doubled width", width, err)
	}
}
