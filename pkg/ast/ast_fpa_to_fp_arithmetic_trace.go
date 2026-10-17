// Source: src/ast/rewriter/fpa_rewriter.cpp:159,168,177,179,195,209,226;
// src/util/trace.h:40-46.
package ast

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func (m *ASTManager) emitFPAToFPTrace(line int, body func(io.Writer) error) error {
	tags := m.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("fp_rewriter") {
		return nil
	}
	err := tags.EmitConditional("fp_rewriter", "mk_to_fp", "src/ast/rewriter/fpa_rewriter.cpp", line,
		func() bool { return true }, body)
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}

func (m *ASTManager) traceFPAToFPRational(line int, label string, r *unumeral.OwnedRational) error {
	return m.emitFPAToFPTrace(line, func(w io.Writer) error {
		if _, e := io.WriteString(w, label); e != nil {
			return e
		}
		value, e := m.numerals.String(r)
		if e != nil {
			return e
		}
		_, e = io.WriteString(w, value+"\n")
		return e
	})
}

func (m *ASTManager) traceFPAToFPArithmeticInput(r *unumeral.OwnedRational) error {
	return m.traceFPAToFPRational(159, "r: ", r)
}

func (m *ASTManager) traceFPAToFPRationalPair(line int, first, second *unumeral.OwnedRational) error {
	return m.emitFPAToFPTrace(line, func(w io.Writer) error {
		if _, e := io.WriteString(w, "r1: "); e != nil {
			return e
		}
		left, e := m.numerals.String(first)
		if e != nil {
			return e
		}
		if _, e = io.WriteString(w, left+", r2: "); e != nil {
			return e
		}
		right, e := m.numerals.String(second)
		if e != nil {
			return e
		}
		_, e = io.WriteString(w, right+"\n")
		return e
	})
}

func (m *ASTManager) traceFPAToFPRawRational(line int, label string, r *unumeral.Rational) error {
	return m.emitFPAToFPTrace(line, func(w io.Writer) error {
		if _, e := io.WriteString(w, label); e != nil {
			return e
		}
		qm, e := m.floatingPointPlugin.numeric.BorrowRationalManager()
		if e != nil {
			return e
		}
		value, e := qm.String(r)
		if e != nil {
			return e
		}
		_, e = io.WriteString(w, value+"\n")
		return e
	})
}

func (m *ASTManager) traceFPAToFPValue(line int, label string, v *unumeral.ArbitraryFloat) error {
	return m.emitFPAToFPTrace(line, func(w io.Writer) error {
		if _, e := io.WriteString(w, label); e != nil {
			return e
		}
		value, e := m.floatingPointPlugin.numeric.ToString(v)
		if e != nil {
			return e
		}
		_, e = io.WriteString(w, value+"\n")
		return e
	})
}
