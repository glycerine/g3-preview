package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

type lpExplanationCollector struct {
	values    []*numeral.Rational // borrowed actual row cells, never owning copies
	witnesses []uint32
	failAt    int
	err       error
}

func (c *lpExplanationCollector) Consume(value *numeral.Rational, witness uint32) error {
	c.values = append(c.values, value)
	c.witnesses = append(c.witnesses, witness)
	if c.err != nil && len(c.values)-1 == c.failAt {
		return c.err
	}
	return nil
}

type lpExplanationFixture struct {
	s         *LARSolver
	p         *LARBoundPropagator
	coeff     []*big.Rat
	endpoints []lpRowEndpoint
	target    uint32
}

// This fixture supplies a general exact row to the actual row analyzer and
// explanation entry. It does not claim a canonical simplex tableau or Solve.
func lpNewExplanationFixture(t *testing.T, h *lpPairHarness, world int) *lpExplanationFixture {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	f := &lpExplanationFixture{s: s, coeff: make([]*big.Rat, 4), endpoints: make([]lpRowEndpoint, 4), target: uint32(world % 4)}
	for j := 0; j < 4; j++ {
		_, err = s.AddVar(uint32(31 + 11*j))
		h.must(err)
	}
	h.must(s.Core.RA.AddRow())
	for j := 0; j < 4; j++ {
		c := int64(1 + (world*3+j*5)%13)
		if (world>>uint(j))&1 != 0 {
			c = -c
		}
		f.coeff[j] = big.NewRat(c, int64(2+(world+j)%7))
		h.must(s.Core.RA.Set(0, uint32(j), &h.pair(f.coeff[j].RatString(), "0").X))
		if uint32(j) == f.target {
			continue
		}
		lo, hi := int64(-7+j), int64(3+(world+j)%11)
		ls, us := (world+j)%3 == 0, (world+j)%4 == 0
		if (world+j)%9 == 0 {
			hi, ls, us = lo, false, false
			_, err = s.AddVarBound(uint32(j), LConstraintEQ, &h.pair(fmt.Sprint(lo), "0").X)
			h.must(err)
		} else {
			kind := LConstraintGE
			if ls {
				kind = LConstraintGT
			}
			_, err = s.AddVarBound(uint32(j), kind, &h.pair(fmt.Sprint(lo), "0").X)
			h.must(err)
			kind = LConstraintLE
			if us {
				kind = LConstraintLT
			}
			_, err = s.AddVarBound(uint32(j), kind, &h.pair(fmt.Sprint(hi), "0").X)
			h.must(err)
		}
		f.endpoints[j] = lpRowEndpoint{big.NewRat(lo, 1), big.NewRat(hi, 1), ls, us}
	}
	f.p = s.NewBoundPropagator()
	h.must(s.AnalyzeNewBoundsOnRowTableau(0, f.p))
	lpCheckRowBounds(t, h, f.p, lpRowOracle(f.coeff, f.endpoints, 0))
	return f
}

func TestLARExplanationIndependentConstraintCombination(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 160; world++ {
		f := lpNewExplanationFixture(t, h, world)
		for _, b := range f.p.FoundBounds {
			collector := &lpExplanationCollector{}
			f.p.Evidence = collector
			// Sign inspection and forwarding borrowed coefficients reserve no
			// Runtime storage; the collector uses ordinary Go slice storage.
			h.budget.FailAfter(0)
			err := f.s.ExplainImpliedBound(b, f.p)
			h.budget.FailAfter(-1)
			h.must(err)
			if len(collector.values) != 3 {
				t.Fatal("explanation length", world, len(collector.values))
			}
			sum, strict, out := new(big.Rat), false, 0
			for j := 0; j < 4; j++ {
				if uint32(j) == f.target {
					continue
				}
				if collector.values[out] != &f.s.Core.RA.Rows[0][j].Value {
					t.Fatal("coefficient was copied, normalized or reordered")
				}
				constraint, err := f.s.Constraint(collector.witnesses[out])
				h.must(err)
				vc, ok := constraint.(*LARVarConstraint)
				if !ok || vc.Column != uint32(j) {
					t.Fatal("witness identifies wrong source constraint")
				}
				kind := vc.Kind
				upper := kind == LConstraintLE || kind == LConstraintLT
				needMax := (f.coeff[f.target].Sign() > 0) == b.IsLowBound
				wantUpper := (f.coeff[j].Sign() > 0) == needMax
				if kind != LConstraintEQ && upper != wantUpper {
					t.Fatal("selected endpoint cannot prove target inequality", world, j, kind)
				}
				text, err := h.m.String(&vc.RightSide)
				h.must(err)
				sum.Add(sum, new(big.Rat).Mul(f.coeff[j], mustLPRat(t, text)))
				strict = strict || kind == LConstraintLT || kind == LConstraintGT
				out++
			}
			proof := new(big.Rat).Quo(new(big.Rat).Neg(sum), f.coeff[f.target])
			lpSparseRationalCheck(t, h, &b.Bound, proof)
			if b.Strict != strict {
				t.Fatal("strictness differs from selected constraints")
			}
		}
		h.must(f.p.Close())
		h.must(f.s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARExplanationTermAliasAndOutput(t *testing.T) {
	h := newLPPairHarness(t)
	s, term := lpDispatchModel(t, h, false, "2", "-3")
	p := s.NewBoundPropagator()
	h.must(s.CalculateImpliedBoundsForRow(0, p))
	var output bytes.Buffer
	p.Output = &output
	for _, b := range p.FoundBounds {
		if b.Column != term {
			t.Fatal("term alias missing")
		}
		h.must(s.ExplainImpliedBound(b, p))
	}
	if output.String() != strings.Repeat("doh\n", 4) {
		t.Fatal("source default consume transport", output.String())
	}
	h.must(p.Close())
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARExplanationConsumerRefusalPrefix(t *testing.T) {
	failure := errors.New("evidence refused")
	for failAt := 0; failAt < 3; failAt++ {
		h := newLPPairHarness(t)
		f := lpNewExplanationFixture(t, h, 7)
		c := &lpExplanationCollector{failAt: failAt, err: failure}
		f.p.Evidence = c
		b := f.p.FoundBounds[0]
		if err := f.s.ExplainImpliedBound(b, f.p); !errors.Is(err, failure) || len(c.values) != failAt+1 {
			t.Fatal("consumer failure publication", err, len(c.values))
		}
		raw, division, comparison := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
		if err := f.s.ExplainImpliedBound(b, f.p); !errors.Is(err, ErrExactVectorPending) || len(c.values) != failAt+1 {
			t.Fatal("stopped explanation accepted", err)
		}
		if raw != h.m.RawInvocationState() || division != h.m.DivisionInvocationState() || comparison != h.m.ComparisonInvocationState() || f.s.calls.pending() || f.p.calls.pending() {
			t.Fatal("outer model captured consumer failure")
		}
		h.must(f.s.Close())
		if len(c.witnesses) != failAt+1 || len(f.p.FoundBounds) != 2 {
			t.Fatal("model destroyed borrowed evidence/propagator")
		}
		h.must(f.p.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	h := newLPPairHarness(t)
	f := lpNewExplanationFixture(t, h, 7)
	f.p.Output = lpRefusingWriter{failure}
	if err := f.s.ExplainImpliedBound(f.p.FoundBounds[0], f.p); !errors.Is(err, failure) {
		t.Fatal("default writer refusal", err)
	}
	h.must(f.s.Close())
	h.must(f.p.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARExplanationSourceTermLookupAndSkipOrder(t *testing.T) {
	h := newLPPairHarness(t)
	for _, mode := range []string{"missing-row", "missing-witness", "cell-alias"} {
		s := lpPreparedLARPivot(t, h, "-2", "-3", "1")
		input := NewSparseRationalVector(h.m)
		h.must(input.Append(0, &h.pair("1", "0").X))
		term, err := s.AddTerm(input.Data, &h.pair("0", "0").X)
		h.must(err)
		if _, exists := s.ExternalToColumns[term]; exists {
			t.Fatal("term was already registered")
		}
		target, row := term, uint32(0)
		if mode == "missing-row" {
			row = 99
		}
		if mode == "cell-alias" {
			_, err = s.AddVarBound(2, LConstraintLE, &h.pair("20", "0").X)
			h.must(err)
			target = 0
			// An explicit noncanonical source-row fixture: the explanation's
			// term-ID branch reads IDs independently of matrix column links.
			// Restore the column before destroying that matrix owner.
			s.Core.RA.Rows[0][0].Column = term
		}
		bound, err := NewImpliedBound(h.m, &h.pair("0", "0").X, target, true, true, row, false)
		h.must(err)
		p := s.NewBoundPropagator()
		collector := &lpExplanationCollector{}
		p.Evidence = collector
		err = s.ExplainImpliedBound(bound, p)
		column, exists := s.ExternalToColumns[term]
		if !exists || column != 0 || len(s.ColumnsToExternal) != 3 || len(s.ExternalToColumns) != 4 {
			t.Fatal("operator[] zero publication or directory prefix")
		}
		switch mode {
		case "missing-row":
			if !errors.Is(err, ErrBounds) || len(collector.values) != 0 {
				t.Fatal("map insertion must precede row access", err)
			}
		case "missing-witness":
			if sourceDebugEnabled {
				if !errors.Is(err, ErrAssertion) || len(collector.values) != 1 {
					t.Fatal("invalid-witness assertion prefix", err)
				}
			} else {
				h.must(err)
				if len(collector.values) != 2 || collector.witnesses[1] != InvalidLARIndex {
					t.Fatal("native invalid witness must be forwarded")
				}
			}
		case "cell-alias":
			h.must(err)
			if len(collector.values) != 3 || collector.values[0] != &s.Core.RA.Rows[0][0].Value || collector.witnesses[0] != 0 {
				t.Fatal("skip must precede cell term remapping")
			}
			s.Core.RA.Rows[0][0].Column = 0
		}
		h.must(bound.Close())
		h.must(p.Close())
		h.must(input.Close())
		h.must(s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

type lpExplanationOwningConsumer struct{ owner *lpOwningInterest }

func (c *lpExplanationOwningConsumer) Consume(value *numeral.Rational, witness uint32) error {
	_, err := c.owner.BoundIsInteresting(witness, LConstraintGE, value)
	return err
}

func TestLARExplanationLeavesConsumerDivisionOwned(t *testing.T) {
	for _, direct := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 50 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%v/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f := lpNewExplanationFixture(t, h, 7)
				owner := &lpOwningInterest{h: h, ordinal: ordinal, operands: h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")}
				f.p.Evidence = &lpExplanationOwningConsumer{owner}
				err := f.s.ExplainImpliedBound(f.p.FoundBounds[0], f.p)
				state := h.m.DivisionInvocationState()
				if state.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) || f.s.calls.pending() || f.p.calls.pending() || !owner.calls.pending() {
						t.Fatal("consumer numeric ownership", err)
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				// Hook operands are independently owned by the caller, not row
				// cell borrows. Neither enclosing model nor propagator owns them.
				h.must(f.s.Close())
				h.must(f.p.Close())
				if state.Active && state != h.m.DivisionInvocationState() {
					t.Fatal("enclosing owners consumed hook division")
				}
				h.must(owner.calls.cleanupRaw(h.m))
				h.must(owner.calls.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no physical consumer-owned division refusal", direct)
		}
	}
}

func TestLARExplanationMetadataOwnerAndZeroGuards(t *testing.T) {
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	for _, mode := range []string{"nil-bound", "nil-propagator", "foreign-bound", "foreign-propagator", "row-range", "witness-range", "zero-coefficient"} {
		f := lpNewExplanationFixture(t, h, 7)
		g := lpNewExplanationFixture(t, other, 7)
		bound, bp := f.p.FoundBounds[0], f.p
		collector := &lpExplanationCollector{}
		bp.Evidence = collector
		want := ErrBounds
		switch mode {
		case "nil-bound":
			bound = nil
		case "nil-propagator":
			bp = nil
		case "foreign-bound":
			bound = g.p.FoundBounds[0]
			want = ErrExactVectorOwner
		case "foreign-propagator":
			bp = g.p
			want = ErrExactVectorOwner
		case "row-range":
			bound.RowOrTermIndex = 99
		case "witness-range":
			// Explicit invalid public row metadata, not an allocation failure.
			f.s.Core.RA.Rows[0][0].Column = 99
		case "zero-coefficient":
			// The source inspects the current value and treats is_pos(0) as
			// false. It forwards a stored zero rather than skipping that cell.
			h.must(h.m.Copy(&f.s.Core.RA.Rows[0][0].Value, &h.pair("0", "0").X))
			want = nil
		}
		err := f.s.ExplainImpliedBound(bound, bp)
		if !errors.Is(err, want) {
			t.Fatal(mode, err, want)
		}
		if mode == "zero-coefficient" {
			witness, e := f.s.Bounds.Witnesses.At(0)
			h.must(e)
			expected := witness.LowBoundWitness
			if !bound.IsLowBound {
				expected = witness.UpperBoundWitness
			}
			if len(collector.values) != 3 || collector.witnesses[0] != expected {
				t.Fatal("literal zero coefficient witness", collector.witnesses)
			}
		} else if len(collector.values) != 0 {
			t.Fatal("guard consumed evidence", mode)
		}
		if mode == "witness-range" {
			f.s.Core.RA.Rows[0][0].Column = 0
		}
		h.must(f.s.Close())
		h.must(f.p.Close())
		other.must(g.s.Close())
		other.must(g.p.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
	lpExactVectorNamedCensus(t, other, nil)
}

func TestLARExplanationLeavesPartialBoundOwned(t *testing.T) {
	for _, direct := range []bool{false, true} {
		var h *lpPairHarness
		if direct {
			h = lpNewDirectSparseHarness(t)
		} else {
			h = newLPPairHarness(t)
		}
		f := lpNewExplanationFixture(t, h, 7)
		large := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
		h.budget.FailAfter(0)
		bound, err := NewImpliedBound(h.m, &large.X, 3, true, true, 0, false)
		h.budget.FailAfter(-1)
		if !lpExactPriorityPhysicalRefusal(err) {
			t.Fatal("no actual bound-construction refusal", err)
		}
		raw, div, cmp := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
		if e := f.s.ExplainImpliedBound(bound, f.p); !errors.Is(e, ErrExactVectorPending) || f.s.calls.pending() {
			t.Fatal("partial implied bound accepted/adopted", e)
		}
		h.must(f.s.Close())
		h.must(f.p.Close())
		if raw != h.m.RawInvocationState() || div != h.m.DivisionInvocationState() || cmp != h.m.ComparisonInvocationState() {
			t.Fatal("model consumed separately stopped implied-bound calls")
		}
		h.must(bound.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARExplanationLiteralOriginalRowAfterLUPivot(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpPreparedLARPivot(t, h, "-2", "-3", "-1")
	p := s.NewBoundPropagator()
	h.must(s.CalculateImpliedBoundsForRow(0, p))
	if len(p.FoundBounds) != 2 {
		t.Fatal("negative-basis actual LU row bounds")
	}
	for _, b := range p.FoundBounds {
		want := int64(0)
		if b.IsLowBound {
			want = -47
		}
		lpSparseRationalCheck(t, h, &b.Bound, big.NewRat(want, 1))
		if !b.CoefficientIsPositive || b.Strict != b.IsLowBound {
			t.Fatal("unit pivot iterator evidence")
		}
		c := &lpExplanationCollector{}
		p.Evidence = c
		h.must(s.ExplainImpliedBound(b, p))
		wantWitnesses := []uint32{1, 3}
		if b.IsLowBound {
			wantWitnesses = []uint32{0, 2}
		}
		if len(c.values) != 2 || c.witnesses[0] != wantWitnesses[0] || c.witnesses[1] != wantWitnesses[1] {
			t.Fatal("source original-row witness profile", c.witnesses)
		}
		sum := new(big.Rat)
		for i, witness := range c.witnesses {
			constraint, err := s.Constraint(witness)
			h.must(err)
			coefficient, err := h.m.String(c.values[i])
			h.must(err)
			rhs, err := h.m.String(&constraint.Base().RightSide)
			h.must(err)
			sum.Add(sum, new(big.Rat).Mul(mustLPRat(t, coefficient), mustLPRat(t, rhs)))
		}
		// Original target coefficient is -1, so elimination returns this sum.
		// The source chooses witnesses using the normalized unit pivot sign
		// while consuming original coefficients, yielding the opposite endpoint.
		if sum.Cmp(big.NewRat(want, 1)) == 0 {
			t.Fatal("source explanation unexpectedly proves the implied endpoint")
		}
		if b.IsLowBound && sum.Sign() != 0 || !b.IsLowBound && sum.Cmp(big.NewRat(-47, 1)) != 0 {
			t.Fatal("literal wrong-endpoint proof", sum)
		}
	}
	h.must(s.Close())
	h.must(p.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
