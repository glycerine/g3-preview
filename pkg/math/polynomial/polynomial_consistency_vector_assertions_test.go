package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Change the actual numeral-manager mode through its public API, rather than
// poisoning a polynomial field: copying coefficient7 in Z5 produces2, exposing
// the source consistency check for a polynomial from the previous Z mode.
func TestPolynomialConsistencyInnerAssertionActions(t *testing.T) {
	for _, action := range []string{"nested-T", "T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := "CCCCCCCCCCCCCCCC"
			if action == "T" {
				choices = "CTT"
			}
			if action == "nested-T" {
				choices = "TT"
			}
			if action == "disabled" {
				choices = ""
			}
			output := polynomialAssertionActionsFixture(t, choices)
			if action == "disabled" && polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			c := polynomialCellFixture(t)
			value := unumeral.NewInt(7)
			p, e := c.MkConstantTake(&value)
			serviceMust(t, e)
			defer func() { serviceMust(t, c.modular.SetZ()); serviceMust(t, c.CleanupValue(&p)) }()
			serviceMust(t, c.modular.SetZPUint64(5))
			sink := &polynomialDisplayTestSink{}
			body := c.DisplayInto(p, sink, false)
			throwing := polynomialSourceDebug && (action == "T" || action == "nested-T")
			if throwing {
				var thrown z3exception.DefaultException
				if !errors.As(body, &thrown) || errors.Is(body, ErrPolynomialAssertion) != (action == "T") || sink.text.Len() != 0 {
					t.Fatal(body, sink.text.String())
				}
			} else if body != nil || sink.text.Len() == 0 {
				t.Fatal(body, sink.text.String())
			}
			if polynomialSourceDebug {
				_, e = output.Seek(0, 0)
				serviceMust(t, e)
				text, e := io.ReadAll(output)
				serviceMust(t, e)
				if action == "disabled" {
					if len(text) != 0 {
						t.Fatal(string(text))
					}
				} else {
					if strings.Contains(string(text), "Line: 7426") {
						t.Fatalf("nested exception re-reported by outer assertion: %q", text)
					}
					if action == "nested-T" {
						if strings.Count(string(text), "ASSERTION VIOLATION") != 1 || !strings.Contains(string(text), "File: src/util/mpzzp.h\nLine: 140") {
							t.Fatal(string(text))
						}
					} else if strings.Count(string(text), "Line: 1854\nm_manager.eq(a, p->a(i))") != 1 {
						t.Fatalf("missing or repeated inner polynomial assertion: %q", text)
					}
				}
			}
		})
	}
}

func TestPolynomialSOMVectorClosedAssertionOwnerPaths(t *testing.T) {
	c := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, v.CloseChecked()) })
	serviceMust(t, v.SetOwner(c))
	child, e := v.At(0)
	serviceMust(t, e)
	polynomialClosedCellAssertionContext(t)
	before := c.integer.Budget().Used()
	owner := v.SetOwner(c)
	got, at := v.At(0)
	if polynomialSourceDebug {
		if !errors.Is(owner, arena.ErrClosed) || !errors.Is(at, arena.ErrClosed) || got != nil {
			t.Fatalf("SetOwner=%v, At=%v, child=%p", owner, at, got)
		}
	} else if owner != nil || at != nil || got != child {
		t.Fatalf("SetOwner=%v, At=%v, child=%p", owner, at, got)
	}
	if v.owner != c || len(v.children) != 1 || v.children[0] != child || v.pending != nil || c.integer.Budget().Used() != before {
		t.Fatal("read-only source assertion gates changed actual vector ownership")
	}
}
