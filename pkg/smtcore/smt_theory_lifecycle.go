// Source: src/smt/smt_context.cpp theory setup, search initialization,
// restart, and round-robin final-check dispatch.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (s *SMTInternalizerState) SetupTheories() error {
	if err := s.open(); err != nil {
		return err
	}
	if s.theoriesSetup {
		return nil
	}
	for _, theory := range s.theoryOrder {
		if err := theory.Setup(); err != nil {
			return err
		}
	}
	s.theoriesSetup = true
	return nil
}

func (s *SMTInternalizerState) InitializeTheorySearch() error {
	if err := s.SetupTheories(); err != nil {
		return err
	}
	for _, theory := range s.theoryOrder {
		if err := theory.InitializeSearch(); err != nil {
			return err
		}
	}
	return s.dynamicAck.schedule.initSearch()
}

func (s *SMTInternalizerState) RestartTheories() error {
	if err := s.open(); err != nil {
		return err
	}
	if s.inconsistent {
		return nil
	}
	for _, theory := range s.theoryOrder {
		if err := theory.Restart(); err != nil {
			return err
		}
	}
	return nil
}

func (s *SMTInternalizerState) FinalCheckTheories() (SMTFinalCheckStatus, error) {
	if err := s.open(); err != nil {
		return SMTFinalCheckGiveUp, err
	}
	s.statistics.NumFinalChecks++
	s.incompleteTheories = s.incompleteTheories[:0]
	count := len(s.theoryOrder)
	if count == 0 {
		return SMTFinalCheckDone, nil
	}
	start := s.finalCheckIndex % count
	result := SMTFinalCheckDone
	for offset := 0; offset < count; offset++ {
		index := (start + offset) % count
		status, err := s.theoryOrder[index].FinalCheck()
		s.finalCheckIndex = (index + 1) % count
		if err != nil {
			return SMTFinalCheckGiveUp, err
		}
		switch status {
		case SMTFinalCheckDone:
		case SMTFinalCheckContinue:
			return status, nil
		case SMTFinalCheckGiveUp:
			result = status
			s.incompleteTheories = append(s.incompleteTheories, s.theoryOrder[index])
		default:
			return SMTFinalCheckGiveUp, ast.ErrASTDomain
		}
	}
	for _, theory := range s.theoryOrder {
		if theory.CanPropagate() {
			return SMTFinalCheckContinue, nil
		}
	}
	return result, nil
}

func (s *SMTInternalizerState) IncompleteTheories() []SMTTheoryInternalizer {
	if s == nil {
		return nil
	}
	return append([]SMTTheoryInternalizer(nil), s.incompleteTheories...)
}
