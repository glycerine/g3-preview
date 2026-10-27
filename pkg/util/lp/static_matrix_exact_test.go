package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpExactStaticCheck(t *testing.T, h *lpPairHarness, m *RationalStaticMatrix, want map[IndexPair]*big.Rat) {
	t.Helper()
	seen := make(map[IndexPair]bool)
	for i, row := range m.Rows {
		for j, c := range row {
			p := IndexPair{uint32(i), c.Column}
			if seen[p] || want[p] == nil {
				t.Fatal("unexpected/duplicate", p)
			}
			seen[p] = true
			lpSparseRationalCheck(t, h, &m.Rows[i][j].Value, want[p])
			if int(c.Column) >= len(m.Columns) || int(c.Offset) >= len(m.Columns[c.Column]) {
				t.Fatal("row link bounds")
			}
			cc := m.Columns[c.Column][c.Offset]
			if cc.Row != uint32(i) || cc.Offset != uint32(j) {
				t.Fatal("row backlink", i, j, c, cc)
			}
		}
	}
	if len(seen) != len(want) || int(m.Size()) != len(want) {
		t.Fatal("missing coordinates", len(seen), len(want), m.Size())
	}
	n := 0
	for j, col := range m.Columns {
		for k, c := range col {
			if int(c.Row) >= len(m.Rows) || int(c.Offset) >= len(m.Rows[c.Row]) {
				t.Fatal("column bounds")
			}
			rc := m.Rows[c.Row][c.Offset]
			if rc.Column != uint32(j) || rc.Offset != uint32(k) {
				t.Fatal("column backlink")
			}
			v, e := m.ColumnValue(c)
			h.must(e)
			lpSparseRationalCheck(t, h, v, want[IndexPair{c.Row, uint32(j)}])
			n++
		}
	}
	if n != len(want) {
		t.Fatal("column size", n, len(want))
	}
}

func TestLPExactStaticMatrixIndependentMutationOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	m := NewRationalStaticMatrix(h.m, 5, 7)
	t.Cleanup(func() { h.must(m.Close()) })
	p := h.pair("0", "0")
	r := rand.New(rand.NewPCG(721, 9875))
	want := make(map[IndexPair]*big.Rat)
	for step := 0; step < 1800; step++ {
		i, j := uint32(r.IntN(5)), uint32(r.IntN(7))
		key := IndexPair{i, j}
		switch r.IntN(5) {
		case 0, 1:
			if want[key] != nil {
				continue
			}
			num := new(big.Int).Lsh(big.NewInt(int64(r.IntN(19)-9)), uint(r.IntN(120)))
			x := new(big.Rat).SetFrac(num, big.NewInt(int64(1+r.IntN(39))))
			h.must(h.m.SetString(&p.X, x.RatString()))
			if step%2 == 0 {
				h.must(m.Set(i, j, &p.X))
				if x.Sign() != 0 {
					want[key] = x
				}
			} else {
				h.must(m.AddNewElement(i, j, &p.X))
				want[key] = x
			}
		case 2:
			if len(m.Rows[i]) > 0 {
				off := uint32(r.IntN(len(m.Rows[i])))
				delete(want, IndexPair{i, m.Rows[i][off].Column})
				h.must(m.RemoveElement(i, off))
			}
		case 3, 4:
			a := new(big.Rat).SetFrac(big.NewInt(int64(r.IntN(7)-3)), big.NewInt(int64(1+r.IntN(9))))
			h.must(h.m.SetString(&p.Y, a.RatString()))
			if step%2 == 0 {
				h.must(m.MultiplyRow(i, &p.Y))
				for k, v := range want {
					if k.I == i {
						v.Mul(v, a)
					}
				}
			} else {
				h.must(m.MultiplyColumn(j, &p.Y))
				for k, v := range want {
					if k.J == j {
						v.Mul(v, a)
					}
				}
			}
		}
		lpExactStaticCheck(t, h, m, want)
	}
	for i := uint32(0); i < 5; i++ {
		for j := uint32(0); j < 8; j++ {
			h.must(m.Element(i, j, &p.X))
			v := want[IndexPair{i, j}]
			if v == nil {
				v = new(big.Rat)
			}
			lpSparseRationalCheck(t, h, &p.X, v)
		}
	}
	h.must(m.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactStaticMatrixIndependentPivotOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	r := rand.New(rand.NewPCG(899, 53))
	multi := 0
	for world := 0; world < 240; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "0")
			m := NewRationalStaticMatrix(h.m, 2, 8)
			t.Cleanup(func() { h.must(m.Close()) })
			want := make(map[IndexPair]*big.Rat)
			alpha := new(big.Rat).SetFrac(big.NewInt(int64(1+r.IntN(9))), big.NewInt(int64(1+r.IntN(13))))
			want[IndexPair{0, 0}] = big.NewRat(1, 1)
			want[IndexPair{1, 0}] = new(big.Rat).Set(alpha)
			for j := uint32(1); j < 8; j++ {
				n := new(big.Int).Lsh(big.NewInt(int64(1+r.IntN(15))), uint(r.IntN(130)))
				x := new(big.Rat).SetFrac(n, big.NewInt(int64(1+r.IntN(23))))
				if r.IntN(3) != 0 {
					want[IndexPair{0, j}] = x
				}
				if r.IntN(3) != 0 {
					if r.IntN(2) == 0 {
						want[IndexPair{1, j}] = new(big.Rat).Mul(alpha, x)
					} else {
						want[IndexPair{1, j}] = big.NewRat(int64(r.IntN(19)-9), int64(1+r.IntN(17)))
					}
				}
			}
			// Shuffled storage orders exercise descending cancellation and tail repair.
			for i := uint32(0); i < 2; i++ {
				for _, jj := range r.Perm(8) {
					j := uint32(jj)
					x := want[IndexPair{i, j}]
					if x == nil {
						continue
					}
					h.must(h.m.SetString(&p.X, x.RatString()))
					h.must(m.AddNewElement(i, j, &p.X))
				}
			}
			cancellations := 0
			delete(want, IndexPair{1, 0})
			for j := uint32(1); j < 8; j++ {
				x := want[IndexPair{0, j}]
				if x == nil {
					continue
				}
				y := want[IndexPair{1, j}]
				if y == nil {
					y = new(big.Rat)
				}
				v := new(big.Rat).Sub(y, new(big.Rat).Mul(alpha, x))
				if v.Sign() == 0 {
					delete(want, IndexPair{1, j})
					cancellations++
				} else {
					want[IndexPair{1, j}] = v
				}
			}
			if cancellations >= 2 {
				multi++
			}
			var pivot ColumnCell
			for _, c := range m.Columns[0] {
				if c.Row == 1 {
					pivot = c
				}
			}
			nonempty, e := m.PivotRowToRow(0, pivot, 0)
			h.must(e)
			n := 0
			for k := range want {
				if k.I == 1 {
					n++
				}
			}
			if nonempty != (n > 0) {
				t.Fatal("nonempty", nonempty, n)
			}
			lpExactStaticCheck(t, h, m, want)
			for _, v := range m.RowOffsets {
				if v != -1 {
					t.Fatal("workspace dirty")
				}
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if multi < 15 {
		t.Fatal("insufficient multiple cancellation worlds", multi)
	}
}

func TestLPExactStaticMatrixLiteralStorageAndAliases(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("2", "3")
	m := NewRationalStaticMatrix(h.m, 2, 3)
	t.Cleanup(func() { h.must(m.Close()) })
	zero := numeral.NewRational()
	h.must(m.Set(99, 99, &zero)) // Zero before bounds.
	h.must(m.Set(0, 1, &p.X))
	h.must(m.Set(0, 1, &p.Y))
	h.must(m.Element(0, 1, &p.X))
	lpSparseRationalCheck(t, h, &p.X, big.NewRat(2, 1))
	if m.Size() != 2 {
		t.Fatal("deduplicated")
	}
	h.must(m.Clear())
	h.must(m.InitRowColumns(2, 3))
	h.must(m.InitRowOffsets())
	h.must(m.Set(0, 0, &p.X))
	h.must(m.Set(0, 1, &p.Y))
	// Live alpha aliases the first destination: it changes 2->4 before the
	// second term uses it, so the second product is 3*4 rather than 3*2.
	h.must(m.MultiplyRow(0, &m.Rows[0][0].Value))
	lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, big.NewRat(4, 1))
	lpSparseRationalCheck(t, h, &m.Rows[0][1].Value, big.NewRat(12, 1))
	h.must(m.AddNewElement(1, 2, &zero))
	if n, e := m.RowSize(1); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if n, e := m.ColumnSize(2); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	h.must(m.Clear())
	h.must(m.InitEmptyMatrix(1, 4))
	if len(m.RowOffsets) != 0 {
		t.Fatal("repaired old-width offsets")
	}
	h.must(m.AddRow())
	h.must(m.AddColumns(2))
	if m.RowCount() != 2 || m.ColumnCount() != 6 || len(m.RowOffsets) != 2 {
		t.Fatal("shape")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactStaticMatrixAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"set", "new", "element", "remove", "row", "column", "pivot", "pivot-existing"} {
		failures, successes := 0, 0
		for ordinal := int64(0); ordinal < 220; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				out := h.pair("0", "0")
				m := NewRationalStaticMatrix(h.m, 2, 4)
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(m.Close()) })
				if op != "set" && op != "new" {
					h.must(m.Set(0, 0, &p.X))
					h.must(m.Set(0, 1, &p.Y))
					h.must(m.Set(1, 0, &p.Y))
					if op == "remove" {
						h.must(h.m.SetInt64(&m.Rows[0][0].Value, 1))
					}
					if op == "pivot" || op == "pivot-existing" {
						h.must(h.m.SetInt64(&m.Rows[0][0].Value, 1))
						h.must(m.Set(1, 2, &p.X))
						h.must(m.Set(0, 3, &p.X))
						if op == "pivot-existing" {
							h.must(m.Set(1, 1, &p.X))
							h.must(m.Set(1, 3, &p.Y))
						}
					}
				}
				h.budget.FailAfter(ordinal)
				var err error
				switch op {
				case "set":
					err = m.Set(0, 0, &p.X)
				case "new":
					err = m.AddNewElement(0, 0, &p.X)
				case "element":
					err = m.Element(0, 0, &out.X)
				case "remove":
					err = m.RemoveElement(0, 0)
				case "row":
					err = m.MultiplyRow(0, &p.Y)
				case "column":
					err = m.MultiplyColumn(0, &p.Y)
				case "pivot", "pivot-existing":
					_, err = m.PivotRowToRow(0, m.Columns[0][1], 0)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
					x, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
					y, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/13")
					switch op {
					case "set", "new":
						lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, x)
					case "element":
						lpSparseRationalCheck(t, h, &out.X, x)
					case "remove":
						lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, y)
					case "row", "column":
						lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, new(big.Rat).Mul(x, y))
					case "pivot-existing":
						h.must(m.Element(1, 1, &out.X))
						lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Sub(x, new(big.Rat).Mul(y, y)))
						h.must(m.Element(1, 3, &out.X))
						lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Sub(y, new(big.Rat).Mul(y, x)))
					case "pivot":
						h.must(m.Element(1, 1, &out.X))
						lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Neg(new(big.Rat).Mul(y, y)))
						h.must(m.Element(1, 3, &out.X))
						lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Neg(new(big.Rat).Mul(y, x)))
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if op == "set" && (len(m.Columns[0]) != 1 || m.Size() != 0) {
						t.Fatal("set publication prefix")
					}
					if op == "new" && (len(m.Columns[0]) != 0 || m.Size() != 0) {
						t.Fatal("new publication prefix")
					}
					if m.ready() != ErrExactVectorPending {
						t.Fatal("lost stopped owner")
					}
					if e := m.Set(0, 0, &p.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.Clear(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.AddRow(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.AddColumn(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.InitEmptyMatrix(0, 0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.MultiplyRow(0, &p.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.MultiplyColumn(0, &p.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.RemoveElement(0, 0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.AddNewElement(0, 0, &p.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.Element(0, 0, &out.X); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if _, e := m.PivotRowToRow(0, ColumnCell{}, 0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.InitRowColumns(0, 0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.InitRowOffsets(); e != ErrExactVectorPending {
						t.Fatal(e)
					}
					if e := m.AddColumns(0); e != ErrExactVectorPending {
						t.Fatal(e)
					}
				}
				h.must(m.Close())
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.RawInvocationState().Active != [3]bool{} {
					t.Fatal("orphan frame")
				}
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("campaign incomplete", op, failures, successes)
		}
	}
}

func TestLPExactStaticMatrixBoundsAndMutationPrefixes(t *testing.T) {
	for _, op := range []string{"set", "new", "element", "row-size", "column-size", "remove-row", "remove-column", "remove-column-tail", "remove-row-offset", "remove-row-tail", "column-value", "row", "column", "column-link", "init", "pivot-cell", "pivot-zero", "pivot-coefficient-zero", "pivot-workspace", "pivot-invalid", "pivot-self", "pivot-tall"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("2", "3")
			m := NewRationalStaticMatrix(h.m, 3, 3)
			t.Cleanup(func() { h.must(m.Close()) })
			h.must(m.Set(0, 0, &p.X))
			h.must(m.Set(0, 1, &p.Y))
			h.must(m.Set(1, 0, &p.Y))
			want := ErrBounds
			var err error
			switch op {
			case "set":
				err = m.Set(7, 0, &p.X)
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "new":
				err = m.AddNewElement(0, 7, &p.X)
			case "element":
				err = m.Element(7, 0, &p.X)
			case "row-size":
				_, err = m.RowSize(7)
			case "column-size":
				_, err = m.ColumnSize(7)
			case "remove-row":
				err = m.RemoveElement(7, 0)
			case "remove-column":
				m.Rows[0][0].Column = 7
				err = m.RemoveElement(0, 0)
			case "remove-column-tail":
				m.Columns[0][1].Row = 7
				err = m.RemoveElement(0, 0)
				if m.Columns[0][0].Row != 7 {
					t.Fatal("column copy prefix lost")
				}
			case "remove-row-offset":
				m.Columns[0][0].Offset = 7
				err = m.RemoveElement(0, 0)
			case "remove-row-tail":
				m.Rows[0][1].Column = 7
				err = m.RemoveElement(0, 0)
				if m.Rows[0][0].Column != 7 {
					t.Fatal("row copy prefix lost")
				}
				lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, big.NewRat(3, 1))
			case "column-value":
				_, err = m.ColumnValue(ColumnCell{0, 7})
			case "row":
				err = m.MultiplyRow(7, &p.X)
			case "column":
				err = m.MultiplyColumn(7, &p.X)
			case "column-link":
				m.Columns[0][1].Row = 7
				err = m.MultiplyColumn(0, &p.X)
				lpSparseRationalCheck(t, h, &m.Rows[0][0].Value, big.NewRat(4, 1))
			case "init":
				err = m.InitRowColumns(1, 1)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
				if !sourceDebugTests && (m.RowCount() != 4 || m.ColumnCount() != 4) {
					t.Fatal("native append")
				}
			case "pivot-cell":
				_, err = m.PivotRowToRow(0, ColumnCell{1, 7}, 0)
			case "pivot-zero":
				h.must(h.m.SetInt64(&m.Rows[1][0].Value, 0))
				_, err = m.PivotRowToRow(0, ColumnCell{1, 0}, 0)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "pivot-coefficient-zero":
				h.must(h.m.SetInt64(&m.Rows[0][1].Value, 0))
				_, err = m.PivotRowToRow(0, ColumnCell{1, 0}, 0)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "pivot-workspace":
				h.must(m.Set(1, 2, &p.X))
				m.RowOffsets = nil
				_, err = m.PivotRowToRow(0, ColumnCell{1, 0}, 0)
				if len(m.Rows[1]) != 1 || m.Rows[1][0].Column != 2 {
					t.Fatal("pivot removal prefix")
				}
			case "pivot-invalid":
				_, err = m.PivotRowToRow(7, ColumnCell{1, 0}, 0)
				if sourceDebugTests {
					want = ErrAssertion
				} else if len(m.Rows[1]) != 0 {
					t.Fatal("native removal prefix")
				}
			case "pivot-self":
				_, err = m.PivotRowToRow(0, ColumnCell{0, 0}, 0)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			case "pivot-tall":
				m.Columns = m.Columns[:2]
				h.must(m.Set(2, 0, &p.X))
				_, err = m.PivotRowToRow(0, ColumnCell{2, 0}, 0)
				want = nil
				if sourceDebugTests {
					want = ErrAssertion
				}
			}
			if err != want {
				t.Fatal(err, want)
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactStaticMatrixTinyPivotAndBorrowedAliases(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("1", "1/1"+strings.Repeat("0", 330))
	m := NewRationalStaticMatrix(h.m, 2, 2)
	t.Cleanup(func() { h.must(m.Close()) })
	h.must(m.Set(0, 0, &p.X))
	h.must(m.Set(0, 1, &p.Y))
	h.must(m.Set(1, 0, &m.Rows[0][0].Value))
	// Adding to the very same row may relocate its slice: coefficient copies
	// must finish before publication and have independent ownership afterward.
	h.must(m.Set(1, 1, &m.Rows[1][0].Value))
	nonempty, err := m.PivotRowToRow(0, ColumnCell{1, 0}, 0)
	h.must(err)
	if !nonempty {
		t.Fatal("dropped precise nonzero")
	}
	x := new(big.Rat)
	s, e := h.m.String(&p.Y)
	h.must(e)
	x.SetString(s)
	h.must(m.Element(1, 1, &p.X))
	lpSparseRationalCheck(t, h, &p.X, new(big.Rat).Sub(big.NewRat(1, 1), x))
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactStaticMatrixLeavesOlderRawInvocationOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	p := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	q := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	z := h.pair("1", "2")
	m := NewRationalStaticMatrix(h.m, 2, 2)
	t.Cleanup(func() { h.must(m.Close()) })
	h.must(m.Set(0, 0, &z.X))
	h.must(m.Set(0, 1, &z.Y))
	h.must(m.Set(1, 0, &z.X))
	local := numeral.NewRational()
	before, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	h.budget.FailAfter(0)
	err = h.m.Add(&p.X, &q.X, &local)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(err) {
		t.Fatal(err)
	}
	var roots numeral.NumericAllocationRoots
	h.must(roots.After(h.m.Z, before))
	older := h.m.RawInvocationState()
	if !older.Active[1] {
		t.Fatal("older addition not reached", older)
	}
	// Inline coefficients exercise a real completed pivot while the older
	// caller retains its stopped arbitrary-precision addition and scratch.
	_, err = m.PivotRowToRow(0, ColumnCell{1, 0}, 0)
	// This row elimination fills a new slot without reentering the stopped
	// manager addition. It must leave the old receipt untouched.
	h.must(err)
	lpSparseRationalCheck(t, h, &m.Rows[1][0].Value, big.NewRat(-2, 1))
	if m.calls.pending() {
		t.Fatal("adopted older raw frame")
	}
	h.must(m.Close())
	if h.m.RawInvocationState() != older {
		t.Fatal("consumed older frame")
	}
	var generations [3]uint64
	for i, active := range older.Active {
		if active {
			generations[i] = older.Generations[i]
		}
	}
	h.must(h.m.CleanupRawInvocations(&generations))
	h.must(h.m.Del(&local))
	h.must(numeral.CleanupNumericAllocationRoots(h.m.Z, &roots))
	lpExactVectorNamedCensus(t, h, nil)
}
