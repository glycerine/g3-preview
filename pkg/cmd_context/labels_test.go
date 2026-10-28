package cmd_context

// Source differential: src/cmd_context/basic_cmds.cpp labels_cmd.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestCommandContextWritesRelevantLabelsWithSMT2Quoting(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_UF"))
	m, err := c.Manager()
	contextMust(t, err)
	first, err := m.Symbols.Intern("first")
	contextMust(t, err)
	spaced, err := m.Symbols.Intern("a b")
	contextMust(t, err)
	literal, err := m.MkLabelLit([]usymbol.Symbol{first, spaced})
	contextMust(t, err)
	contextMust(t, c.Assert(literal, ""))
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("label status", status)
	}
	var output bytes.Buffer
	contextMust(t, c.WriteLabels(&output))
	if output.String() != "(labels first |a b|)\n" {
		t.Fatal("label output", output.String())
	}
}
