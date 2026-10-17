package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPZBitwiseSourcePreconditionThrowAndContinue(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, location string
		want           int32
		call           func(*IntegerManager, *Int, *Int, *Int) error
	}{
		{"or-first", "Line: 1019\nis_nonneg(a)\n", -1, func(m *IntegerManager, a, b, c *Int) error { return m.BitwiseOr(a, b, c) }},
		{"and-second", "Line: 1070\nis_nonneg(b)\n", 3, func(m *IntegerManager, a, b, c *Int) error { return m.BitwiseAnd(b, a, c) }},
		{"xor-first", "Line: 1107\nis_nonneg(a)\n", -4, func(m *IntegerManager, a, b, c *Int) error { return m.BitwiseXor(a, b, c) }},
		{"not", "Line: 1152\nis_nonneg(a)\n", 0, func(m *IntegerManager, a, b, c *Int) error { return m.BitwiseNot(7, a, c) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, out := bindMPNDebugActions(t, "TC")
			m, _ := integerManager(t)
			a, b, c := NewInt(-1), NewInt(3), NewInt(77)
			if err := tc.call(m, &a, &b, &c); !isMPNVerificationThrow(err) || c != NewInt(77) {
				t.Fatal("source bitwise precondition Throw and unchanged destination", err, c)
			}
			if err := tc.call(m, &a, &b, &c); err != nil || c != NewInt(tc.want) {
				t.Fatal("source bitwise precondition Continue reaches operation", err, c, tc.want)
			}
			contents, err := os.ReadFile(out.Name())
			serviceMust(t, err)
			if got := strings.Count(string(contents), tc.location); got != 2 {
				t.Fatal("source bitwise assertion location", got, string(contents))
			}
		})
	}
}

func TestMPZBitwiseNotChunkTracePrecedesNestedAllocateAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	w := &mpnCloseDebugOnFlush{debug: d, closeAt: 1}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("bitwise_not")
	if !tags.ShouldTrace("bitwise_not") {
		t.Skip("source bitwise_not TRACE erased")
	}
	serviceMust(t, m.BindTrace(tags))
	a, out := NewInt(1), NewInt(77)
	err := m.BitwiseNot(64, &a, &out)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.flushes == 0 || out != NewInt(0) ||
		m.BitwiseNotRaw.Values[3] != NewInt(0) {
		t.Fatal("chunk trace must precede nested allocation assertion and temporary setter", err, w.closeErr, w.flushes, out, m.BitwiseNotRaw.Values[3])
	}
}

func TestMPZBitwisePreconditionSwitchDisabledAndClosed(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	a, b, c := NewInt(-1), NewInt(3), NewInt(77)
	serviceMust(t, d.EnableAssertions(false))
	if err := m.BitwiseOr(&a, &b, &c); err != nil || c != NewInt(-1) {
		t.Fatal("disabled source precondition continues negative inline OR", err, c)
	}
	serviceMust(t, d.Close())
	c = NewInt(77)
	valid := NewInt(1)
	if err := m.BitwiseAnd(&valid, &b, &c); !errors.Is(err, arena.ErrClosed) || c != NewInt(77) {
		t.Fatal("true precondition still reads closed owner before publication", err, c)
	}
}
