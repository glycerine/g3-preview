package solver

import "testing"

func TestSMTLogicsSourceMembership(t *testing.T) {
	rows := []struct {
		name string
		fn   func(string) bool
		yes  []string
		no   []string
	}{
		{"reals-only", LogicHasRealsOnly, []string{"QF_RDL", "QF_LRA", "UFLRA", "LRA", "RDL", "QF_NRA", "QF_UFNRA", "QF_UFLRA"}, []string{"QF_LIA", "ALL"}},
		{"arithmetic", LogicHasArithmetic, []string{"QF_LIA", "QF_FPLRA", "ALL", "HORN"}, []string{"QF_UF", "QF_AX"}},
		{"bit-vector", LogicHasBitVector, []string{"UFBV", "QF_BV", "QF_FPBV", "QF_FD"}, []string{"QF_FP", "QF_LIA"}},
		{"array", LogicHasArray, []string{"QF_AX", "QF_AUFLIA", "AUFBV", "HORN"}, []string{"QF_BV", "QF_UF"}},
		{"sequence", LogicHasSequence, []string{"QF_BVRE", "QF_S", "ALL"}, []string{"QF_BV"}},
		{"string", LogicHasString, []string{"QF_S", "ALL"}, []string{"QF_BVRE"}},
		{"fpa", LogicHasFPA, []string{"QF_FP", "QF_FPBV", "QF_BVFP", "QF_FPLRA", "ALL"}, []string{"QF_BV"}},
		{"uf", LogicHasUF, []string{"QF_UF", "UF", "QF_DT"}, []string{"ALL", "QF_UFBV"}},
		{"horn", LogicHasHorn, []string{"HORN"}, []string{"ALL"}},
		{"pb", LogicHasPB, []string{"QF_FD", "ALL"}, []string{"QF_LIA"}},
		{"fd", LogicHasFD, []string{"QF_FD"}, []string{"ALL"}},
		{"datatype", LogicHasDatatype, []string{"QF_FD", "ALL", "QF_DT"}, []string{"QF_UF"}},
	}
	for _, row := range rows {
		for _, logic := range row.yes {
			if !row.fn(logic) {
				t.Errorf("%s rejected %s", row.name, logic)
			}
		}
		for _, logic := range row.no {
			if row.fn(logic) {
				t.Errorf("%s accepted %s", row.name, logic)
			}
		}
	}
	for _, logic := range []string{"QF_UF", "ALL", "QF_FD", "QF_LIA", "QF_BV", "QF_AX", "QF_S", "HORN", "QF_FP"} {
		if !SupportedLogic(logic) {
			t.Error("supported logic rejected", logic)
		}
	}
	for _, logic := range []string{"", "UNKNOWN", "QF_PB"} {
		if SupportedLogic(logic) {
			t.Error("unsupported logic accepted", logic)
		}
	}
}
