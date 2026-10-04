// Licensed under the MIT license.
package g3

import "testing"

// Port the literal constructors' actual SASSERT(var/sign) obligations from
// src/sat/sat_types.h; the other checks add finite algebraic/rendering coverage.
func TestSourceSATLiteralConstructors(t *testing.T) {
	n := satDefaultLiteral()
	if n.variable() != satNullBoolVar || n.sign() {
		t.Fatal("source default literal")
	}
	for _, v := range []uint32{0, 1, 31, 1 << 24, uint32(satNullBoolVar)} {
		for _, sign := range []bool{false, true} {
			l := satMakeLiteral(satBoolVar(v), sign)
			if l.variable() != satBoolVar(v) || l.sign() != sign {
				t.Fatal("constructor assertion")
			}
			if l.negated().negated() != l || l.unsign().sign() || l.unsign().variable() != l.variable() {
				t.Fatal("literal algebra")
			}
			if l.hash() != l.index() || satLiteralFromUint(l.index()) != l {
				t.Fatal("encoding/hash")
			}
			mutable := l
			mutable.neg()
			if mutable != l.negated() {
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
		l := satMakeLiteral(satBoolVar(c.v), c.sign)
		if l.String() != c.internal || l.dimacsString() != c.dimacs {
			t.Fatalf("render %v", l)
		}
	}
	m := satModel{lTrue, lUndef, lFalse, lTrue}
	if m.String() != "0 -2 3" {
		t.Fatal(m.String())
	}
	if m.valueAt(satMakeLiteral(0, false)) != lTrue || m.valueAt(satMakeLiteral(0, true)) != lFalse || m.valueAt(satMakeLiteral(1, true)) != lUndef {
		t.Fatal("partial model values")
	}
	if satLiteralsString([]satLiteral{satMakeLiteral(0, true), satMakeLiteral(3, false)}) != "-0 3" || satLiteralsString(nil) != "" {
		t.Fatal("literal vector render")
	}
}
func TestLiftedBooleanSourceSemantics(t *testing.T) {
	for _, c := range []struct {
		v               lbool
		neg             lbool
		printed, status string
	}{{lFalse, lTrue, "l_false", "unsatisfiable"}, {lUndef, lUndef, "l_undef", "unknown"}, {lTrue, lFalse, "l_true", "satisfiable"}} {
		if c.v.negated() != c.neg || c.v.String() != c.printed || c.v.satString() != c.status {
			t.Fatalf("value %d", c.v)
		}
	}
	if toLBool(false) != lFalse || toLBool(true) != lTrue {
		t.Fatal("Boolean conversion")
	}
	if lbool(5).String() != "l_undef" || lbool(5).satString() != "unknown" {
		t.Fatal("source default rendering branch")
	}
}
