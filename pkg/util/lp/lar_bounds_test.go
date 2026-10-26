package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

type lpBoundOracle struct {
	low, upper               *big.Rat
	lowStrict, upperStrict   bool
	lowWitness, upperWitness uint32
}

func (o *lpBoundOracle) columnType() ColumnType {
	if o.low == nil && o.upper == nil {
		return ColumnFree
	}
	if o.low == nil {
		return ColumnUpperBound
	}
	if o.upper == nil {
		return ColumnLowBound
	}
	if o.low.Cmp(o.upper) == 0 && !o.lowStrict && !o.upperStrict {
		return ColumnFixed
	}
	return ColumnBoxed
}

// Intersect exact open/closed intervals; it does not reproduce the source's
// type-dispatch implementation or use production numeral comparisons.
func (o *lpBoundOracle) intersect(kind LConstraintKind, x *big.Rat, ci uint32) (conflict, changed bool) {
	oldType := o.columnType()
	if kind == LConstraintEQ || kind == LConstraintGE || kind == LConstraintGT {
		strict := kind == LConstraintGT
		if o.low == nil || x.Cmp(o.low) > 0 || x.Cmp(o.low) == 0 && strict && !o.lowStrict {
			o.low = new(big.Rat).Set(x)
			o.lowStrict = strict
			o.lowWitness = ci
			changed = true
		}
	}
	if kind == LConstraintEQ || kind == LConstraintLE || kind == LConstraintLT {
		strict := kind == LConstraintLT
		if o.upper == nil || x.Cmp(o.upper) < 0 || x.Cmp(o.upper) == 0 && strict && !o.upperStrict {
			o.upper = new(big.Rat).Set(x)
			o.upperStrict = strict
			o.upperWitness = ci
			changed = true
		}
	}
	if o.low != nil && o.upper != nil {
		c := o.low.Cmp(o.upper)
		conflict = c > 0 || c == 0 && (o.lowStrict || o.upperStrict)
	}
	if !conflict && kind == LConstraintEQ && oldType != ColumnFixed {
		o.lowWitness = ci
		o.upperWitness = ci
	}
	return
}

func TestLARBoundsIndependentIntervalIntersection(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("0", "0")
	r := rand.New(rand.NewPCG(51933, 8701))
	kinds := []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ}
	decisions, conflicts := 0, 0
	for world := 0; world < 300; world++ {
		b := NewLARBounds(h.m)
		h.must(b.AppendFreeColumn(uint32(world)))
		o := lpBoundOracle{lowWitness: InvalidLARIndex, upperWitness: InvalidLARIndex}
		for step := 0; step < 40; step++ {
			x := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(r.IntN(21)-10)), uint(r.IntN(130))), big.NewInt(int64(r.IntN(17)+1)))
			if o.low != nil && r.IntN(3) == 0 {
				x.Set(o.low)
			}
			if o.upper != nil && r.IntN(3) == 0 {
				x.Set(o.upper)
			}
			kind := kinds[r.IntN(len(kinds))]
			ci := uint32(step + 17)
			typeBefore := o.columnType()
			wasUpperConflict := typeBefore == ColumnBoxed && (kind == LConstraintLE || kind == LConstraintLT) && o.low != nil && (x.Cmp(o.low) < 0 || x.Cmp(o.low) == 0 && (kind == LConstraintLT || o.lowStrict))
			conflict, changed := o.intersect(kind, x, ci)
			h.must(b.ChangedColumns.Clear())
			h.must(h.m.SetString(&p.X, x.RatString()))
			err := b.Update(0, kind, &p.X, ci)
			decisions++
			if conflict {
				conflicts++
				if wasUpperConflict && sourceDebugEnabled {
					if err != ErrAssertion || b.InfeasibleColumn.Value != -1 {
						t.Fatal("boxed debug conflict prefix", err, b.InfeasibleColumn.Value)
					}
				} else {
					h.must(err)
					if b.InfeasibleColumn.Value != 0 {
						t.Fatal("missing infeasible column")
					}
				}
				if b.Status.Value != StatusInfeasible {
					t.Fatal("missed independent interval contradiction", world, step)
				}
				break
			}
			h.must(err)
			typeAfter, e := b.Types.At(0)
			h.must(e)
			if typeAfter != o.columnType() || b.Status.Value != StatusOptimal {
				t.Fatal("type/status", world, step, typeAfter, o.columnType(), b.Status.Value)
			}
			if b.ChangedColumns.Contains(0) != changed {
				t.Fatal("changed set", world, step, kind, changed)
			}
			w, e := b.Witnesses.At(0)
			h.must(e)
			if w.LowBoundWitness != o.lowWitness || w.UpperBoundWitness != o.upperWitness || w.Row != uint32(world) {
				t.Fatal("witnesses", world, step, w, o)
			}
			if o.low != nil {
				v, e := b.Low.At(0)
				h.must(e)
				y := big.NewRat(0, 1)
				if o.lowStrict {
					y.SetInt64(1)
				}
				h.check(v, o.low, y)
			}
			if o.upper != nil {
				v, e := b.Upper.At(0)
				h.must(e)
				y := big.NewRat(0, 1)
				if o.upperStrict {
					y.SetInt64(-1)
				}
				h.check(v, o.upper, y)
			}
		}
		h.must(b.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	if decisions < 700 || conflicts < 250 {
		t.Fatal("insufficient independent worlds", decisions, conflicts)
	}
}

func lpSeedLARBounds(t *testing.T, h *lpPairHarness, kind ColumnType, low, upper string) *LARBounds {
	t.Helper()
	b := NewLARBounds(h.m)
	h.must(b.AppendFreeColumn(91))
	lo := h.pair(low, "0")
	up := h.pair(upper, "0")
	if kind == ColumnLowBound || kind == ColumnBoxed || kind == ColumnFixed {
		h.must(b.Update(0, LConstraintGE, &lo.X, 11))
	}
	if kind == ColumnUpperBound || kind == ColumnBoxed {
		h.must(b.Update(0, LConstraintLE, &up.X, 12))
	}
	if kind == ColumnFixed {
		h.must(b.Update(0, LConstraintEQ, &lo.X, 13))
	}
	h.must(b.ChangedColumns.Clear())
	return b
}

func TestLARBoundsSourceConflictPublication(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		typ                      ColumnType
		kind                     LConstraintKind
		rhs                      string
		wantType                 ColumnType
		low, upper               string
		lowWitness, upperWitness uint32
		changed, debugAssert     bool
	}{
		{"upper-lower", ColumnUpperBound, LConstraintGE, "3", ColumnBoxed, "3", "2", 71, 12, true, false},
		{"low-upper", ColumnLowBound, LConstraintLE, "-1", ColumnLowBound, "0", "-1", 11, 71, true, false},
		{"upper-equality", ColumnUpperBound, LConstraintEQ, "3", ColumnUpperBound, "0", "2", 71, 12, false, false},
		{"low-equality", ColumnLowBound, LConstraintEQ, "-1", ColumnLowBound, "0", "0", 11, 71, true, false},
		{"boxed-upper", ColumnBoxed, LConstraintLE, "-1", ColumnBoxed, "0", "-1", 11, 71, true, true},
		{"boxed-lower", ColumnBoxed, LConstraintGE, "3", ColumnBoxed, "3", "2", 71, 12, true, false},
		{"boxed-low-equality", ColumnBoxed, LConstraintEQ, "-1", ColumnBoxed, "0", "2", 11, 71, false, false},
		{"boxed-high-equality", ColumnBoxed, LConstraintEQ, "3", ColumnBoxed, "0", "2", 71, 12, false, false},
		{"fixed-strict-upper", ColumnFixed, LConstraintLT, "0", ColumnFixed, "0", "0", 13, 71, false, false},
		{"fixed-strict-lower", ColumnFixed, LConstraintGT, "0", ColumnFixed, "0", "0", 71, 13, false, false},
		{"fixed-low-equality", ColumnFixed, LConstraintEQ, "-1", ColumnFixed, "0", "0", 13, 71, false, false},
		{"fixed-high-equality", ColumnFixed, LConstraintEQ, "1", ColumnFixed, "0", "0", 71, 13, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newLPPairHarness(t)
			b := lpSeedLARBounds(t, h, tc.typ, "0", "2")
			p := h.pair(tc.rhs, "0")
			err := b.Update(0, tc.kind, &p.X, 71)
			if tc.debugAssert && sourceDebugEnabled {
				if err != ErrAssertion || b.InfeasibleColumn.Value != -1 {
					t.Fatal(err, b.InfeasibleColumn.Value)
				}
			} else {
				h.must(err)
				if b.InfeasibleColumn.Value != 0 {
					t.Fatal("infeasible index")
				}
			}
			if b.Status.Value != StatusInfeasible || b.Types.Values()[0] != tc.wantType || b.ChangedColumns.Contains(0) != tc.changed {
				t.Fatal("state prefix", b.Status.Value, b.Types.Values(), b.ChangedColumns.Indices())
			}
			lo, e := b.Low.At(0)
			h.must(e)
			up, e := b.Upper.At(0)
			h.must(e)
			x, _ := new(big.Rat).SetString(tc.low)
			h.check(lo, x, big.NewRat(0, 1))
			x, _ = new(big.Rat).SetString(tc.upper)
			h.check(up, x, big.NewRat(0, 1))
			w, e := b.Witnesses.At(0)
			h.must(e)
			if w.LowBoundWitness != tc.lowWitness || w.UpperBoundWitness != tc.upperWitness {
				t.Fatal(w)
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARBoundsPhysicalUpdateRefusals(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, typ := range []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed} {
			for _, kind := range []LConstraintKind{LConstraintLE, LConstraintGE, LConstraintEQ} {
				failures, succeeded := 0, false
				for ordinal := int64(0); ordinal < 180; ordinal++ {
					done := false
					t.Run(fmt.Sprintf("direct=%v/%d/%d/%d", direct, typ, kind, ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						low := strings.Repeat("1234567890", 100) + "/7"
						upper := strings.Repeat("9876543210", 100) + "/7"
						b := lpSeedLARBounds(t, h, typ, low, upper)
						p := h.pair(strings.Repeat("4567890123", 100)+"/13", "0")
						h.budget.FailAfter(ordinal)
						err := b.Update(0, kind, &p.X, 71)
						h.budget.FailAfter(-1)
						if err != nil {
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							failures++
							if b.ready() != ErrExactVectorPending {
								t.Fatal("lost stopped bound call")
							}
							if e := b.Update(0, kind, &p.X, 72); e != ErrExactVectorPending {
								t.Fatal(e)
							}
							if e := b.AppendFreeColumn(InvalidLARIndex); e != ErrExactVectorPending {
								t.Fatal(e)
							}
						} else {
							done = true
						}
						h.must(b.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						succeeded = true
						break
					}
				}
				if !succeeded || failures == 0 {
					t.Fatal("incomplete actual refusal campaign", typ, kind, failures, succeeded)
				}
			}
		}
	}
}

func TestLARBoundWitnessStackIndependentSnapshots(t *testing.T) {
	var v StackedLARWitnessVector
	var want []LARBoundWitnesses
	var scopes [][]LARBoundWitnesses
	r := rand.New(rand.NewPCG(7831, 351))
	for step := 0; step < 1600; step++ {
		switch r.IntN(5) {
		case 0:
			w := NewLARBoundWitnesses(r.Uint32())
			v.Append(w)
			want = append(want, w)
		case 1, 2:
			if len(want) > 0 {
				j := r.IntN(len(want))
				w := LARBoundWitnesses{r.Uint32(), r.Uint32(), r.Uint32()}
				if err := v.Set(uint32(j), w); err != nil {
					t.Fatal(err)
				}
				want[j] = w
			}
		case 3:
			v.Push()
			scopes = append(scopes, append([]LARBoundWitnesses(nil), want...))
		case 4:
			if len(scopes) > 0 {
				k := r.IntN(len(scopes)) + 1
				want = append([]LARBoundWitnesses(nil), scopes[len(scopes)-k]...)
				scopes = scopes[:len(scopes)-k]
				if err := v.Pop(uint32(k)); err != nil {
					t.Fatal(err)
				}
			}
		}
		if int(v.Size()) != len(want) || !reflect.DeepEqual(append([]LARBoundWitnesses(nil), v.Values()...), want) {
			t.Fatal("snapshot", step)
		}
		for i, w := range want {
			got, err := v.At(uint32(i))
			if err != nil || got != w {
				t.Fatal(got, err)
			}
			if err := v.Set(uint32(i), w); err != nil {
				t.Fatal(err)
			}
		}
		for k := 1; k <= len(scopes); k++ {
			n, err := v.PeekSize(uint32(k))
			if err != nil || int(n) != len(scopes[len(scopes)-k]) {
				t.Fatal(n, err)
			}
		}
	}
	if _, err := v.At(v.Size()); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Set(v.Size(), LARBoundWitnesses{}); err != ErrBounds {
		t.Fatal(err)
	}
	for _, k := range []uint32{0, uint32(len(scopes) + 1)} {
		if _, err := v.PeekSize(k); err != ErrScope {
			t.Fatal(err)
		}
		if err := v.Pop(k); err != ErrScope {
			t.Fatal(err)
		}
	}
}

func TestLARBoundsUnknownRelationsAndMetadataPrefixes(t *testing.T) {
	for _, typ := range []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) {
			h := newLPPairHarness(t)
			b := lpSeedLARBounds(t, h, typ, "0", "2")
			p := h.pair("1", "0")
			w, _ := b.Witnesses.At(0)
			err := b.Update(0, LConstraintKind(91), &p.X, 71)
			if err != sourceAssertionFailure() {
				t.Fatal(err)
			}
			if b.Types.Values()[0] != typ || b.Status.Value != StatusOptimal || b.InfeasibleColumn.Value != -1 || b.Witnesses.Values()[0] != w {
				t.Fatal("unknown relation modified state")
			}
			if b.ChangedColumns.Contains(0) != (!sourceDebugEnabled && typ == ColumnFree) {
				t.Fatal("free source fallthrough")
			}
			h.must(b.Close())
		})
	}
	for _, mode := range []string{"column", "type", "witness", "changed-domain", "upper-short", "low-short"} {
		t.Run(mode, func(t *testing.T) {
			h := newLPPairHarness(t)
			b := NewLARBounds(h.m)
			h.must(b.AppendFreeColumn(17))
			p := h.pair("3/7", "0")
			j, kind := uint32(0), LConstraintGE
			switch mode {
			case "column":
				j = 1
			case "type":
				h.must(b.Types.Set(0, ColumnType(99)))
			case "witness":
				b.Witnesses = StackedLARWitnessVector{}
			case "changed-domain":
				h.must(b.ChangedColumns.Resize(0))
			case "upper-short":
				h.must(b.Upper.Close())
				kind = LConstraintLE
			case "low-short":
				h.must(b.Low.Close())
			}
			err := b.Update(j, kind, &p.X, 71)
			want := ErrBounds
			if mode == "type" {
				want = sourceAssertionFailure()
			}
			if mode == "upper-short" && sourceDebugEnabled {
				want = ErrAssertion
			}
			if err != want {
				t.Fatal(mode, err, want)
			}
			if mode == "witness" || mode == "changed-domain" {
				lo, e := b.Low.At(0)
				h.must(e)
				h.check(lo, big.NewRat(3, 7), big.NewRat(0, 1))
				if mode == "changed-domain" && b.Witnesses.Values()[0].LowBoundWitness != 71 {
					t.Fatal("witness publication prefix")
				}
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARBoundsExistingScopedOwnersAndAlias(t *testing.T) {
	h := newLPPairHarness(t)
	b := lpSeedLARBounds(t, h, ColumnBoxed, "0", "10")
	p := h.pair("3", "0")
	// Exercise actual scoped components used by LAR push/pop, without claiming
	// matrix/basis/constraint-stack restoration which is not implemented yet.
	b.Status.Push()
	b.Witnesses.Push()
	b.InfeasibleColumn.Push()
	b.Types.Push()
	h.must(b.Low.Push())
	h.must(b.Upper.Push())
	h.must(b.Update(0, LConstraintGE, &p.X, 71))
	b.Status.Push()
	b.Witnesses.Push()
	b.InfeasibleColumn.Push()
	b.Types.Push()
	h.must(b.Low.Push())
	h.must(b.Upper.Push())
	upper, e := b.Upper.At(0)
	h.must(e)
	h.must(b.Update(0, LConstraintEQ, &upper.X, 72))
	if b.Types.Values()[0] != ColumnFixed {
		t.Fatal("aliased upper RHS was lost")
	}
	h.must(h.m.SetInt32(&p.X, 11))
	h.must(b.Update(0, LConstraintGE, &p.X, 73))
	if b.Status.Value != StatusInfeasible {
		t.Fatal("fixed contradiction")
	}
	for k := 0; k < 2; k++ {
		b.Status.Pop(1)
		b.InfeasibleColumn.Pop(1)
		h.must(b.Witnesses.Pop(1))
		h.must(b.Types.Pop(1))
		h.must(b.Low.Pop(1))
		h.must(b.Upper.Pop(1))
		if b.Status.Value != StatusOptimal || b.InfeasibleColumn.Value != -1 || b.Types.Values()[0] != ColumnBoxed {
			t.Fatal("bound scope restoration")
		}
		low, e := b.Low.At(0)
		h.must(e)
		want := int64(3)
		witness := uint32(71)
		if k == 1 {
			want = 0
			witness = 11
		}
		h.check(low, big.NewRat(want, 1), big.NewRat(0, 1))
		w, e := b.Witnesses.At(0)
		h.must(e)
		if w.LowBoundWitness != witness || w.UpperBoundWitness != 12 || w.Row != 91 {
			t.Fatal("scoped witnesses", w)
		}
	}
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARBoundsDebugInvariantProfiles(t *testing.T) {
	for _, mode := range []string{"boxed-equal", "fixed-different", "fixed-low-epsilon", "fixed-upper-epsilon", "free-low-upper-short"} {
		t.Run(mode, func(t *testing.T) {
			h := newLPPairHarness(t)
			typ := ColumnFixed
			if mode == "boxed-equal" {
				typ = ColumnBoxed
			}
			if mode == "free-low-upper-short" {
				typ = ColumnFree
			}
			b := lpSeedLARBounds(t, h, typ, "0", "2")
			p := h.pair("0", "0")
			switch mode {
			case "boxed-equal":
				lo, e := b.Low.At(0)
				h.must(e)
				h.must(b.Upper.Set(0, lo))
			case "fixed-different":
				v := h.pair("1", "0")
				h.must(b.Upper.Set(0, v))
			case "fixed-low-epsilon":
				v := h.pair("0", "1")
				h.must(b.Low.Set(0, v))
				h.must(b.Upper.Set(0, v))
			case "fixed-upper-epsilon":
				// Distinct raw epsilon values cannot be structurally equal. The
				// earlier source equality assertion wins before the zero checks.
				v := h.pair("0", "-1")
				h.must(b.Upper.Set(0, v))
			case "free-low-upper-short":
				h.must(b.Upper.Close())
			}
			err := b.Update(0, LConstraintGE, &p.X, 71)
			if sourceDebugEnabled {
				if err != ErrAssertion {
					t.Fatal(err)
				}
			} else {
				h.must(err)
			}
			h.must(b.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARBoundsMultipleColumnsAndExistingInfeasibility(t *testing.T) {
	h := newLPPairHarness(t)
	b := NewLARBounds(h.m)
	p := h.pair("0", "0")
	for row := uint32(0); row < 4; row++ {
		h.must(b.AppendFreeColumn(row + 30))
	}
	for _, tc := range []struct {
		column uint32
		kind   LConstraintKind
		value  int32
		ci     uint32
	}{
		{2, LConstraintGE, 3, 71}, {2, LConstraintLE, 5, 72}, {2, LConstraintGE, 7, 73},
		{1, LConstraintEQ, 0, 74}, {1, LConstraintLE, -1, 75}, {2, LConstraintLE, 8, 76},
	} {
		h.must(h.m.SetInt32(&p.X, tc.value))
		h.must(b.Update(tc.column, tc.kind, &p.X, tc.ci))
	}
	if b.Status.Value != StatusInfeasible || b.InfeasibleColumn.Value != 1 {
		t.Fatal("source global infeasibility publication")
	}
	if !reflect.DeepEqual(b.ChangedColumns.Indices(), []uint32{2, 1}) {
		t.Fatal("source changed-column order", b.ChangedColumns.Indices())
	}
	w, _ := b.Witnesses.At(2)
	if w.LowBoundWitness != 73 || w.UpperBoundWitness != 72 || w.Row != 32 {
		t.Fatal("weaker bound changed witnesses", w)
	}
	w, _ = b.Witnesses.At(1)
	if w.LowBoundWitness != 74 || w.UpperBoundWitness != 75 || w.Row != 31 {
		t.Fatal("fixed contradiction witness", w)
	}
	w, _ = b.Witnesses.At(0)
	if w != NewLARBoundWitnesses(30) {
		t.Fatal("unrelated column changed", w)
	}
	h.must(b.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
