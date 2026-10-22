package smtcore

import "testing"

func TestSMTCoreTypesFailureFinalCheckAndTheoryVarList(t *testing.T) {
	if SMTNullBoolVar != -1 || SMTTrueBoolVar != 0 || SMTFirstBoolVar != 1 || SMTNullTheoryID != -1 || SMTNullTheoryVar != -1 {
		t.Fatal("source sentinels")
	}
	if SMTFinalCheckDone.String() != "done" || SMTFinalCheckContinue.String() != "continue" || SMTFinalCheckGiveUp.String() != "giveup" || SMTFinalCheckStatus(255).String() != "" {
		t.Fatal("final check display")
	}
	if SMTFailureOK != 0 || SMTFailureQuantifiers != 8 {
		t.Fatal("failure enumeration", SMTFailureOK, SMTFailureQuantifiers)
	}
	if SMTFailureOK.String() != "OK" || SMTFailureMemory.String() != "MEMOUT" || SMTFailureConflictLimit.String() != "NUM_CONFLICTS" || SMTFailure(255).String() != "?" {
		t.Fatal("failure names", SMTFailureOK, SMTFailureMemory, SMTFailureConflictLimit, SMTFailure(255))
	}
	tail := NewSMTTheoryVarList(4, 19, nil)
	head := NewDefaultSMTTheoryVarList()
	if head.TheoryID() != SMTNullTheoryID || head.TheoryVar() != SMTNullTheoryVar || head.Next() != nil {
		t.Fatal("default theory variable node")
	}
	head.SetTheoryID(3)
	head.SetTheoryVar(11)
	head.SetNext(tail)
	if head.TheoryID() != 3 || head.TheoryVar() != 11 || head.Next() != tail || tail.TheoryID() != 4 || tail.TheoryVar() != 19 {
		t.Fatal("theory variable list accessors")
	}
	var absent *SMTTheoryVarList
	if absent.TheoryID() != SMTNullTheoryID || absent.TheoryVar() != SMTNullTheoryVar || absent.Next() != nil {
		t.Fatal("nil theory variable sentinel access")
	}
	absent.SetTheoryID(1)
	absent.SetTheoryVar(1)
	absent.SetNext(head)
}
