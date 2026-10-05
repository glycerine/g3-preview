// Licensed under the MIT license.
// Concrete sequence ownership tests; runtime binds actual source services.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
	"testing"
)

func TestUPolynomialSequenceCopyMoveAndRetainedRows(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{7, -9})
	seq := upolySturmSequence(t, m)
	original := mustUPolyView(t, m, p)
	if e := seq.PushCopy(original); e != nil {
		t.Fatal(e)
	}
	row, e := seq.RowView(0)
	if e != nil {
		t.Fatal(e)
	}
	before := seq.Stats()
	if before.BeginCount != 1 || before.SizeCount != 1 || before.CoefficientCount != 2 {
		t.Fatal(before)
	}
	// Metadata growth alone cannot invalidate the source coefficient pointer.
	empty := original.WithCount(0)
	for i := 0; i < 12; i++ {
		if e = seq.PushCopy(empty); e != nil {
			t.Fatal(e)
		}
	}
	var value mpnum.Int
	if e = m.ReadView(row, 1, &value); e != nil {
		t.Fatal(e)
	}
	v, e := m.IntegerManager().Int64(&value)
	if e != nil || v != -9 {
		t.Fatal(v, e)
	}
	if e = m.IntegerManager().Del(&value); e != nil {
		t.Fatal(e)
	}
	if e = seq.PushMove(original); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, p, []int64{0, 0}) // raw swap zeros; source size is unchanged
	upolySturmExpectRow(t, m, seq, 13, []int64{7, -9})
	if e = m.ReadView(row, 0, &value); !errors.Is(e, mpnum.ErrUPolynomialOwner) {
		t.Fatal("growth did not invalidate captured coefficients", e)
	}
	current, e := seq.RowView(13)
	if e != nil {
		t.Fatal(e)
	}
	capacityBefore := seq.Stats()
	used := b.Used()
	if e = seq.Reset(); e != nil {
		t.Fatal(e)
	}
	after := seq.Stats()
	if after.BeginCount != 0 || after.SizeCount != 0 || after.CoefficientCount != 0 || after.BeginCapacity != capacityBefore.BeginCapacity || after.SizeCapacity != capacityBefore.SizeCapacity || after.CoefficientCapacity != capacityBefore.CoefficientCapacity || b.Used() != used {
		t.Fatal(capacityBefore, after, used, b.Used())
	}
	if e = m.ReadView(current, 0, &value); e != nil {
		t.Fatal("retained initialized pointer", e)
	}
	v, e = m.IntegerManager().Int64(&value)
	if e != nil || v != 7 {
		t.Fatal(v, e)
	} // Del of inline value preserves m_val
	if e = m.IntegerManager().Del(&value); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialSequenceExactMetadataFailurePrefixes(t *testing.T) {
	for fault := int64(0); fault < 3; fault++ {
		m, _, b := upolySourceOwner(t)
		p := upolySourceNew(t, m, []int32{7})
		seq := upolySturmSequence(t, m)
		b.FailAfter(fault)
		e := seq.PushCopy(mustUPolyView(t, m, p))
		stats := seq.Stats()
		owned := m.Stats()
		// Audit before disabling faults/reading numerals/Close. All coefficients are
		// inline, so no sequence numeral cell justifies an unreferenced live root.
		if owned.IntegerLiveCells != owned.IntegerCachedCells+owned.OwnedNumeralCells || stats.OwnedNumeralCells != 0 {
			t.Fatal("immediate scalar graph", fault, owned, stats)
		}
		upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Sequences: []*mpnum.UPolynomialSequence{seq}})
		b.FailAfter(-1)
		if !numericStorageReservationRefusal(e) {
			t.Fatal(fault, e)
		}
		wantBegin, wantSize := uint32(0), uint32(0)
		if fault >= 1 {
			wantBegin = 1
		}
		if fault >= 2 {
			wantSize = 1
		}
		if stats.BeginCount != wantBegin || stats.SizeCount != wantSize || stats.CoefficientCount != 0 {
			t.Fatal("source metadata publication", fault, stats)
		}
		if fault == 2 {
			row, e := seq.RowView(0)
			if e != nil {
				t.Fatal(e)
			}
			var n mpnum.Int
			if e = m.ReadView(row, 0, &n); !errors.Is(e, mpnum.ErrUPolynomialOwner) && !errors.Is(e, arena.ErrBounds) {
				t.Fatal("unconstructed coefficient", e)
			}
		}
	}
}

func TestUPolynomialSequencePhysicalOwnerAndCloseGuard(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	before := b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(10, func() {
		seq, e := m.NewSequence()
		if !numericStorageReservationRefusal(e) || seq != nil {
			t.Fatal(seq, e)
		}
	})
	b.FailAfter(-1)
	if allocs != 0 || b.Used() != before {
		t.Fatal("constructor reserves before host allocation", allocs, b.Used(), before)
	}
	seq, e := m.NewSequence()
	if e != nil {
		t.Fatal(e)
	}
	if seq.Stats().OwnerBytes != uint64(reflect.TypeOf(seq).Elem().Size()) || b.Used() != before+seq.Stats().OwnerBytes {
		t.Fatal("physical cold owner", seq.Stats(), b.Used(), before)
	}
	if e = m.Close(); !errors.Is(e, mpnum.ErrUPolynomialLive) {
		t.Fatal("live sequence close guard", e)
	}
	copy := *seq
	if e = copy.Close(); !errors.Is(e, mpnum.ErrUPolynomialOwner) {
		t.Fatal("copied owner accepted", e)
	}
	if e = seq.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != before {
		t.Fatal("sequence owner refund", b.Used(), before)
	}
	var zero mpnum.UPolynomialSequence
	if e = zero.Close(); !errors.Is(e, mpnum.ErrUPolynomialOwner) {
		t.Fatal("zero owner", e)
	}
	if e = seq.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner", e)
	}
}

func TestUPolynomialHalfRootStackDivisorNoReservation(t *testing.T) {
	m, _, b, limit, _ := upolySourceOwnerServices(t)
	p := upolySourceNew(t, m, []int32{3, -1, -12, 4})
	out := upolySourceNew(t, m, []int32{1, 1, 1}) // source swaps an already-sized output into q scratch
	if e := m.RemoveOneHalfRoot(mustUPolyView(t, m, p), out); e != nil {
		t.Fatal(e)
	}
	before := m.Stats()
	used := b.Used()
	steps := limit.steps
	b.FailAfter(0)
	e := m.RemoveOneHalfRoot(mustUPolyView(t, m, p), out)
	after := m.Stats()
	finalUsed := b.Used()
	finalSteps := limit.steps
	b.FailAfter(-1)
	if e != nil || finalUsed != used || after.TotalBytes != before.TotalBytes || finalSteps-steps != 4 {
		t.Fatal("source lexical immediate divisor must not allocate", e, used, finalUsed, finalSteps-steps, before, after)
	}
	upolyGCDExpect(t, m, out, []int64{-24, -40, 16})
	b.FailAfter(0)
	var callError error
	allocs := testing.AllocsPerRun(20, func() { callError = m.RemoveOneHalfRoot(mustUPolyView(t, m, p), out) })
	b.FailAfter(-1)
	if callError != nil || allocs != 0 {
		t.Fatal("lexical pair escaped or reserved", allocs, callError)
	}
}

func TestUPolynomialSequenceBigCopyFailureImmediateGraph(t *testing.T) {
	failures, successes := 0, 0
	for fault := int64(0); fault < 64; fault++ {
		m, _, b := upolySourceOwner(t)
		p := upolySourceNew(t, m, []int32{0, 0, 1})
		seq := upolySturmSequence(t, m)
		var bigValue mpnum.Int
		digits := make([]uint32, 90)
		digits[89] = 1
		digits[0] = 37
		if e := m.IntegerManager().SetLimbs(&bigValue, digits); e != nil {
			t.Fatal(e)
		}
		for i := uint32(0); i < 3; i++ {
			if e := m.SetCoefficient(p, i, &bigValue); e != nil {
				t.Fatal(e)
			}
		}
		if e := m.IntegerManager().Del(&bigValue); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(fault)
		e := seq.PushCopy(mustUPolyView(t, m, p))
		managerGraph, sequenceGraph := m.Stats(), seq.Stats()
		if managerGraph.IntegerLiveCells != managerGraph.IntegerCachedCells+managerGraph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager())+sequenceGraph.OwnedNumeralCells {
			t.Fatal("immediate complete graph", fault, e, managerGraph, sequenceGraph)
		}
		upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Sequences: []*mpnum.UPolynomialSequence{seq}})
		b.FailAfter(-1)
		if e != nil {
			failures++
			if !numericStorageReservationRefusal(e) {
				t.Fatal(fault, e)
			}
		} else {
			successes++
			n, e := seq.Size()
			if e != nil || n != 1 {
				t.Fatal(n, e)
			}
			row, e := seq.RowView(0)
			if e != nil {
				t.Fatal(e)
			}
			for i := uint32(0); i < 3; i++ {
				var value mpnum.Int
				if e = m.ReadView(row, i, &value); e != nil {
					t.Fatal(e)
				}
				negative, actual, e := m.IntegerManager().Decompose(&value, nil)
				if e != nil || negative || len(actual) != 90 || actual[0] != 37 || actual[89] != 1 {
					t.Fatal(e)
				}
				if e = m.IntegerManager().Del(&value); e != nil {
					t.Fatal(e)
				}
			}
		}
		// Close must release failed-prefix live companion coefficients too, not
		// merely hide them by closing the borrowed numerical manager afterward.
		if e = seq.Reset(); e != nil {
			t.Fatal("prefix reset", fault, e)
		}
		managerGraph, sequenceGraph = m.Stats(), seq.Stats()
		if sequenceGraph.OwnedNumeralCells != 0 || managerGraph.IntegerLiveCells != managerGraph.IntegerCachedCells+managerGraph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager()) {
			t.Fatal("reset lost roots", fault, managerGraph, sequenceGraph)
		}
		if successes > 0 {
			break
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}

func TestUPolynomialTransformCancellationAfterComposition(t *testing.T) {
	m, q, _, limit, _ := upolySourceOwnerServices(t)
	p := upolySourceNew(t, m, []int32{1, 2, 3})
	c := mpnum.NewRational()
	if e := q.SetFraction(&c, 1, 3); e != nil {
		t.Fatal(e)
	}
	defer q.Del(&c)
	limit.incCancel()
	steps := limit.steps
	e := m.TranslateRationalViewInPlace(mustUPolyView(t, m, p), &c)
	graph := m.Stats()
	if !errors.Is(e, mpnum.ErrUPolynomialCanceled) || limit.steps != steps+1 || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager()) {
		t.Fatal("actual canceled checkpoint prefix", e, limit.steps, steps, graph)
	}
	upolyGCDExpect(t, m, p, []int64{9, 6, 3}) // denominator composition BEFORE checkpoint
}

func TestUPolynomialSturmFailureAndCancellationImmediateGraph(t *testing.T) {
	failures, successes := 0, 0
	for fault := int64(0); fault < 2048; fault++ {
		m, _, budget := upolySourceOwner(t)
		p := upolySourceNew(t, m, []int32{8, 2, 8, 10, 10, 0, 1, 0, 1, 3, 7})
		seq := upolySturmSequence(t, m)
		budget.FailAfter(fault)
		e := m.SturmSequence(mustUPolyView(t, m, p), seq)
		graph, external := m.Stats(), seq.Stats()
		if graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager())+external.OwnedNumeralCells {
			t.Fatal("Sturm immediate roots", fault, e, graph, external)
		}
		upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Sequences: []*mpnum.UPolynomialSequence{seq}})
		budget.FailAfter(-1)
		if e != nil {
			failures++
			if !numericStorageReservationRefusal(e) {
				t.Fatal(fault, e)
			}
		} else {
			successes++
			size, e := seq.Size()
			if e != nil || size != 11 {
				t.Fatal(size, e)
			}
		}
		if e = seq.Reset(); e != nil {
			t.Fatal("failed-prefix cleanup", fault, e)
		}
		graph, external = m.Stats(), seq.Stats()
		if external.OwnedNumeralCells != 0 || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager()) {
			t.Fatal("lost local", fault, graph, external)
		}
		if successes != 0 {
			break
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
	m, _, _, limit, _ := upolySourceOwnerServices(t)
	p := upolySourceNew(t, m, []int32{-1, 0, 1})
	seq := upolySturmSequence(t, m)
	limit.incCancel()
	steps := limit.steps
	e := m.SturmSequence(mustUPolyView(t, m, p), seq)
	graph, external := m.Stats(), seq.Stats()
	if !errors.Is(e, mpnum.ErrUPolynomialCanceled) || limit.steps != steps+1 || graph.IntegerLiveCells != graph.IntegerCachedCells+graph.OwnedNumeralCells+upolyIntegerRawOwnedCells(t, m.IntegerManager())+external.OwnedNumeralCells {
		t.Fatal("canceled remainder", e, limit.steps, graph, external)
	}
	size, e := seq.Size()
	if e != nil || size != 2 {
		t.Fatal(size, e)
	}
	upolySturmExpectRow(t, m, seq, 0, []int64{-1, 0, 1})
	upolySturmExpectRow(t, m, seq, 1, []int64{0, 2})
	t.Logf("real reservation sweep: %d failures, %d success; canceled remainder preserves both published rows", failures, successes)
}

func TestUPolynomialSequenceSelfGrowthCloseAndAscendingBigReset(t *testing.T) {
	for _, move := range []bool{false, true} {
		m, _, _ := upolySourceOwner(t)
		p := upolySourceNew(t, m, []int32{7, -9})
		seq := upolySturmSequence(t, m)
		if e := seq.PushCopy(mustUPolyView(t, m, p)); e != nil {
			t.Fatal(e)
		}
		captured, e := seq.RowView(0)
		if e != nil {
			t.Fatal(e)
		}
		if move {
			e = seq.PushMove(captured)
		} else {
			e = seq.PushCopy(captured)
		}
		prefix := seq.Stats()
		if !errors.Is(e, mpnum.ErrUPolynomialOwner) || prefix.BeginCount != 2 || prefix.SizeCount != 2 || prefix.CoefficientCount != 3 {
			t.Fatal("growth frees source before next read", move, e, prefix)
		}
		upolySturmExpectRow(t, m, seq, 0, []int64{7, -9})
	}
	m, _, budget := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1})
	seq, e := m.NewSequence()
	if e != nil {
		t.Fatal(e)
	}
	if e = seq.PushCopy(mustUPolyView(t, m, p)); e != nil {
		t.Fatal(e)
	}
	captured, e := seq.RowView(0)
	if e != nil {
		t.Fatal(e)
	}
	if e = seq.Close(); e != nil {
		t.Fatal(e)
	}
	var value mpnum.Int
	if e = m.ReadView(captured, 0, &value); !errors.Is(e, mpnum.ErrUPolynomialOwner) {
		t.Fatal("closed captured view", e)
	}
	seq = upolySturmSequence(t, m)
	if e = m.SetInts(p, []mpnum.Int{mpnum.NewInt(1), mpnum.NewInt(1), mpnum.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	digits := []uint32{31, 17, 1}
	var bigValue mpnum.Int
	if e = m.IntegerManager().SetLimbs(&bigValue, digits); e != nil {
		t.Fatal(e)
	}
	for i := uint32(0); i < 3; i++ {
		if e = m.SetCoefficient(p, i, &bigValue); e != nil {
			t.Fatal(e)
		}
	}
	if e = m.IntegerManager().Del(&bigValue); e != nil {
		t.Fatal(e)
	}
	if e = seq.PushCopy(mustUPolyView(t, m, p)); e != nil {
		t.Fatal(e)
	}
	// Read-only reflection inspects actual scalar cell IDs; it does not copy
	// numeric ownership, call an implementation cleanup helper or mutate state.
	state := reflect.ValueOf(seq).Elem().FieldByName("state").FieldByName("coefficients")
	backing := state.FieldByName("backing").Uint()
	values := reflect.ValueOf(m).Elem().FieldByName("store").Elem().FieldByName("backings").Index(int(backing - 1)).FieldByName("values")
	slots := [3]uint32{}
	for i := range slots {
		slots[i] = uint32(values.Index(i).FieldByName("cell").Uint())
		if slots[i] == 0 {
			t.Fatal("not big")
		}
	}
	used := budget.Used()
	physicalBefore := m.IntegerManager().RuntimeOwner().Stats().TotalBytes
	arraysBefore := seq.Stats().TotalBytes
	if e = seq.Reset(); e != nil {
		t.Fatal(e)
	}
	physicalAfter := m.IntegerManager().RuntimeOwner().Stats().TotalBytes
	if budget.Used() != used-(physicalBefore-physicalAfter) || seq.Stats().TotalBytes != arraysBefore || seq.Stats().OwnedNumeralCells != 0 {
		t.Fatal("reset retains arrays and frees owned roots", used, budget.Used(), seq.Stats())
	}
	var probes [3]mpnum.Int
	for i := range probes {
		if e = m.IntegerManager().SetLimbs(&probes[i], digits); e != nil {
			t.Fatal(e)
		}
		actual := uint32(reflect.ValueOf(probes[i]).FieldByName("cell").Uint())
		if actual != slots[2-i] {
			t.Fatal("actual ascending release/LIFO reuse", i, actual, slots)
		}
	}
	for i := range probes {
		if e = m.IntegerManager().Del(&probes[i]); e != nil {
			t.Fatal(e)
		}
	}
}
