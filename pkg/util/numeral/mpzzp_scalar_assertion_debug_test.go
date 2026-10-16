//go:build g3debug

package numeral

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: src/util/mpzzp.h:133-162,188. Assertions precede scalar operations.
func TestIntegerModularScalarSourceAssertionActions(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	a := NewInt(8) // outside the centered [-3, 3] residue interval
	budget := arena.NewBudget(^uint64(0))
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	in, e := os.CreateTemp(t.TempDir(), "mpzzp-assert-input-*")
	serviceMust(t, e)
	_, e = in.WriteString(strings.Repeat("CT", 12) + "TCCCTCTTTTCTTTC")
	serviceMust(t, e)
	_, e = in.Seek(0, 0)
	serviceMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "mpzzp-assert-output-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: out, Stderr: out})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, in)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, e)
	serviceMust(t, d.BindAssertionStreams(streams))
	serviceMust(t, d.BindAssertionActions(actions))
	old, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(old)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, in.Close())
		serviceMust(t, out.Close())
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
	})

	if got, e := m.String(&a); e != nil || got != "8" {
		t.Fatal("Continue must reach raw numeral formatting", got, e)
	}
	var thrown z3exception.DefaultException
	if _, e := m.String(&a); !errors.As(e, &thrown) {
		t.Fatal("Throw must stop before numeral formatting", e)
	}
	b := NewInt(0)
	calls := []func() error{
		func() error { _, e := m.IsZero(&a); return e },
		func() error { _, e := m.IsOne(&a); return e },
		func() error { _, e := m.IsPos(&a); return e },
		func() error { _, e := m.IsNeg(&a); return e },
		func() error { _, e := m.IsNonPos(&a); return e },
		func() error { _, e := m.IsNonNeg(&a); return e },
		func() error { _, e := m.IsMinusOne(&a); return e },
		func() error { _, e := m.Equal(&a, &b); return e },
		func() error { _, e := m.Less(&a, &b); return e },
		func() error { _, e := m.LessEqual(&a, &b); return e },
		func() error { _, e := m.Greater(&a, &b); return e },
		func() error { _, e := m.GreaterEqual(&a, &b); return e },
	}
	for i, call := range calls {
		e := call()
		if i%2 == 0 {
			if e != nil {
				t.Fatal("Continue must reach the raw scalar operation", i, e)
			}
		} else if !errors.As(e, &thrown) {
			t.Fatal("Throw must stop before the raw scalar operation", i, e)
		}
	}
	outValue := NewInt(99)
	if e := m.Add(&a, &b, &outValue); e != nil {
		t.Fatal("Add Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 1)
	outValue = NewInt(99)
	if e := m.Sub(&a, &b, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("Sub Throw must precede output mutation", outValue, e)
	}
	v := NewInt(8)
	if e := m.Inc(&v); e != nil {
		t.Fatal("Inc Continue must reach mutation", e)
	}
	modularValue(t, z, &v, 2)
	v = NewInt(8)
	if e := m.Dec(&v); !errors.As(e, &thrown) || v.Val != 8 {
		t.Fatal("Dec Throw must precede mutation", v, e)
	}
	if e := m.Mul(&a, &b, &outValue); e != nil {
		t.Fatal("Mul Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 0)
	outValue = NewInt(99)
	if e := m.AddMul(&a, &b, &b, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("AddMul Throw must precede output mutation", outValue, e)
	}
	if e := m.SubMul(&a, &b, &b, &outValue); e != nil {
		t.Fatal("SubMul first Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 1)
	outValue = NewInt(99)
	if e := m.SubMul(&b, &a, &b, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("SubMul second Throw must precede output mutation", outValue, e)
	}
	if e := m.SubMul(&b, &b, &a, &outValue); e != nil {
		t.Fatal("SubMul third Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 0)
	if e := m.Swap(&a, &b); !errors.As(e, &thrown) || a.Val != 8 || b.Val != 0 {
		t.Fatal("Swap Throw must precede receiver mutation", a, b, e)
	}
	one := NewInt(1)
	outValue = NewInt(99)
	if e := m.Div(&a, &one, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("Div input Throw must precede divisor copy and output mutation", outValue, e)
	}
	if e := m.Div(&a, &one, &outValue); e != nil {
		t.Fatal("Div input and nested Mul Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 1)
	outValue = NewInt(99)
	if e := m.GCD(&a, &one, &outValue); e != nil {
		t.Fatal("GCD input Continue must reach arithmetic", e)
	}
	modularValue(t, z, &outValue, 1)
	outValue = NewInt(99)
	if e := m.GCD(&a, &a, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("GCD input Throw must precede output mutation", outValue, e)
	}
	if e := m.GCD(&a, &a, &outValue); !errors.As(e, &thrown) || outValue.Val != 8 {
		t.Fatal("GCD result Throw must retain published output", outValue, e)
	}
	if e := m.GCDMany([]Int{NewInt(8), NewInt(12)}, &outValue); !errors.As(e, &thrown) || outValue.Val != 4 {
		t.Fatal("GCDMany result Throw must retain published output", outValue, e)
	}
	x, y, g := NewInt(91), NewInt(92), NewInt(93)
	if e := m.ExtendedGCD(&a, &one, &x, &y, &g); !errors.As(e, &thrown) || x.Val != 91 || y.Val != 92 || g.Val != 93 {
		t.Fatal("ExtendedGCD input Throw must precede output mutation", x, y, g, e)
	}
	zero := NewInt(0)
	if e := m.Inv(&zero); !errors.As(e, &thrown) || zero.Val != 0 {
		t.Fatal("inverse zero Throw must precede arithmetic", zero, e)
	}
	if e := m.Inv(&zero); !errors.As(e, &thrown) {
		t.Fatal("inverse zero Continue must reach final gcd assertion", zero, e)
	}
	serviceMust(t, m.SetZPUint64(6))
	two := NewInt(2)
	if e := m.Inv(&two); !errors.As(e, &thrown) || two.Val != 1 {
		t.Fatal("inverse gcd Throw must follow published swap", two, e)
	}
	serviceMust(t, m.SetPSquare())
	outValue = NewInt(99)
	if e := m.Div(&one, &one, &outValue); !errors.As(e, &thrown) || outValue.Val != 99 {
		t.Fatal("nonprime Div Throw must precede divisor copy", outValue, e)
	}
	if e := m.Div(&one, &one, &outValue); e != nil {
		t.Fatal("nonprime Div Continue must reach inversion and multiplication", e)
	}
	modularValue(t, z, &outValue, 1)
	serviceMust(t, d.EnableAssertions(false))
	if got, e := m.String(&a); e != nil || got != "8" {
		t.Fatal("disabled source assertion must leave numeral readable", got, e)
	}
	if _, e := m.Greater(&a, &b); e != nil {
		t.Fatal("disabled pair assertion must reach comparison", e)
	}
	if e := m.Add(&a, &b, &outValue); e != nil {
		t.Fatal("disabled arithmetic assertion must reach addition", e)
	}
	zero = NewInt(0)
	if e := m.Inv(&zero); e != nil {
		t.Fatal("disabled inverse assertions must not consume an action", e)
	}
	unknown, e := NewIntegerModularManagerWithUint64(z, 7, false)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, unknown.Close()) })
	outValue = NewInt(99)
	if e := unknown.Div(&one, &one, &outValue); e != nil || outValue.Val != 1 {
		t.Fatal("disabled prime assertion must not read the uninitialized flag", outValue, e)
	}
	data, e := os.ReadFile(out.Name())
	serviceMust(t, e)
	if strings.Count(string(data), "File: src/util/mpzzp.h\nLine: 146\n") != 2 ||
		strings.Count(string(data), "is_p_normalized(a)\n") != 14 ||
		strings.Count(string(data), "is_p_normalized(a) && is_p_normalized(b)\n") != 13 ||
		strings.Count(string(data), "is_p_normalized(a) && is_p_normalized(b) && is_p_normalized(c)\n") != 1 ||
		strings.Count(string(data), "Line: 161\nis_p_normalized(b)\n") != 1 ||
		strings.Count(string(data), "Line: 162\nis_p_normalized(c)\n") != 1 {
		t.Fatal("missing source assertion records", string(data))
	}
	for line := 133; line <= 144; line++ {
		if strings.Count(string(data), "Line: "+strconv.Itoa(line)+"\n") != 1 {
			t.Fatal("missing scalar assertion source line", line, string(data))
		}
	}
	for _, line := range []int{150, 151, 152, 153, 154, 156, 160, 161, 162, 188} {
		want := 1
		if line == 154 {
			want = 2
		}
		if strings.Count(string(data), "Line: "+strconv.Itoa(line)+"\n") != want {
			t.Fatal("missing arithmetic assertion source line", line, string(data))
		}
	}
	if strings.Count(string(data), "Line: 201\nis_p_normalized(a)\n") != 2 {
		t.Fatal("missing division input assertion records", string(data))
	}
	if strings.Count(string(data), "Line: 212\nis_p_normalized(a) && is_p_normalized(b)\n") != 3 ||
		strings.Count(string(data), "Line: 214\nis_p_normalized(c)\n") != 1 ||
		strings.Count(string(data), "Line: 219\nis_p_normalized(g)\n") != 1 ||
		strings.Count(string(data), "Line: 223\nis_p_normalized(r1) && is_p_normalized(r2)\n") != 1 {
		t.Fatal("missing GCD source assertion records", string(data))
	}
	if strings.Count(string(data), "Line: 172\n!is_zero(a)\n") != 2 ||
		strings.Count(string(data), "Line: 183\nm().is_one(m_inv_tmp3)\n") != 2 {
		t.Fatal("missing inverse source assertion records", string(data))
	}
	if strings.Count(string(data), "Line: 200\nm_p_prime\n") != 2 {
		t.Fatal("missing prime-flag source assertion records", string(data))
	}
	state, e := input.State()
	serviceMust(t, e)
	if state.Position != 39 {
		t.Fatal("disabled assertion consumed an action", state.Position)
	}
}
