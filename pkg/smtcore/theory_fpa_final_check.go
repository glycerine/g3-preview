package smtcore

// Source: src/smt/theory_fpa.cpp:716-720.

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (t *SMTFloatingPointTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if t == nil || t.converter == nil {
		return SMTFinalCheckContinue, ast.ErrASTHandle
	}
	if err := t.traceFPAFinalCheck(); err != nil {
		return SMTFinalCheckContinue, err
	}
	if diagnostic.DebugBuild {
		d := diagnostic.BorrowProcessDebugContext()
		enabled, err := diagnostic.SourceAssertionsEnabled(d)
		if err != nil {
			return SMTFinalCheckContinue, err
		}
		if enabled && len(t.converter.ExtraAssertions()) != 0 {
			if err = diagnostic.DebugAssertionFailure(d, "src/smt/theory_fpa.cpp", 718, "m_converter.m_extra_assertions.empty()"); err != nil {
				return SMTFinalCheckContinue, err
			}
		}
	}
	return SMTFinalCheckDone, nil
}
