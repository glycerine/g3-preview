package cmd_context

import "testing"

func TestCommandContextDatatypeBuiltinVisibility(t *testing.T) {
	for _, logic := range []string{"", "ALL", "QF_DT", "QF_FD", "QF_UF", "QF_LIA"} {
		t.Run(logic, func(t *testing.T) {
			c := contextFixture(t)
			if logic != "" {
				contextMust(t, c.SetLogic(logic))
			}
			_, err := c.Manager()
			contextMust(t, err)
			wantIs := logic == "" || logic == "ALL" || logic == "QF_DT" || logic == "QF_FD"
			wantUpdate := logic == "" || logic == "ALL"
			if (len(c.builtins["is"]) != 0) != wantIs || (len(c.builtins["update-field"]) != 0) != wantUpdate {
				t.Fatal("datatype builtin logic gate", logic)
			}
		})
	}
}
