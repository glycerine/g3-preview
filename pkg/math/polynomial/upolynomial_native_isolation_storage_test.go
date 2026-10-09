package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestUPolynomialNativeBinaryGrowthMovesBIGAndDefaults(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	v := upolyIsolationVector(t, b)
	var numeral util.Int
	serviceMust(t, q.Integers().SetUint64(&numeral, 1<<40|1))
	value := util.NewBinaryRational()
	serviceMust(t, b.SetIntegerPower(&value, &numeral, 7))
	serviceMust(t, q.Integers().Del(&numeral))
	serviceMust(t, v.PushCopy(&value))
	serviceMust(t, b.Del(&value))
	token := v.state.values[0].BorrowNumerator().NativeCellToken()
	old := v.state.values
	used, before := budget.Used(), q.Integers().Memory.Stats()
	cells := q.Integers().StorageStats()
	budget.FailAfter(0)
	t.Cleanup(func() { budget.FailAfter(-1) })
	serviceMust(t, v.Resize(4097))
	for _, row := range old {
		if row != (util.BinaryRational{}) {
			t.Fatal("relocated old binary field retains ownership", row)
		}
	}
	if v.state.values[0].BorrowNumerator().NativeCellToken() != token || v.state.values[0].K() != 7 || !q.Integers().IsLiveCellToken(token) {
		t.Fatal("native relocation lost BIG binary ownership")
	}
	for i := 1; i < len(v.state.values); i++ {
		if v.state.values[i] != (util.BinaryRational{}) {
			t.Fatal("default field", i)
		}
	}
	if budget.Used() != used || q.Integers().Memory.Stats() != before || q.Integers().StorageStats() != cells || v.Stats().BackingBytes != 0 {
		t.Fatal("native binary growth charged numeric storage")
	}
	serviceMust(t, v.Shrink(1))
	serviceMust(t, v.Resize(129))
	serviceMust(t, v.Reset())
	if q.Integers().IsLiveCellToken(token) {
		t.Fatal("reset retained actual BIG binary numeral")
	}
	serviceMust(t, v.Reserve(33))
	for i := uint32(0); i < 33; i++ {
		row, e := numericBinaryRationalManager_binarySlot(b, &v.state, i)
		serviceMust(t, e)
		if *row != (util.BinaryRational{}) {
			t.Fatal("reset/reextend default", i, *row)
		}
	}
	_ = m
}

func TestUPolynomialNativeIsolationStacksAndNumericTransfer(t *testing.T) {
	m, q, budget := upolySourceOwner(t)
	b := upolyIsolationBinaryOwner(t, q, budget)
	var ss upolySSStack
	defer numericBinaryRationalManager_dropSS(b, &ss)
	first, e := numericBinaryRationalManager_pushSS(b, &ss)
	serviceMust(t, e)
	serviceMust(t, q.Integers().SetUint64(first.lower.BorrowNumerator(), 1<<40))
	serviceMust(t, q.Integers().SetUint64(first.upper.BorrowNumerator(), 1<<41))
	lower, upper := first.lower.BorrowNumerator().NativeCellToken(), first.upper.BorrowNumerator().NativeCellToken()
	old := ss.values
	used, before := budget.Used(), q.Integers().Memory.Stats()
	cells := q.Integers().StorageStats()
	budget.FailAfter(0)
	t.Cleanup(func() { budget.FailAfter(-1) })
	var drs upolyDRSStack
	defer m.dropDRS(&drs)
	for i := uint32(0); i < 4097; i++ {
		row, e := numericBinaryRationalManager_pushSS(b, &ss)
		serviceMust(t, e)
		row.lowerSV, row.upperSV = i, i+1
		serviceMust(t, m.pushDRS(&drs, newUPolyDRSFrame(i, 0, i%2 == 0)))
	}
	for _, row := range old {
		if row != (upolySSFrame{}) {
			t.Fatal("relocated old Sturm frame retains ownership")
		}
	}
	for i := uint32(0); i < 4097; i++ {
		row := ss.values[i+1]
		if row.lowerSV != i || row.upperSV != i+1 || row.lower != (util.BinaryRational{}) || row.upper != (util.BinaryRational{}) {
			t.Fatal("native Sturm frame", i, row)
		}
		frame := drs.values[i]
		if frame.parent != i || frame.polynomialSize() != 0 || !frame.firstVisit() || frame.leftChild() != (i%2 == 0) {
			t.Fatal("native DRS frame", i, frame)
		}
	}
	if ss.values[0].lower.BorrowNumerator().NativeCellToken() != lower || ss.values[0].upper.BorrowNumerator().NativeCellToken() != upper || budget.Used() != used || q.Integers().Memory.Stats() != before || q.Integers().StorageStats() != cells {
		t.Fatal("native stacks copied/lost numeric ownership")
	}
	var coefficients upolyVectorState
	for len(drs.values) != 0 {
		serviceMust(t, m.popDRS(&coefficients, &drs))
	}
	for len(ss.values) > 1 {
		serviceMust(t, numericBinaryRationalManager_popSS(b, &ss))
	}
	serviceMust(t, numericBinaryRationalManager_popSS(b, &ss))
	if q.Integers().IsLiveCellToken(lower) || q.Integers().IsLiveCellToken(upper) {
		t.Fatal("Sturm pop retained numeric endpoints")
	}
}
