package lp

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/numeral"
	"math"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func lpPairVectorStrings(t *testing.T, v *StackedPairVector) []string {
	t.Helper()
	out := make([]string, v.Size())
	for i := range out {
		p, err := v.At(uint32(i))
		if err != nil {
			t.Fatal(err)
		}
		out[i], err = p.String(v.manager)
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}
func TestLPStackedPairVectorIndependentSnapshots(t *testing.T) {
	h := newLPPairHarness(t)
	v := NewStackedPairVector(h.m)
	t.Cleanup(func() { h.must(v.Close()) })
	var values []*RationalPair
	for i := 0; i < 21; i++ {
		values = append(values, h.pair(fmt.Sprintf("%d/7", i-10), fmt.Sprintf("%d/11", 10-i)))
	}
	base := h.m.Z.Cells.Stats().Live
	if v.Size() != 0 || len(v.Values()) != 0 {
		t.Fatal(v)
	}
	if _, err := v.At(0); err != ErrBounds {
		t.Fatal(err)
	}
	if err := v.Set(0, values[0]); err != ErrBounds {
		t.Fatal(err)
	}
	for _, k := range []uint32{0, 1, math.MaxUint32} {
		if _, err := v.PeekSize(k); err != ErrScope {
			t.Fatal(err)
		}
		if err := v.Pop(k); err != ErrScope {
			t.Fatal(err)
		}
	}
	// Baseline replacement journal must survive later scope pops.
	h.must(v.Append(values[0]))
	h.must(v.Set(0, values[1]))
	journalSize := len(v.changes)
	h.must(v.Set(0, values[1]))
	if len(v.changes) != journalSize {
		t.Fatal("same value recorded")
	}
	model := lpPairVectorStrings(t, v)
	var snapshots [][]string
	r := rand.New(rand.NewPCG(219, 787))
	for step := 0; step < 1600; step++ {
		switch r.IntN(6) {
		case 0:
			p := values[r.IntN(len(values))]
			h.must(v.Append(p))
			text, err := p.String(h.m)
			h.must(err)
			model = append(model, text)
		case 1:
			if len(model) > 0 {
				i := r.IntN(len(model))
				p := values[r.IntN(len(values))]
				h.must(v.Set(uint32(i), p))
				text, err := p.String(h.m)
				h.must(err)
				model[i] = text
			}
		case 2:
			h.must(v.Push())
			snapshots = append(snapshots, append([]string(nil), model...))
		case 3:
			if len(snapshots) > 0 {
				k := r.IntN(len(snapshots)) + 1
				h.must(v.Pop(uint32(k)))
				model = append([]string(nil), snapshots[len(snapshots)-k]...)
				snapshots = snapshots[:len(snapshots)-k]
			}
		case 4:
			h.must(v.IncreaseSizeByOne())
			model = append(model, "(0, 0)")
		case 5:
			if len(model) > 0 {
				p, err := v.At(0)
				h.must(err)
				h.must(v.Set(0, p))
			}
		}
		if got := lpPairVectorStrings(t, v); !reflect.DeepEqual(got, model) {
			t.Fatal(step, got, model)
		}
		if uint32(len(v.Values())) != v.Size() {
			t.Fatal(step)
		}
		for k := 1; k <= len(snapshots); k++ {
			size, err := v.PeekSize(uint32(k))
			h.must(err)
			if int(size) != len(snapshots[len(snapshots)-k]) {
				t.Fatal(step, k, size)
			}
		}
	}
	h.must(v.Close())
	h.must(v.Close())
	if v.Size() != 0 || h.m.Z.Cells.Stats().Live != base {
		t.Fatal("scoped roots retained", h.m.Z.Cells.Stats().Live, base)
	}
	// Owner can be reused after complete Close.
	h.must(v.Append(values[0]))
	h.must(v.Close())
}

func TestLPStackedPairVectorBigOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair("123456789012345678901234567890/7", "-987654321098765432109876543210/11")
	b := h.pair("1234567890123456789012345678901234567890/13", "76543210987654321098765432109876543210/17")
	want, err := a.String(h.m)
	h.must(err)
	base := h.m.Z.Cells.Stats().Live
	removed := uint32(0)
	if a.X.Num.Cell != 0 {
		removed++
	}
	if a.X.Den.Cell != 0 {
		removed++
	}
	if a.Y.Num.Cell != 0 {
		removed++
	}
	if a.Y.Den.Cell != 0 {
		removed++
	}
	v := NewStackedPairVector(h.m)
	t.Cleanup(func() { h.must(v.Close()) })
	h.must(v.Append(a))
	h.must(v.Push())
	h.must(v.Set(0, b))
	h.must(v.Append(a))
	h.must(v.Set(1, b))
	// Deleting original input must not invalidate either the vector or journal.
	h.must(a.Del(h.m))
	h.must(v.Pop(1))
	if got := lpPairVectorStrings(t, v); !reflect.DeepEqual(got, []string{want}) {
		t.Fatal(got)
	}
	if h.m.Z.Cells.Stats().Live <= base-removed {
		t.Fatal("missing vector roots")
	}
	h.must(v.Close())
	// Manager scratch roots and b remain; the vector and journal do not.
	if live := h.m.Z.Cells.Stats().Live; live != base-removed {
		t.Fatal("journal/vector leaks", live)
	}
}

func TestLPStackedPairVectorAllocationFailureOwnership(t *testing.T) {
	for _, operation := range []string{"append", "replace", "replace-grow", "restore"} {
		failed, succeeded := false, false
		for fail := int64(0); fail < 30; fail++ {
			terminal := false
			t.Run(fmt.Sprintf("%s/%d", operation, fail), func(t *testing.T) {
				h := newLPPairHarness(t)
				bigValue := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
				zero := h.pair("0", "0")
				baseline := h.m.Z.Cells.Stats().Live
				baselineBlocks := h.m.Z.Memory.Stats().LiveBlocks
				v := NewStackedPairVector(h.m)
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(v.Close()) })
				switch operation {
				case "replace":
					h.must(v.Append(bigValue))
				case "replace-grow":
					h.must(v.Append(zero))
					h.must(v.Push())
				case "restore":
					h.must(v.Append(bigValue))
					h.must(v.Push())
					h.must(v.Set(0, zero))
				}
				h.budget.FailAfter(fail)
				var err error
				switch operation {
				case "append":
					err = v.Append(bigValue)
				case "replace":
					err = v.Set(0, zero)
				case "replace-grow":
					err = v.Set(0, bigValue)
				case "restore":
					err = v.Pop(1)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					succeeded = true
					terminal = true
				} else {
					failed = true
				}
				if v.unfinished() {
					// The failed copied object remains a real lexical owner.
					if err := v.Append(zero); err != ErrPairVectorPending {
						t.Fatal(err)
					}
					if err := v.Set(0, zero); err != ErrPairVectorPending {
						t.Fatal(err)
					}
					if err := v.IncreaseSizeByOne(); err != ErrPairVectorPending {
						t.Fatal(err)
					}
					if err := v.Push(); err != ErrPairVectorPending {
						t.Fatal(err)
					}
					if err := v.Pop(1); err != ErrPairVectorPending {
						t.Fatal(err)
					}
				}
				h.must(v.Close())
				if blocks := h.m.Z.Memory.Stats().LiveBlocks; blocks != baselineBlocks {
					t.Fatal("orphan physical roots", operation, fail, blocks, baselineBlocks)
				}
				if live := h.m.Z.Cells.Stats().Live; live != baseline {
					t.Fatal("orphan numeric roots", operation, fail, live, baseline)
				}
			})
			if terminal {
				break
			}
		}
		if !failed || !succeeded {
			t.Fatal("campaign did not reach both outcomes", operation, failed, succeeded)
		}
	}
}

// Another caller may already own a failed allocation in the same confined
// manager. A vector must neither acquire nor release that older root. Likewise,
// consuming its own receipt does not authorize cleanup of a replacement root.
func TestLPStackedPairVectorPendingAllocationIdentity(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprint(replace), func(t *testing.T) {
			h := newLPPairHarness(t)
			bigValue := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
			zero := h.pair("0", "0")
			baseline := h.m.Z.Cells.Stats().Live
			v := NewStackedPairVector(h.m)
			t.Cleanup(func() { h.budget.FailAfter(-1); h.must(v.Close()) })
			h.must(v.Append(bigValue))
			if replace {
				h.budget.FailAfter(1)
				err := v.Set(0, zero)
				h.budget.FailAfter(-1)
				if err == nil || !numeral.NumericAllocationPending(&v.copyRoots) {
					t.Fatal("missing vector receipt", err)
				}
				h.must(h.m.Z.CleanupPendingAllocation())
			}
			local := numeral.NewRational()
			h.budget.FailAfter(1)
			err := h.m.Copy(&local, &bigValue.X)
			h.budget.FailAfter(-1)
			if err == nil || h.m.Z.PendingCell == (numeral.SourceCellBacking{}) {
				t.Fatal("missing older/replacement root", err)
			}
			pending := h.m.Z.PendingCell
			if !replace {
				if err := v.Set(0, zero); err == nil {
					t.Fatal("pending allocator should block copying")
				}
				if numeral.NumericAllocationPending(&v.copyRoots) {
					t.Fatal("claimed older allocation")
				}
			}
			h.must(v.Close())
			if h.m.Z.PendingCell != pending {
				t.Fatal("vector consumed another caller's allocation")
			}
			h.must(h.m.Z.CleanupPendingAllocation())
			h.must(h.m.Del(&local))
			if live := h.m.Z.Cells.Stats().Live; live != baseline {
				t.Fatal("numeric roots retained", live, baseline)
			}
		})
	}
}
