package subpaving

import (
	"bytes"
	"errors"
	"testing"
)

func TestRationalContextInverseRootPrecisionAndCanceledDispatch(t *testing.T) {
	c, m, value, _, _ := rationalClauseFixture(t)
	a, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	precision, err := m.String(&c.rootPrecision)
	must(err)
	if precision != "1/8192" {
		t.Fatal("source default rational root precision", precision)
	}
	must(m.SetInt32(value, 4))
	must(a.SetFinite(value, value, false, false))
	must(c.XnEqY(out, a, 2))
	var b bytes.Buffer
	must(out.Display(&b))
	if b.String() != "[-2, 2]" {
		t.Fatal("context inverse root dispatch", b.String())
	}
	must(m.SetInt32(value, 2))
	must(a.SetFinite(value, value, false, false))
	c.limit.Limit = c.limit.Steps
	if err := c.XnEqY(out, a, 2); !errors.Is(err, ErrRationalContextCanceled) {
		t.Fatal("context root cancellation transport", err)
	}
	must(c.UpdateRootPrecision(0))
	precision, err = m.String(&c.rootPrecision)
	must(err)
	if precision != "1" {
		t.Fatal("zero root denominator fallback", precision)
	}
}
