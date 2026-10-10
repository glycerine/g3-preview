package lp

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/numeral"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLPIndexedRationalVectorExactContracts(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("3/7", "-2/11")
	z := h.pair("0", "0")
	v := NewIndexedRationalVector(h.m, 4)
	t.Cleanup(func() { h.must(v.Close()) })
	if v.DataSize() != 4 || v.Size() != 0 || !v.IsOK() {
		t.Fatal(v)
	}
	h.must(v.SetValue(&p.X, 2))
	h.must(v.SetValue(&z.X, 1))
	if !reflect.DeepEqual(v.Index, []uint32{2, 1}) {
		t.Fatal(v.Index)
	}
	err := v.SetValue(&z.X, 2)
	if sourceDebugTests {
		if err != ErrAssertion || v.Size() != 2 {
			t.Fatal(err, v.Index)
		}
	} else if err != nil || v.Size() != 3 {
		t.Fatal(err, v.Index)
	}
	h.must(v.CleanUp())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.SetValue(&p.X, 3))
	v.EraseFromIndex(1)
	v.EraseFromIndex(3)
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.RestoreIndex())
	if !reflect.DeepEqual(v.Index, []uint32{3}) {
		t.Fatal(v.Index)
	}
	h.must(v.Clear())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.SetValue(&p.X, 1))
	h.must(v.Assign(v))
	if !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal("source self-assignment index", v.Index)
	}
	h.must(v.CleanUp())
	if v.Size() != 0 {
		t.Fatal("source self assignment clears data")
	}
	h.must(v.Resize(7))
	if v.DataSize() != 7 {
		t.Fatal(v.DataSize())
	}
	h.must(v.Resize(2))
	if v.DataSize() != 2 {
		t.Fatal(v.DataSize())
	}
	h.must(v.ClearAll())
	// Exact coordinates below the floating drop tolerance must survive.
	p = h.pair("1/100000000000000000000000000000000000000000000000000", "0")
	h.must(v.AddWithDropTolerance(0, &p.X))
	if v.Size() != 1 {
		t.Fatal("exact value was rounded away")
	}
	neg := h.pair("-1/100000000000000000000000000000000000000000000000000", "0")
	h.must(v.AddWithDropTolerance(0, &neg.X))
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.Add(0, &z.X))
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.AddWithDropTolerance(0, &z.X))
	for _, op := range []func() error{
		func() error { return v.SetValue(&p.X, 99) }, func() error { return v.Add(99, &p.X) },
		func() error { return v.AddWithDropTolerance(99, &p.X) },
	} {
		if err := op(); err != ErrBounds {
			t.Fatal(err)
		}
	}
	v.Index = []uint32{0, 99}
	if err := v.Clear(); err != ErrBounds || len(v.Index) != 2 {
		t.Fatal(err, v.Index)
	}
	if err := v.Resize(0); err != ErrBounds || v.DataSize() != 2 {
		t.Fatal(err)
	}
	if err := v.CleanUp(); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Assign(v); err != ErrBounds {
		t.Fatal(err)
	}
	h.must(v.ClearAll())
	y := NewIndexedRationalVector(h.m, 1)
	t.Cleanup(func() { h.must(y.Close()) })
	y.Index = []uint32{99}
	if err := v.Assign(y); err != ErrBounds || v.DataSize() != 1 || !reflect.DeepEqual(v.Index, y.Index) {
		t.Fatal(err, v.Index)
	}
	foreign := newLPPairHarness(t)
	f := NewIndexedRationalVector(foreign.m, 0)
	if err := v.Assign(f); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(f.Close())
}

func TestLPIndexedRationalVectorExactCopyOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	bigValue := h.pair(strings.Repeat("1234567890", 30)+"/7", strings.Repeat("9876543210", 30)+"/11")
	baseline := h.m.Z.Cells.Stats().Live
	v := NewIndexedRationalVector(h.m, 4)
	t.Cleanup(func() { h.must(v.Close()) })
	p := bigValue
	h.must(v.SetValue(&p.X, 2))
	// A source copy constructor copies this unindexed dense slot too.
	h.must(h.m.Copy(&v.Data[0], &bigValue.X))
	clone, err := v.Clone()
	h.must(err)
	t.Cleanup(func() { h.must(clone.Close()) })
	if !reflect.DeepEqual(clone.Index, v.Index) {
		t.Fatal(clone.Index)
	}
	i := 0
	before, err := h.m.String(&v.Data[i])
	h.must(err)
	h.must(v.ClearAll())
	got, err := h.m.String(&clone.Data[i])
	h.must(err)
	if got != before {
		t.Fatal("copy constructor lost unindexed value", got, before)
	}
	dst := NewIndexedRationalVector(h.m, 1)
	t.Cleanup(func() { h.must(dst.Close()) })
	h.must(dst.Assign(clone))
	if dst.DataSize() != 4 || !reflect.DeepEqual(dst.Index, clone.Index) {
		t.Fatal(dst.Index)
	}
	// Only indexed source entries are assigned; its unindexed slot stays zero.
	i = 0
	got, err = h.m.String(&dst.Data[i])
	h.must(err)
	if got != "0" {
		t.Fatal(got)
	}
	h.must(clone.Close())
	h.must(dst.Close())
	h.must(v.Close())
	h.must(v.Close())
	if live := h.m.Z.Cells.Stats().Live; live != baseline {
		t.Fatal("vector roots retained", live, baseline)
	}
}

func TestLPIndexedRationalVectorExactSparseOracle(t *testing.T) {
	h := newLPPairHarness(t)
	var values []*RationalPair
	for i := -4; i <= 4; i++ {
		values = append(values, h.pair(fmt.Sprintf("%d/7", i), fmt.Sprintf("%d/11", 2*i)))
	}
	v := NewIndexedRationalVector(h.m, 13)
	t.Cleanup(func() { h.must(v.Close()) })
	model := make(map[uint32][2]*big.Rat)
	r := rand.New(rand.NewPCG(991, 123))
	for step := 0; step < 2500; step++ {
		j := uint32(r.IntN(13))
		p := values[r.IntN(len(values))]
		px, err := h.m.String(&p.X)
		h.must(err)
		py, err := h.m.String(&p.Y)
		h.must(err)
		x, y := mustLPRat(t, px), mustLPRat(t, py)
		switch r.IntN(6) {
		case 0, 1, 2:
			if step%2 == 0 {
				h.must(v.Add(j, &p.X))
			} else {
				h.must(v.AddWithDropTolerance(j, &p.X))
			}
			old, ok := model[j]
			if !ok {
				old = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			}
			next := [2]*big.Rat{new(big.Rat).Add(old[0], x), new(big.Rat).Add(old[1], y)}
			if next[0].Sign() == 0 && true {
				delete(model, j)
			} else {
				model[j] = next
			}
		case 3:
			h.must(v.CleanUp())
		case 4:
			h.must(v.RestoreIndex())
		case 5:
			if r.IntN(10) == 0 {
				h.must(v.Clear())
				clear(model)
			}
		}
		if v.Size() != uint32(len(model)) {
			t.Fatal(step, v.Index, model)
		}
		seen := make(map[uint32]bool)
		for _, i := range v.Index {
			if seen[i] {
				t.Fatal("duplicate", step, v.Index)
			}
			seen[i] = true
		}
		for i := range v.Data {
			got, err := h.m.String(&v.Data[i])
			h.must(err)
			want, ok := model[uint32(i)]
			if !ok {
				want = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			}
			text := want[0].RatString()
			if got != text {
				t.Fatal(step, i, got, text)
			}
		}
	}
}

func TestLPIndexedRationalVectorAllocationFailureCensus(t *testing.T) {
	for _, op := range []string{"set", "assign", "clone", "add", "drop"} {
		failures, successes := 0, 0
		for fail := int64(0); fail < 80; fail++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, fail), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
				other := h.pair(strings.Repeat("7654321090", 100)+"/13", strings.Repeat("3456789012", 100)+"/17")
				v := NewIndexedRationalVector(h.m, 2)
				src := NewIndexedRationalVector(h.m, 2)
				h.must(src.SetValue(&p.X, 0))
				if op == "add" || op == "drop" {
					h.must(v.SetValue(&other.X, 0))
				}

				h.budget.FailAfter(fail)
				var err error
				var cloned *IndexedRationalVector
				switch op {
				case "set":
					err = v.SetValue(&p.X, 0)
				case "assign":
					err = v.Assign(src)
				case "clone":
					cloned, err = src.Clone()
				case "add":
					err = v.Add(0, &p.X)
				case "drop":
					err = v.AddWithDropTolerance(0, &p.X)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
				} else {
					failures++
				}
				pending := v.calls.pending()
				if op == "clone" {
					pending = cloned.calls.pending()
				}
				if pending {
					target := v
					if op == "clone" {
						target = cloned
					}
					if err := target.Add(0, &p.X); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.AddWithDropTolerance(0, &p.X); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.Clear(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.ClearAll(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.CleanUp(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.RestoreIndex(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.Assign(src); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.SetValue(&p.X, 0); err != ErrExactVectorPending {
						t.Fatal(err)
					}
				}
				if cloned != nil {
					h.must(cloned.Close())
				}
				h.must(v.Close())
				var extra []*numeral.Rational
				for i := range src.Data {
					extra = append(extra, &src.Data[i])
				}
				lpExactVectorNamedCensus(t, h, extra)

				// Source raw manager scratch may have retained physical capacity; ensure
				// invocation metadata itself and concrete vector copies have been consumed.
				if state := h.m.RawInvocationState(); state.Active != [3]bool{} {
					t.Fatal("raw invocation retained", state)
				}
				h.must(src.Close())
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete fault campaign", op, failures, successes)
		}
	}
}

func TestLPIndexedPairVectorExactContracts(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("3/7", "-2/11")
	z := h.pair("0", "0")
	v := NewIndexedPairVector(h.m, 4)
	t.Cleanup(func() { h.must(v.Close()) })
	if v.DataSize() != 4 || v.Size() != 0 || !v.IsOK() {
		t.Fatal(v)
	}
	h.must(v.SetValue(p, 2))
	h.must(v.SetValue(z, 1))
	if !reflect.DeepEqual(v.Index, []uint32{2, 1}) {
		t.Fatal(v.Index)
	}
	err := v.SetValue(z, 2)
	if sourceDebugTests {
		if err != ErrAssertion || v.Size() != 2 {
			t.Fatal(err, v.Index)
		}
	} else if err != nil || v.Size() != 3 {
		t.Fatal(err, v.Index)
	}
	h.must(v.CleanUp())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.SetValue(p, 3))
	v.EraseFromIndex(1)
	v.EraseFromIndex(3)
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.RestoreIndex())
	if !reflect.DeepEqual(v.Index, []uint32{3}) {
		t.Fatal(v.Index)
	}
	h.must(v.Clear())
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.SetValue(p, 1))
	h.must(v.Assign(v))
	if !reflect.DeepEqual(v.Index, []uint32{1}) {
		t.Fatal("source self-assignment index", v.Index)
	}
	h.must(v.CleanUp())
	if v.Size() != 0 {
		t.Fatal("source self assignment clears data")
	}
	h.must(v.Resize(7))
	if v.DataSize() != 7 {
		t.Fatal(v.DataSize())
	}
	h.must(v.Resize(2))
	if v.DataSize() != 2 {
		t.Fatal(v.DataSize())
	}
	h.must(v.ClearAll())
	// Exact coordinates below the floating drop tolerance must survive.
	p = h.pair("1/100000000000000000000000000000000000000000000000000", "0")
	h.must(v.AddWithDropTolerance(0, p))
	if v.Size() != 1 {
		t.Fatal("exact value was rounded away")
	}
	neg := h.pair("-1/100000000000000000000000000000000000000000000000000", "0")
	h.must(v.AddWithDropTolerance(0, neg))
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.Add(0, z))
	if v.Size() != 0 {
		t.Fatal(v.Index)
	}
	h.must(v.AddWithDropTolerance(0, z))
	for _, op := range []func() error{
		func() error { return v.SetValue(p, 99) }, func() error { return v.Add(99, p) },
		func() error { return v.AddWithDropTolerance(99, p) },
	} {
		if err := op(); err != ErrBounds {
			t.Fatal(err)
		}
	}
	v.Index = []uint32{0, 99}
	if err := v.Clear(); err != ErrBounds || len(v.Index) != 2 {
		t.Fatal(err, v.Index)
	}
	if err := v.Resize(0); err != ErrBounds || v.DataSize() != 2 {
		t.Fatal(err)
	}
	if err := v.CleanUp(); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Assign(v); err != ErrBounds {
		t.Fatal(err)
	}
	h.must(v.ClearAll())
	y := NewIndexedPairVector(h.m, 1)
	t.Cleanup(func() { h.must(y.Close()) })
	y.Index = []uint32{99}
	if err := v.Assign(y); err != ErrBounds || v.DataSize() != 1 || !reflect.DeepEqual(v.Index, y.Index) {
		t.Fatal(err, v.Index)
	}
	foreign := newLPPairHarness(t)
	f := NewIndexedPairVector(foreign.m, 0)
	if err := v.Assign(f); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(f.Close())
}

func TestLPIndexedPairVectorExactCopyOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	bigValue := h.pair(strings.Repeat("1234567890", 30)+"/7", strings.Repeat("9876543210", 30)+"/11")
	baseline := h.m.Z.Cells.Stats().Live
	v := NewIndexedPairVector(h.m, 4)
	t.Cleanup(func() { h.must(v.Close()) })
	p := bigValue
	h.must(v.SetValue(p, 2))
	// A source copy constructor copies this unindexed dense slot too.
	h.must(v.Data[0].Copy(h.m, bigValue))
	clone, err := v.Clone()
	h.must(err)
	t.Cleanup(func() { h.must(clone.Close()) })
	if !reflect.DeepEqual(clone.Index, v.Index) {
		t.Fatal(clone.Index)
	}
	i := 0
	before, err := v.Data[i].String(h.m)
	h.must(err)
	h.must(v.ClearAll())
	got, err := clone.Data[i].String(h.m)
	h.must(err)
	if got != before {
		t.Fatal("copy constructor lost unindexed value", got, before)
	}
	dst := NewIndexedPairVector(h.m, 1)
	t.Cleanup(func() { h.must(dst.Close()) })
	h.must(dst.Assign(clone))
	if dst.DataSize() != 4 || !reflect.DeepEqual(dst.Index, clone.Index) {
		t.Fatal(dst.Index)
	}
	// Only indexed source entries are assigned; its unindexed slot stays zero.
	i = 0
	got, err = dst.Data[i].String(h.m)
	h.must(err)
	if got != "(0, 0)" {
		t.Fatal(got)
	}
	h.must(clone.Close())
	h.must(dst.Close())
	h.must(v.Close())
	h.must(v.Close())
	if live := h.m.Z.Cells.Stats().Live; live != baseline {
		t.Fatal("vector roots retained", live, baseline)
	}
}

func TestLPIndexedPairVectorExactSparseOracle(t *testing.T) {
	h := newLPPairHarness(t)
	var values []*RationalPair
	for i := -4; i <= 4; i++ {
		values = append(values, h.pair(fmt.Sprintf("%d/7", i), fmt.Sprintf("%d/11", 2*i)))
	}
	v := NewIndexedPairVector(h.m, 13)
	t.Cleanup(func() { h.must(v.Close()) })
	model := make(map[uint32][2]*big.Rat)
	r := rand.New(rand.NewPCG(991, 123))
	for step := 0; step < 2500; step++ {
		j := uint32(r.IntN(13))
		p := values[r.IntN(len(values))]
		px, err := h.m.String(&p.X)
		h.must(err)
		py, err := h.m.String(&p.Y)
		h.must(err)
		x, y := mustLPRat(t, px), mustLPRat(t, py)
		switch r.IntN(6) {
		case 0, 1, 2:
			if step%2 == 0 {
				h.must(v.Add(j, p))
			} else {
				h.must(v.AddWithDropTolerance(j, p))
			}
			old, ok := model[j]
			if !ok {
				old = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			}
			next := [2]*big.Rat{new(big.Rat).Add(old[0], x), new(big.Rat).Add(old[1], y)}
			if next[0].Sign() == 0 && next[1].Sign() == 0 {
				delete(model, j)
			} else {
				model[j] = next
			}
		case 3:
			h.must(v.CleanUp())
		case 4:
			h.must(v.RestoreIndex())
		case 5:
			if r.IntN(10) == 0 {
				h.must(v.Clear())
				clear(model)
			}
		}
		if v.Size() != uint32(len(model)) {
			t.Fatal(step, v.Index, model)
		}
		seen := make(map[uint32]bool)
		for _, i := range v.Index {
			if seen[i] {
				t.Fatal("duplicate", step, v.Index)
			}
			seen[i] = true
		}
		for i := range v.Data {
			got, err := v.Data[i].String(h.m)
			h.must(err)
			want, ok := model[uint32(i)]
			if !ok {
				want = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			}
			text := "(" + want[0].RatString() + ", " + want[1].RatString() + ")"
			if got != text {
				t.Fatal(step, i, got, text)
			}
		}
	}
}

func TestLPIndexedPairVectorAllocationFailureCensus(t *testing.T) {
	for _, op := range []string{"set", "assign", "clone", "add", "drop"} {
		failures, successes := 0, 0
		for fail := int64(0); fail < 80; fail++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, fail), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
				other := h.pair(strings.Repeat("7654321090", 100)+"/13", strings.Repeat("3456789012", 100)+"/17")
				v := NewIndexedPairVector(h.m, 2)
				src := NewIndexedPairVector(h.m, 2)
				h.must(src.SetValue(p, 0))
				if op == "add" || op == "drop" {
					h.must(v.SetValue(other, 0))
				}

				h.budget.FailAfter(fail)
				var err error
				var cloned *IndexedPairVector
				switch op {
				case "set":
					err = v.SetValue(p, 0)
				case "assign":
					err = v.Assign(src)
				case "clone":
					cloned, err = src.Clone()
				case "add":
					err = v.Add(0, p)
				case "drop":
					err = v.AddWithDropTolerance(0, p)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					successes++
					done = true
				} else {
					failures++
				}
				pending := v.calls.pending()
				if op == "clone" {
					pending = cloned.calls.pending()
				}
				if pending {
					target := v
					if op == "clone" {
						target = cloned
					}
					if err := target.Add(0, p); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.AddWithDropTolerance(0, p); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.Clear(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.ClearAll(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.CleanUp(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.RestoreIndex(); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.Assign(src); err != ErrExactVectorPending {
						t.Fatal(err)
					}
					if err := target.SetValue(p, 0); err != ErrExactVectorPending {
						t.Fatal(err)
					}
				}
				if cloned != nil {
					h.must(cloned.Close())
				}
				h.must(v.Close())
				var extra []*numeral.Rational
				for i := range src.Data {
					extra = append(extra, &src.Data[i].X, &src.Data[i].Y)
				}
				lpExactVectorNamedCensus(t, h, extra)

				// Source raw manager scratch may have retained physical capacity; ensure
				// invocation metadata itself and concrete vector copies have been consumed.
				if state := h.m.RawInvocationState(); state.Active != [3]bool{} {
					t.Fatal("raw invocation retained", state)
				}
				h.must(src.Close())
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 {
			t.Fatal("incomplete fault campaign", op, failures, successes)
		}
	}
}

// Arithmetic may destroy a previous reusable manager scratch cell before a
// failed growth. Count current named owners rather than assuming an unchanged
// global baseline. No slab registry scan is used to infer ownership.
func lpExactVectorNamedCensus(t *testing.T, h *lpPairHarness, extra []*numeral.Rational) {
	t.Helper()
	named := make(map[numeral.NumericCellRef]string)
	direct := uint32(0)
	add := func(ref numeral.NumericCellRef, name string) {
		if ref == 0 || !numeral.ValidNumericRef(h.m.Z.Cells, ref) {
			return
		}
		if previous, ok := named[ref]; ok {
			t.Fatalf("duplicate owning cell %s/%s", previous, name)
		}
		named[ref] = name
		cell := h.m.Z.Cell(ref)
		if cell.Backing.Offset == 0 {
			direct++
		}
		if _, err := h.m.Z.Allocator.ObjectExtent(cell.Backing.Object); err != nil {
			t.Fatal(name, err)
		}
	}
	for i, ref := range h.m.Z.Tmp {
		add(ref, fmt.Sprintf("Tmp/%d", i))
	}
	for i, ref := range h.m.Z.Arg {
		add(ref, fmt.Sprintf("Arg/%d", i))
	}
	for i, p := range []*numeral.Int{&h.m.Z.IntMin, &h.m.Z.Two64, &h.m.NTmp, &h.m.AddTmp1, &h.m.AddTmp2} {
		add(p.Cell, fmt.Sprintf("manager/%d", i))
	}
	rationals := []*numeral.Rational{&h.m.AddMulTmp, &h.m.LtTmp1, &h.m.LtTmp2}
	for _, p := range h.pairs {
		rationals = append(rationals, &p.X, &p.Y)
	}
	rationals = append(rationals, extra...)
	for i, p := range rationals {
		add(p.Num.Cell, fmt.Sprintf("numerator/%d", i))
		add(p.Den.Cell, fmt.Sprintf("denominator/%d", i))
	}
	if live := h.m.Z.Cells.Stats().Live; live != uint32(len(named)) {
		t.Fatal("orphan exact cells", live, len(named), named)
	}
	if stats := h.m.Z.Allocator.Stats(); stats.DirectObjects != direct || stats.PendingBlocks != 0 {
		t.Fatal("orphan direct allocation", stats.DirectObjects, direct, stats.PendingBlocks)
	}
	roots, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	if numeral.NumericAllocationPending(&roots) {
		t.Fatal("pending physical allocation retained", roots)
	}
}

// A stopped arithmetic frame created by another caller retains its own
// generation and allocation roots across vector failure and Close.
func TestLPIndexedRationalVectorOlderRawInvocationIdentity(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	q := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	v := NewIndexedRationalVector(h.m, 1)
	h.must(v.SetValue(&p.X, 0))
	local := numeral.NewRational()
	beforeRoots, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	h.budget.FailAfter(0)
	err = h.m.Add(&p.X, &q.X, &local)
	h.budget.FailAfter(-1)
	if err == nil {
		t.Fatal("older raw call did not fail")
	}
	var olderRoots numeral.NumericAllocationRoots
	h.must(olderRoots.After(h.m.Z, beforeRoots))
	older := h.m.RawInvocationState()
	if !older.Active[1] {
		t.Fatal("missing older addition frame", older)
	}
	if err := v.Add(0, &p.X); err != numeral.ErrRationalRawArithmeticPending {
		t.Fatal(err)
	}
	if v.calls.pending() {
		t.Fatal("vector claimed the older invocation")
	}
	h.must(v.Close())
	if got := h.m.RawInvocationState(); got != older {
		t.Fatal("vector consumed older arithmetic", got, older)
	}
	generations := [3]uint64{}
	for i, active := range older.Active {
		if active {
			generations[i] = older.Generations[i]
		}
	}
	h.must(h.m.CleanupRawInvocations(&generations))
	h.must(h.m.Del(&local))
	h.must(numeral.CleanupNumericAllocationRoots(h.m.Z, &olderRoots))
	lpExactVectorNamedCensus(t, h, nil)
}

// A stopped arithmetic frame created by another caller retains its own
// generation and allocation roots across vector failure and Close.
func TestLPIndexedPairVectorOlderRawInvocationIdentity(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	q := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	v := NewIndexedPairVector(h.m, 1)
	h.must(v.SetValue(p, 0))
	local := numeral.NewRational()
	beforeRoots, err := numeral.CaptureNumericAllocationRoots(h.m.Z)
	h.must(err)
	h.budget.FailAfter(0)
	err = h.m.Add(&p.X, &q.X, &local)
	h.budget.FailAfter(-1)
	if err == nil {
		t.Fatal("older raw call did not fail")
	}
	var olderRoots numeral.NumericAllocationRoots
	h.must(olderRoots.After(h.m.Z, beforeRoots))
	older := h.m.RawInvocationState()
	if !older.Active[1] {
		t.Fatal("missing older addition frame", older)
	}
	if err := v.Add(0, p); err != numeral.ErrRationalRawArithmeticPending {
		t.Fatal(err)
	}
	if v.calls.pending() {
		t.Fatal("vector claimed the older invocation")
	}
	h.must(v.Close())
	if got := h.m.RawInvocationState(); got != older {
		t.Fatal("vector consumed older arithmetic", got, older)
	}
	generations := [3]uint64{}
	for i, active := range older.Active {
		if active {
			generations[i] = older.Generations[i]
		}
	}
	h.must(h.m.CleanupRawInvocations(&generations))
	h.must(h.m.Del(&local))
	h.must(numeral.CleanupNumericAllocationRoots(h.m.Z, &olderRoots))
	lpExactVectorNamedCensus(t, h, nil)
}

// Genuine noninteger operands reach integer GCD/division temporaries during
// rational normalization. Their receipts belong to the vector, alongside the
// outer rational arithmetic frame, even after a completed coordinate prefix.
func TestLPIndexedExactVectorNormalizationFailureOwnership(t *testing.T) {
	integerReceipts := 0
	for _, pair := range []bool{false, true} {
		for _, drop := range []bool{false, true} {
			for _, direct := range []bool{false, true} {
				t.Run(fmt.Sprint(pair, "/", drop, "/", direct), func(t *testing.T) {
					failures, successes := 0, 0
					for ordinal := int64(0); ordinal < 300; ordinal++ {
						done := false
						t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
							var h *lpPairHarness
							if direct {
								h = lpNewDirectSparseHarness(t)
							} else {
								h = newLPPairHarness(t)
							}
							xText, yText := "1"+strings.Repeat("0", 120)+"/7", "2"+strings.Repeat("0", 120)+"/13"
							p, q := h.pair(xText, yText), h.pair(yText, xText)
							v := NewIndexedRationalVector(h.m, 1)
							pv := NewIndexedPairVector(h.m, 1)
							h.must(v.SetValue(&p.X, 0))
							h.must(pv.SetValue(p, 0))
							h.budget.FailAfter(ordinal)
							var err error
							if pair {
								if drop {
									err = pv.AddWithDropTolerance(0, q)
								} else {
									err = pv.Add(0, q)
								}
							} else {
								if drop {
									err = v.AddWithDropTolerance(0, &p.Y)
								} else {
									err = v.Add(0, &p.Y)
								}
							}
							h.budget.FailAfter(-1)
							if err != nil {
								if !lpExactPriorityPhysicalRefusal(err) {
									t.Fatal(err)
								}
								failures++
								calls := &v.calls
								if pair {
									calls = &pv.calls
								}
								if calls.integerDivision.Invocation != 0 || calls.gcd.Invocation != 0 {
									integerReceipts++
								}
							} else {
								successes++
								done = true
								x, _ := new(big.Rat).SetString(xText)
								y, _ := new(big.Rat).SetString(yText)
								want := new(big.Rat).Add(x, y)
								if pair {
									h.check(&pv.Data[0], want, want)
								} else {
									lpSparseRationalCheck(t, h, &v.Data[0], want)
								}
							}
							h.must(v.Close())
							h.must(pv.Close())
							lpExactVectorNamedCensus(t, h, nil)
						})
						if done {
							break
						}
					}
					if successes != 1 || (direct && failures == 0) {
						t.Fatal("incomplete normalization campaign", failures, successes)
					}
				})
			}
		}
	}
	if integerReceipts == 0 {
		t.Fatal("no actual integer normalization receipt")
	}
}
