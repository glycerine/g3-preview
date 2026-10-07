package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This is a closing-state protocol fixture, not an injected numeric Del refusal.
// These paths have no source resource/cooperation calls. The callbacks reject
// accidental use; root source fixtures separately bind the actual utilities.
func TestUPolynomialFactorsClosingStateProtocol(t *testing.T) {
	for _, cleared := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial-clear", true: "member-destruction"}[cleared], func(t *testing.T) {
			b := arena.NewBudget(1 << 20)
			z, e := util.NewIntegerManager(util.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}})
			if e != nil {
				t.Fatal(e)
			}
			m, e := NewUPolynomialManager(UPolynomialConfig{Integer: z, Checkpoint: func() bool { panic("unexpected source checkpoint") }, Cooperation: func() error { panic("unexpected source cooperation") }})
			if e != nil {
				t.Fatal(e)
			}
			fs, e := m.NewFactors()
			if e != nil {
				t.Fatal(e)
			}
			p, e := m.NewPolynomial()
			if e != nil {
				t.Fatal(e)
			}
			if e = m.SetInts(p, []util.Int{util.NewInt(1), util.NewInt(1)}); e != nil {
				t.Fatal(e)
			}
			v, e := m.View(p, 0, 2)
			if e != nil {
				t.Fatal(e)
			}
			if e = fs.PushCopy(v, 1); e != nil {
				t.Fatal(e)
			}
			if cleared {
				if e = fs.Clear(); e != nil {
					t.Fatal(e)
				}
			}
			fs.closeCleared = cleared
			fs.closing = true
			before := fs.Stats()
			used := b.Used()
			one := util.NewInt(7)
			for name, e := range map[string]error{"set": fs.SetConstant(&one), "push": fs.PushCopy(v, 2), "clear": fs.Clear()} {
				if !errors.Is(e, arena.ErrClosed) {
					t.Errorf("%s permits mutation while closing: %v", name, e)
				}
			}
			if after := fs.Stats(); after != before || b.Used() != used {
				t.Fatal("closing guards changed actual ownership", before, after, used, b.Used())
			}
			if m.liveFactors != 1 {
				t.Fatal("closing owner lost retain", m.liveFactors)
			}
			if e = fs.Close(); e != nil {
				t.Fatal("cleanup retry", e)
			}
			if m.liveFactors != 0 || fs.Stats().TotalBytes != 0 {
				t.Fatal("completed close retain/header", m.liveFactors, fs.Stats())
			}
			if e = m.DelPolynomial(&p); e != nil {
				t.Fatal(e)
			}
			if e = m.Close(); e != nil {
				t.Fatal(e)
			}
			z.Close()
			if b.Used() != 0 {
				t.Fatal("whole constructed graph", b.Used())
			}
		})
	}
}
