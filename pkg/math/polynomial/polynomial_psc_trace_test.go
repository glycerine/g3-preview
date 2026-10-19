package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialPSCSourceTraceRecordsAndGating(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 2}}})
	q := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	out, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.monomials.config.Trace = tags
	tags.Enable("psc_chain_classic")
	serviceMust(t, m.PSCChain(p, q, 0, out))
	serviceMust(t, m.tracePSCIteration(m.unit, m.unit, m.unit, 1, 0))
	want := ""
	if tags.ShouldTrace("psc_chain_classic") {
		want = "-------- [psc_chain_classic] psc_chain_optimized_core src/math/polynomial/polynomial.cpp:5752 ---------\nP: x0^2\nQ: x0\n------------------------------------------------\n" +
			"-------- [psc_chain_classic] psc_chain_optimized_core src/math/polynomial/polynomial.cpp:5772 ---------\nA: 1\nB: 1\ns: 1\nd: 1, e: 0\n------------------------------------------------\n"
	}
	if output.String() != want {
		t.Fatalf("PSC source trace records: got %q, want %q", output.String(), want)
	}
	output.Reset()
	p2 := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q2 := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, m.PSCChain(p2, q2, 0, out))
	if tags.ShouldTrace("psc_chain_classic") && (!strings.Contains(output.String(), ":5772 ---------\nA: ") || !strings.Contains(output.String(), "\nd: 1, e: 0\n")) {
		t.Fatal("nonzero remainder did not reach iteration trace", output.String())
	}
	want = output.String()
	tags.Disable("psc_chain_classic")
	unread := Polynomial{owner: m.origin, slot: ^uint32(0), generation: 1}
	serviceMust(t, m.tracePSCEntry(unread, unread))
	if output.String() != want {
		t.Fatal("disabled trace read or emitted operands")
	}
	tags.Enable("psc_chain_classic")
	e = m.tracePSCEntry(unread, unread)
	if tags.ShouldTrace("psc_chain_classic") && e == nil {
		t.Fatal("enabled trace omitted operand formatting")
	}
	if !tags.ShouldTrace("psc_chain_classic") {
		serviceMust(t, e)
	}
}

func TestPolynomialPSCSourceTraceStreamRefusal(t *testing.T) {
	m := polynomialCellFixture(t)
	writer := &algebraicIntervalRefusingWriter{stop: 11, err: errors.New("PSC trace stream")}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	m.monomials.config.Trace = tags
	tags.Enable("psc_chain_classic")
	serviceMust(t, m.tracePSCEntry(m.unit, m.unit))
	if tags.ShouldTrace("psc_chain_classic") {
		if writer.writes != 11 || !strings.HasSuffix(writer.String(), "P: ") {
			t.Fatal("refused PSC trace prefix", writer.writes, writer.String())
		}
	} else if writer.writes != 0 {
		t.Fatal("erased trace wrote to stream")
	}
}
