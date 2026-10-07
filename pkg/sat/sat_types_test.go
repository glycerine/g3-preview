// Licensed under the MIT license.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Port the literal constructors' actual SASSERT(var/sign) obligations from
// src/sat/sat_types.h; the other checks add finite algebraic/rendering coverage.
func TestSourceSATLiteralConstructors(t *testing.T) {
	n := DefaultLiteral()
	if n.variable() != NullBoolVar || n.sign() {
		t.Fatal("source default literal")
	}
	for _, v := range []uint32{0, 1, 31, 1 << 24, uint32(NullBoolVar)} {
		for _, sign := range []bool{false, true} {
			l := MakeLiteral(BoolVar(v), sign)
			if l.variable() != BoolVar(v) || l.sign() != sign {
				t.Fatal("constructor assertion")
			}
			if l.Negated().Negated() != l || l.unsign().sign() || l.unsign().variable() != l.variable() {
				t.Fatal("literal algebra")
			}
			if l.hash() != l.index() || LiteralFromUint(l.index()) != l {
				t.Fatal("encoding/hash")
			}
			mutable := l
			mutable.neg()
			if mutable != l.Negated() {
				t.Fatal("mutating negation")
			}
		}
	}
}
func TestSATLiteralAndModelRendering(t *testing.T) {
	for _, c := range []struct {
		v                uint32
		sign             bool
		internal, dimacs string
	}{{0, false, "0", "1"}, {0, true, "-0", "-1"}, {42, false, "42", "43"}, {42, true, "-42", "-43"}} {
		l := MakeLiteral(BoolVar(c.v), c.sign)
		if l.String() != c.internal || l.dimacsString() != c.dimacs {
			t.Fatalf("render %v", l)
		}
	}
	m := Model{util.LTrue, util.LUndef, util.LFalse, util.LTrue}
	if m.String() != "0 -2 3" {
		t.Fatal(m.String())
	}
	if m.valueAt(MakeLiteral(0, false)) != util.LTrue || m.valueAt(MakeLiteral(0, true)) != util.LFalse || m.valueAt(MakeLiteral(1, true)) != util.LUndef {
		t.Fatal("partial model values")
	}
	if satLiteralsString([]Literal{MakeLiteral(0, true), MakeLiteral(3, false)}) != "-0 3" || satLiteralsString(nil) != "" {
		t.Fatal("literal vector render")
	}
}
func TestLiftedBooleanSourceSemantics(t *testing.T) {
	for _, c := range []struct {
		v               util.LBool
		neg             util.LBool
		printed, status string
	}{{util.LFalse, util.LTrue, "l_false", "unsatisfiable"}, {util.LUndef, util.LUndef, "l_undef", "unknown"}, {util.LTrue, util.LFalse, "l_true", "satisfiable"}} {
		if c.v.Negated() != c.neg || c.v.String() != c.printed || c.v.SATString() != c.status {
			t.Fatalf("value %d", c.v)
		}
	}
	if util.ToLBool(false) != util.LFalse || util.ToLBool(true) != util.LTrue {
		t.Fatal("Boolean conversion")
	}
	if util.LBool(5).String() != "l_undef" || util.LBool(5).SATString() != "unknown" {
		t.Fatal("source default rendering branch")
	}
}
