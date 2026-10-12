//go:build g3debug

package lp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

type lpSwapReportPoint struct {
	start, end, flush                   int
	phase, attempts, applied, completed uint32
	state                               uint32
}
type lpSwapOracle struct {
	text                                       bytes.Buffer
	points                                     []lpSwapReportPoint
	rows                                       [10][10]int
	state, phase, attempts, applied, completed uint32
	failed                                     bool
}

// Independent unsigned RNG recurrence, dense row exchanges and explicit
// permutation-array composition; no production matrix/permutation helpers.
func lpBuildSwapOracle(seed uint32) *lpSwapOracle {
	o := &lpSwapOracle{state: seed}
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			o.rows[i][j] = 99 - 10*i - j
		}
	}
	emit := func(text string) {
		p := lpSwapReportPoint{start: o.text.Len(), flush: len(o.points) + 1, phase: o.phase, attempts: o.attempts, applied: o.applied, completed: o.completed, state: o.state}
		o.text.WriteString(text)
		p.end = o.text.Len()
		o.points = append(o.points, p)
	}
	matrix := func() {
		for _, row := range o.rows {
			line := ""
			for _, v := range row {
				line += fmt.Sprintf("%2d ", v)
			}
			emit(line + "\n")
		}
	}
	next := func() uint32 { o.state = o.state*214013 + 2531011; return (o.state >> 16) & 0x7fff }
	for phase := uint32(1); phase <= 2; phase++ {
		o.phase = phase
		emit("testing swaps\n")
		original := o.rows
		q := [10]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		matrix()
		for attempt := 0; attempt < 100; attempt++ {
			a, b := int(next()%10), int(next()%10)
			o.attempts++
			if a == b {
				continue
			}
			emit(fmt.Sprintf("swap %d %d\n", a, b))
			o.rows[a], o.rows[b] = o.rows[b], o.rows[a]
			if phase == 1 {
				for i, v := range q {
					if v == a {
						q[i] = b
					} else if v == b {
						q[i] = a
					}
				}
			} else {
				q[a], q[b] = q[b], q[a]
			}
			o.applied++
			for i := 0; i < 10; i++ {
				if o.rows[q[i]] != original[i] {
					o.failed = true
					return o
				}
			}
			matrix()
			emit("\n")
		}
		o.completed++
	}
	return o
}

func TestLPDriverSwapsIndependentSourceOracle(t *testing.T) {
	for seed := uint32(0); seed < 50; seed++ {
		o := lpBuildSwapOracle(seed)
		random := z3rand.New(seed)
		out := &lpDriverFlushWriter{}
		d := NewSwapTestDriver(out)
		err := d.Run(random)
		if !o.failed || err != ErrAssertion || out.String() != o.text.String() || out.flushes != len(o.points) || d.Phase != o.phase || d.Attempts != o.attempts || d.Applied != o.applied || d.CompletedPhases != o.completed || d.Product == nil || d.closed {
			t.Fatal("literal source swaps", seed, err, d.Phase, d.Attempts, d.Applied, d.CompletedPhases, o.phase, o.attempts, o.applied, o.completed, out.flushes, len(o.points))
		}
		for i, row := range o.rows {
			for j, want := range row {
				got, e := d.Matrix.Get(uint32(i), uint32(j))
				if e != nil || got != float64(want) {
					t.Fatal("logical source matrix", seed, i, j, got, want, e)
				}
			}
		}
		state := o.state*214013 + 2531011
		if random.Next() != (state>>16)&0x7fff {
			t.Fatal("shared RNG reached draw count", seed)
		}
		// fill_matrix stores every cell, including zero, in descending order;
		// its diagnostic two-argument entries have no reciprocal-link contract.
		for i, row := range d.Matrix.Rows {
			if len(row) != 10 || len(d.Matrix.Columns[i].Values) != 10 {
				t.Fatal("fill lost stored zero", i)
			}
			for k, cell := range row {
				if cell.Index != uint32(9-k) || cell.Value != float64(99-10*i-int(cell.Index)) {
					t.Fatal("physical fill order", i, k, cell)
				}
			}
		}
		if d.Run(random) != ErrExactVectorPending {
			t.Fatal("repeat")
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
		if err = d.Close(); err != nil || d.Product != nil || d.Original != nil || d.Permutation != nil {
			t.Fatal("native lexical cleanup", err)
		}
	}
}

func TestLPDriverSwapsOutputPrefixes(t *testing.T) {
	o := lpBuildSwapOracle(0)
	// Every initial report/matrix byte, then selected report stages spanning
	// both helpers, including pre-swap, post-swap matrix and trailing endl.
	cuts := map[int]lpSwapReportPoint{}
	for _, p := range o.points {
		if p.attempts == 0 || p.phase == 2 || p.attempts <= 2 {
			for _, cut := range []int{p.start, p.start + (p.end-p.start)/2, p.end - 1} {
				cuts[cut] = p
			}
		}
		if p.attempts == 0 {
			for cut := p.start; cut < p.end; cut++ {
				cuts[cut] = p
			}
		}
	}
	for cut, p := range cuts {
		random := z3rand.New(0)
		out := &larConstraintPrefixWriter{remaining: cut, err: io.ErrClosedPipe}
		d := NewSwapTestDriver(out)
		err := d.Run(random)
		if !errors.Is(err, io.ErrClosedPipe) || out.output.String() != o.text.String()[:cut] || d.Phase != p.phase || d.Attempts != p.attempts || d.Applied != p.applied || d.CompletedPhases != p.completed {
			t.Fatal("writer stage", cut, err, d.Phase, p.phase, d.Attempts, p.attempts, d.Applied, p.applied)
		}
		state := p.state*214013 + 2531011
		if random.Next() != (state>>16)&0x7fff {
			t.Fatal("writer RNG prefix", cut)
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range o.points {
		if p.attempts > 2 && p.phase != 2 {
			continue
		}
		random := z3rand.New(0)
		out := &lpDriverFlushWriter{failAt: p.flush}
		d := NewSwapTestDriver(out)
		err := d.Run(random)
		if !errors.Is(err, lpErrResidualWrite) || out.String() != o.text.String()[:p.end] || out.flushes != p.flush || d.Attempts != p.attempts || d.Applied != p.applied || d.Phase != p.phase {
			t.Fatal("endl stage", p.flush, err)
		}
		if err = d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLPDriverSwapSetterAndGuards(t *testing.T) {
	m := newFloatSparseMatrix(3)
	if err := setDriverSwapElement(m, 1, 2, 0); err != nil {
		t.Fatal(err)
	}
	if err := setDriverSwapElement(m, 1, 2, 7); err != nil {
		t.Fatal(err)
	}
	if len(m.Rows[1]) != 1 || len(m.Columns[2].Values) != 1 || m.Rows[1][0].Value != 7 || m.Columns[2].Values[0].Value != 7 {
		t.Fatal("source independent directional overwrite")
	}
	if err := m.SwapRows(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SwapColumns(0, 2); err != nil {
		t.Fatal(err)
	}
	if err := setDriverSwapElement(m, 0, 0, 9); err != nil {
		t.Fatal(err)
	}
	if got, err := m.Get(0, 0); err != nil || got != 9 {
		t.Fatal(got, err)
	}
	for _, indices := range [][2]uint32{{3, 0}, {0, 3}} {
		if err := setDriverSwapElement(m, indices[0], indices[1], 1); err != ErrAssertion {
			t.Fatal("extent", err)
		}
	}
	// Deliberately asymmetric metadata exposes the literal two independent scans.
	m.Columns[2].Values = nil
	if err := setDriverSwapElement(m, 0, 0, 11); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Columns[2].Values, []FloatIndexedValue{{Value: 11, Index: 1}}) {
		t.Fatal(m.Columns[2].Values)
	}
	// A shortened Go backing vector is a reached storage-domain guard after
	// the source dimension assertion and row/column permutation reads.
	m.Columns = nil
	if err := setDriverSwapElement(m, 0, 0, 1); err != ErrBounds {
		t.Fatal("backing extent", err)
	}
	d := NewSwapTestDriver(io.Discard)
	if err := d.Run(nil); err != ErrBounds || d.Matrix != nil {
		t.Fatal("nil RNG", err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.Run(z3rand.New(0)); err != ErrExactVectorPending {
		t.Fatal("closed", err)
	}
}
