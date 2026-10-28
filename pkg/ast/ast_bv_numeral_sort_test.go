package ast

// Source: src/ast/bv_decl_plugin.cpp:774-776,858-864.

import "testing"

func bvRecognizerTestSort(t *testing.T, m *ASTManager, fid FamilyID, kind int32, ps []ASTParameter) ASTHandle {
	t.Helper()
	info, e := m.NewSortInfo(fid, kind, ps, SortSize{}, false)
	astTestMust(t, e)
	s, e := m.MkSort(astTestSymbol(t, m, "raw-bv-recognizer-sort"), info)
	astTestMust(t, e)
	astTestMust(t, m.DelSortInfo(&info))
	astTestPin(t, m, s)
	return s
}

func TestASTBitVectorNumeralForSortGuardErasesNumericalRead(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterInt(8)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelParameter(&p)) }()
	for _, s := range []ASTHandle{
		bvRecognizerTestSort(t, m, fid, 0, nil),
		bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{p, p}),
		bvRecognizerTestSort(t, m, fid, 1, []ASTParameter{p}),
		bvRecognizerTestSort(t, m, arith, 0, []ASTParameter{p}),
	} {
		before := m.Stats()
		m.Budget().FailAfter(0)
		isBV, e := m.IsBitVectorSort(s)
		astTestMust(t, e)
		value, e := m.BitVectorNumeralForSort(nil, s)
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if isBV || !value.IsNull() || m.Stats() != before {
			t.Fatal("unmatched sort read numerical input", isBV, value)
		}
	}
}

func TestASTBitVectorNumeralForSortOriginalWidthAndIdentity(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)) }()
	astTestMust(t, m.numerals.SetInt64(&n, -1))
	for _, width := range []uint32{1, 8, 65, 257} {
		s, e := m.BitVectorSort(width)
		astTestMust(t, e)
		yes, e := m.IsBitVectorSort(s)
		astTestMust(t, e)
		if !yes {
			t.Fatal("canonical BV sort rejected", width)
		}
		got, e := m.BitVectorNumeralForSort(&n, s)
		astTestMust(t, e)
		astTestPin(t, m, got)
		want, e := m.BitVectorNumeral(&n, width)
		astTestMust(t, e)
		if got != want {
			t.Fatal("numeral overload identity", width, got, want)
		}
	}
	// The overload takes the width, not the raw sort's name/identity.
	p, e := m.NewParameterInt(8)
	astTestMust(t, e)
	raw := bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{p})
	astTestMust(t, m.DelParameter(&p))
	got, e := m.BitVectorNumeralForSort(&n, raw)
	astTestMust(t, e)
	resultSort, e := m.GetSort(got)
	astTestMust(t, e)
	wantSort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	if resultSort != wantSort || resultSort == raw {
		t.Fatal("overload retained raw sort instead of constructing by width")
	}
}

func TestASTBitVectorSortRecognizerDoesNotReadParameterKind(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterSymbol(astTestSymbol(t, m, "invalid-bv-width"))
	astTestMust(t, e)
	s := bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{p})
	astTestMust(t, m.DelParameter(&p))
	m.Budget().FailAfter(0)
	yes, e := m.IsBitVectorSort(s)
	astTestMust(t, e)
	_, body := m.BitVectorNumeralForSort(nil, s)
	m.Budget().FailAfter(-1)
	if !yes || body != ErrASTDomain {
		t.Fatal("recognizer/width getter order", yes, body)
	}
}

func TestASTBitVectorNumeralForSortZeroWidthPreservesInputCopy(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p, e := m.NewParameterInt(0)
	astTestMust(t, e)
	s := bvRecognizerTestSort(t, m, fid, 0, []ASTParameter{p})
	astTestMust(t, m.DelParameter(&p))
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.Del(&n)) }()
	astTestMust(t, m.numerals.SetInt64(&n, 17))
	h, body := m.BitVectorNumeralForSort(&n, s)
	if !h.IsNull() || body == nil || body.Error() != "bit-vector size must be greater than zero" {
		t.Fatal("inner numeral zero-width guard", h, body)
	}
	f := m.bitVectorPlugin.numeralFrames
	if f == nil {
		t.Fatal("missing input parameter copy before inner guard")
	}
	copyBody, copySource, copyOwn := f.children[2].Causes()
	if copyBody != nil || copySource != nil || copyOwn != nil {
		t.Fatal("input copy did not complete before inner guard", copyBody, copySource, copyOwn)
	}
}
